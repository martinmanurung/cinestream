package usecase

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/martinmanurung/cinestream/internal/domain/movies"
	"github.com/martinmanurung/cinestream/pkg/response"
)

type MovieRepository interface {
	CreateMovie(ctx context.Context, movie *movies.Movie) error
	CreateMovieVideo(ctx context.Context, movieVideo *movies.MovieVideo) error
	FindMovieByID(ctx context.Context, movieID int64) (*movies.Movie, error)
	FindMovieVideoByMovieID(ctx context.Context, movieID int64) (*movies.MovieVideo, error)
	FindAllMovies(ctx context.Context, page, limit int, status string, genre string) ([]movies.MovieListResponse, int64, error)
	FindMovieDetail(ctx context.Context, movieID int64) (*movies.MovieDetailResponse, error)
	UpdateMovie(ctx context.Context, movieID int64, updates map[string]interface{}) error
	UpdateMovieVideo(ctx context.Context, movieID int64, updates map[string]interface{}) error
	DeleteMovie(ctx context.Context, movieID int64) error
	GetHLSURL(ctx context.Context, movieID int64) (string, error)
	// Genre methods
	GetAllGenres(ctx context.Context) ([]movies.Genre, error)
	CreateGenre(ctx context.Context, genre *movies.Genre) error
	DeleteGenre(ctx context.Context, genreID int) error
	AddMovieGenres(ctx context.Context, movieID int64, genreIDs []int) error
	RemoveAllMovieGenres(ctx context.Context, movieID int64) error
	GetMovieGenreIDs(ctx context.Context, movieID int64) ([]int, error)
}

type StorageService interface {
	UploadRawVideo(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader, movieID int64) (string, error)
	GetHLSURL(ctx context.Context, movieID int64) (string, error)
	DeleteRawVideo(ctx context.Context, objectName string) error
	DeleteProcessedVideo(ctx context.Context, movieID int64) error
}

type QueueService interface {
	PublishTranscodingJob(ctx context.Context, movieID int64, rawFilePath string) error
}

type MovieUsecase struct {
	repo           MovieRepository
	storageService StorageService
	queueService   QueueService
}

func NewMovieUsecase(repo MovieRepository, storageService StorageService, queueService QueueService) *MovieUsecase {
	return &MovieUsecase{
		repo:           repo,
		storageService: storageService,
		queueService:   queueService,
	}
}

// UploadMovie handles the complete movie upload process (Admin only)
func (u *MovieUsecase) UploadMovie(ctx context.Context, req movies.UploadMovieRequest, file multipart.File, fileHeader *multipart.FileHeader) (*movies.UploadMovieResponse, error) {
	// 1. Parse release date
	var releaseDate time.Time
	var err error
	if req.ReleaseDate != "" {
		releaseDate, err = time.Parse("2006-01-02", req.ReleaseDate)
		if err != nil {
			return nil, response.NewError(http.StatusBadRequest, "invalid_release_date_format", err)
		}
	}

	// 2. Create movie record in database
	movie := &movies.Movie{
		Title:           req.Title,
		Description:     req.Description,
		ReleaseDate:     releaseDate,
		Director:        req.Director,
		PosterURL:       req.PosterURL,
		TrailerURL:      req.TrailerURL,
		DurationMinutes: req.DurationMinutes,
		Price:           req.Price,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := u.repo.CreateMovie(ctx, movie); err != nil {
		return nil, response.InternalServerError(err)
	}

	// 3. Create movie_video record with PENDING status
	movieVideo := &movies.MovieVideo{
		MovieID:      movie.ID,
		UploadStatus: "PENDING",
		UploadedAt:   time.Now(),
	}

	if err := u.repo.CreateMovieVideo(ctx, movieVideo); err != nil {
		return nil, response.InternalServerError(err)
	}

	// 4. Upload video file to MinIO raw bucket
	rawFilePath, err := u.storageService.UploadRawVideo(ctx, file, fileHeader, movie.ID)
	if err != nil {
		// Update status to FAILED
		u.repo.UpdateMovieVideo(ctx, movie.ID, map[string]interface{}{
			"upload_status": "FAILED",
			"error_message": fmt.Sprintf("Failed to upload file: %v", err),
		})
		return nil, response.InternalServerError(err)
	}

	// 5. Update movie_video with raw_file_path
	if err := u.repo.UpdateMovieVideo(ctx, movie.ID, map[string]interface{}{
		"raw_file_path": rawFilePath,
	}); err != nil {
		return nil, response.InternalServerError(err)
	}

	// 6. Publish transcoding job to Redis queue
	if err := u.queueService.PublishTranscodingJob(ctx, movie.ID, rawFilePath); err != nil {
		// Update status to FAILED
		u.repo.UpdateMovieVideo(ctx, movie.ID, map[string]interface{}{
			"upload_status": "FAILED",
			"error_message": fmt.Sprintf("Failed to queue transcoding job: %v", err),
		})
		return nil, response.InternalServerError(err)
	}

	// 7. Add genres if provided
	if len(req.GenreIDs) > 0 {
		if err := u.repo.AddMovieGenres(ctx, movie.ID, req.GenreIDs); err != nil {
			// Log error but don't fail the upload
			fmt.Printf("Warning: Failed to add genres to movie %d: %v\n", movie.ID, err)
		}
	}

	// 8. Return success response
	return &movies.UploadMovieResponse{
		MovieID: movie.ID,
		Message: "Movie accepted and is now processing",
	}, nil
}

// GetMovieList returns paginated list of movies (Public - only READY movies)
func (u *MovieUsecase) GetMovieList(ctx context.Context, page, limit int, genre string) (*movies.MovieListWithPagination, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 12
	}

	// For public, only show READY movies
	movieList, totalCount, err := u.repo.FindAllMovies(ctx, page, limit, "READY", genre)
	if err != nil {
		return nil, response.InternalServerError(err)
	}

	totalPages := int(totalCount) / limit
	if int(totalCount)%limit != 0 {
		totalPages++
	}

	return &movies.MovieListWithPagination{
		Movies: movieList,
		Pagination: movies.PaginationMeta{
			CurrentPage: page,
			TotalPages:  totalPages,
			TotalItems:  totalCount,
			Limit:       limit,
		},
	}, nil
}

// GetMovieDetail returns detailed information about a movie (Public)
func (u *MovieUsecase) GetMovieDetail(ctx context.Context, movieID int64) (*movies.MovieDetailResponse, error) {
	movieDetail, err := u.repo.FindMovieDetail(ctx, movieID)
	if err != nil {
		return nil, response.InternalServerError(err)
	}

	if movieDetail == nil {
		return nil, response.NewError(http.StatusNotFound, "movie_not_found", nil)
	}

	// Only show READY movies to public
	if movieDetail.UploadStatus != "READY" {
		return nil, response.NewError(http.StatusNotFound, "movie_not_available", nil)
	}

	return movieDetail, nil
}

// UpdateMovie updates movie metadata (Admin only)
func (u *MovieUsecase) UpdateMovie(ctx context.Context, movieID int64, req movies.UpdateMovieRequest) error {
	// Check if movie exists
	movie, err := u.repo.FindMovieByID(ctx, movieID)
	if err != nil {
		return response.InternalServerError(err)
	}
	if movie == nil {
		return response.NewError(http.StatusNotFound, "movie_not_found", nil)
	}

	// Build updates map
	updates := make(map[string]interface{})

	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.ReleaseDate != "" {
		releaseDate, err := time.Parse("2006-01-02", req.ReleaseDate)
		if err != nil {
			return response.NewError(http.StatusBadRequest, "invalid_release_date_format", err)
		}
		updates["release_date"] = releaseDate
	}
	if req.Director != "" {
		updates["director"] = req.Director
	}
	if req.PosterURL != "" {
		updates["poster_url"] = req.PosterURL
	}
	if req.TrailerURL != "" {
		updates["trailer_url"] = req.TrailerURL
	}
	if req.DurationMinutes > 0 {
		updates["duration_minutes"] = req.DurationMinutes
	}
	if req.Price >= 0 {
		updates["price"] = req.Price
	}

	if len(updates) == 0 {
		return response.NewError(http.StatusBadRequest, "no_fields_to_update", nil)
	}

	updates["updated_at"] = time.Now()

	if err := u.repo.UpdateMovie(ctx, movieID, updates); err != nil {
		return response.InternalServerError(err)
	}

	// Update genres if provided
	if len(req.GenreIDs) > 0 {
		// Remove existing genres
		if err := u.repo.RemoveAllMovieGenres(ctx, movieID); err != nil {
			fmt.Printf("Warning: Failed to remove old genres for movie %d: %v\n", movieID, err)
		}
		// Add new genres
		if err := u.repo.AddMovieGenres(ctx, movieID, req.GenreIDs); err != nil {
			fmt.Printf("Warning: Failed to add new genres to movie %d: %v\n", movieID, err)
		}
	}

	return nil
}

// DeleteMovie deletes a movie and its associated files (Admin only)
func (u *MovieUsecase) DeleteMovie(ctx context.Context, movieID int64) error {
	// Check if movie exists
	movie, err := u.repo.FindMovieByID(ctx, movieID)
	if err != nil {
		return response.InternalServerError(err)
	}
	if movie == nil {
		return response.NewError(http.StatusNotFound, "movie_not_found", nil)
	}

	// Get movie_video to delete files
	movieVideo, err := u.repo.FindMovieVideoByMovieID(ctx, movieID)
	if err != nil {
		return response.InternalServerError(err)
	}

	// Delete raw video file from MinIO
	if movieVideo != nil && movieVideo.RawFilePath != "" {
		_ = u.storageService.DeleteRawVideo(ctx, movieVideo.RawFilePath)
	}

	// Delete processed video files from MinIO
	if movieVideo != nil {
		_ = u.storageService.DeleteProcessedVideo(ctx, movieID)
	}

	// Delete movie from database (CASCADE will delete movie_video)
	if err := u.repo.DeleteMovie(ctx, movieID); err != nil {
		return response.InternalServerError(err)
	}

	return nil
}

// GetAllMoviesAdmin returns all movies with any status (Admin only)
func (u *MovieUsecase) GetAllMoviesAdmin(ctx context.Context, page, limit int, status string) (*movies.MovieListWithPagination, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 12
	}

	// Admin can see all statuses
	movieList, totalCount, err := u.repo.FindAllMovies(ctx, page, limit, status, "")
	if err != nil {
		return nil, response.InternalServerError(err)
	}

	totalPages := int(totalCount) / limit
	if int(totalCount)%limit != 0 {
		totalPages++
	}

	return &movies.MovieListWithPagination{
		Movies: movieList,
		Pagination: movies.PaginationMeta{
			CurrentPage: page,
			TotalPages:  totalPages,
			TotalItems:  totalCount,
			Limit:       limit,
		},
	}, nil
}

// Genre management methods

// GetAllGenres returns all available genres
func (u *MovieUsecase) GetAllGenres(ctx context.Context) (*movies.GenreListResponse, error) {
	genres, err := u.repo.GetAllGenres(ctx)
	if err != nil {
		return nil, response.InternalServerError(err)
	}

	return &movies.GenreListResponse{
		Genres: genres,
	}, nil
}

// CreateGenre creates a new genre (Admin only)
func (u *MovieUsecase) CreateGenre(ctx context.Context, req movies.GenreRequest) (*movies.Genre, error) {
	genre := &movies.Genre{
		Name: req.Name,
	}

	if err := u.repo.CreateGenre(ctx, genre); err != nil {
		return nil, response.InternalServerError(err)
	}

	return genre, nil
}

// DeleteGenre deletes a genre (Admin only)
func (u *MovieUsecase) DeleteGenre(ctx context.Context, genreID int) error {
	if err := u.repo.DeleteGenre(ctx, genreID); err != nil {
		return response.InternalServerError(err)
	}

	return nil
}
