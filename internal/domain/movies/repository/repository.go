package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/martinmanurung/cinestream/internal/domain/movies"
	"gorm.io/gorm"
)

type MovieRepository struct {
	db *gorm.DB
}

func NewMovieRepository(db *gorm.DB) *MovieRepository {
	return &MovieRepository{db: db}
}

// CreateMovie creates a new movie record
func (r *MovieRepository) CreateMovie(ctx context.Context, movie *movies.Movie) error {
	return r.db.WithContext(ctx).Create(movie).Error
}

// CreateMovieVideo creates a movie_video record
func (r *MovieRepository) CreateMovieVideo(ctx context.Context, movieVideo *movies.MovieVideo) error {
	return r.db.WithContext(ctx).Create(movieVideo).Error
}

// FindMovieByID finds a movie by its ID
func (r *MovieRepository) FindMovieByID(ctx context.Context, movieID int64) (*movies.Movie, error) {
	var movie movies.Movie
	err := r.db.WithContext(ctx).Where("id = ?", movieID).First(&movie).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &movie, nil
}

// FindMovieVideoByMovieID finds movie_video record by movie_id
func (r *MovieRepository) FindMovieVideoByMovieID(ctx context.Context, movieID int64) (*movies.MovieVideo, error) {
	var movieVideo movies.MovieVideo
	err := r.db.WithContext(ctx).Where("movie_id = ?", movieID).First(&movieVideo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &movieVideo, nil
}

// FindAllMovies returns paginated list of movies with optional filters
func (r *MovieRepository) FindAllMovies(ctx context.Context, page, limit int, status string, genre string) ([]movies.MovieListResponse, int64, error) {
	var results []movies.MovieListResponse
	var totalCount int64

	offset := (page - 1) * limit

	// Base query with JOIN to movie_videos
	query := r.db.WithContext(ctx).
		Table("movies").
		Select("movies.id, movies.title, movies.poster_url, movies.price, movies.duration_minutes, COALESCE(movie_videos.upload_status, 'PENDING') as upload_status").
		Joins("LEFT JOIN movie_videos ON movie_videos.movie_id = movies.id")

	// Apply status filter if provided
	if status != "" {
		query = query.Where("movie_videos.upload_status = ?", status)
	} else {
		// By default, only show READY movies for public
		query = query.Where("movie_videos.upload_status = ?", "READY")
	}

	// Apply genre filter if provided
	if genre != "" {
		query = query.Joins("JOIN movie_genres ON movie_genres.movie_id = movies.id").
			Joins("JOIN genres ON genres.id = movie_genres.genre_id").
			Where("genres.name = ?", genre)
	}

	// Count total records
	countQuery := query
	if err := countQuery.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	if err := query.Offset(offset).Limit(limit).Order("movies.created_at DESC").Find(&results).Error; err != nil {
		return nil, 0, err
	}

	return results, totalCount, nil
}

// FindMovieDetail returns detailed information about a movie
func (r *MovieRepository) FindMovieDetail(ctx context.Context, movieID int64) (*movies.MovieDetailResponse, error) {
	var result movies.MovieDetailResponse

	err := r.db.WithContext(ctx).
		Table("movies").
		Select("movies.*, COALESCE(movie_videos.upload_status, 'PENDING') as upload_status").
		Joins("LEFT JOIN movie_videos ON movie_videos.movie_id = movies.id").
		Where("movies.id = ?", movieID).
		First(&result).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	// Format release_date
	var movie movies.Movie
	if err := r.db.WithContext(ctx).Where("id = ?", movieID).First(&movie).Error; err == nil {
		result.ReleaseDate = movie.ReleaseDate.Format("2006-01-02")
	}

	// Get genres
	result.Genres = r.getMovieGenres(ctx, movieID)

	return &result, nil
}

// UpdateMovie updates movie metadata
func (r *MovieRepository) UpdateMovie(ctx context.Context, movieID int64, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).Model(&movies.Movie{}).Where("id = ?", movieID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("movie with id %d not found", movieID)
	}
	return nil
}

// UpdateMovieVideo updates movie_video record
func (r *MovieRepository) UpdateMovieVideo(ctx context.Context, movieID int64, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).Model(&movies.MovieVideo{}).Where("movie_id = ?", movieID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("movie_video with movie_id %d not found", movieID)
	}
	return nil
}

// DeleteMovie deletes a movie (CASCADE will delete movie_videos too)
func (r *MovieRepository) DeleteMovie(ctx context.Context, movieID int64) error {
	result := r.db.WithContext(ctx).Delete(&movies.Movie{}, movieID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("movie with id %d not found", movieID)
	}
	return nil
}

// GetHLSURL gets the HLS playlist URL for a movie
func (r *MovieRepository) GetHLSURL(ctx context.Context, movieID int64) (string, error) {
	var movieVideo movies.MovieVideo
	err := r.db.WithContext(ctx).
		Where("movie_id = ? AND upload_status = ?", movieID, "READY").
		First(&movieVideo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("movie video not ready or not found")
		}
		return "", err
	}
	return movieVideo.HLSPlaylistURL, nil
}

// Genre-related methods

// GetAllGenres returns all available genres
func (r *MovieRepository) GetAllGenres(ctx context.Context) ([]movies.Genre, error) {
	var genres []movies.Genre
	err := r.db.WithContext(ctx).Order("name ASC").Find(&genres).Error
	return genres, err
}

// CreateGenre creates a new genre
func (r *MovieRepository) CreateGenre(ctx context.Context, genre *movies.Genre) error {
	return r.db.WithContext(ctx).Create(genre).Error
}

// DeleteGenre deletes a genre by ID
func (r *MovieRepository) DeleteGenre(ctx context.Context, genreID int) error {
	result := r.db.WithContext(ctx).Delete(&movies.Genre{}, genreID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("genre with id %d not found", genreID)
	}
	return nil
}

// getMovieGenres gets all genre names for a specific movie
func (r *MovieRepository) getMovieGenres(ctx context.Context, movieID int64) []string {
	var genreNames []string
	r.db.WithContext(ctx).
		Table("genres").
		Select("genres.name").
		Joins("JOIN movie_genres ON genres.id = movie_genres.genre_id").
		Where("movie_genres.movie_id = ?", movieID).
		Order("genres.name ASC").
		Pluck("name", &genreNames)
	return genreNames
}

// AddMovieGenres adds multiple genres to a movie
func (r *MovieRepository) AddMovieGenres(ctx context.Context, movieID int64, genreIDs []int) error {
	if len(genreIDs) == 0 {
		return nil
	}

	// Create movie_genre records
	var movieGenres []movies.MovieGenre
	for _, genreID := range genreIDs {
		movieGenres = append(movieGenres, movies.MovieGenre{
			MovieID: movieID,
			GenreID: genreID,
		})
	}

	return r.db.WithContext(ctx).Create(&movieGenres).Error
}

// RemoveAllMovieGenres removes all genres from a movie
func (r *MovieRepository) RemoveAllMovieGenres(ctx context.Context, movieID int64) error {
	return r.db.WithContext(ctx).
		Where("movie_id = ?", movieID).
		Delete(&movies.MovieGenre{}).Error
}

// GetMovieGenreIDs gets all genre IDs for a specific movie
func (r *MovieRepository) GetMovieGenreIDs(ctx context.Context, movieID int64) ([]int, error) {
	var genreIDs []int
	err := r.db.WithContext(ctx).
		Table("movie_genres").
		Where("movie_id = ?", movieID).
		Pluck("genre_id", &genreIDs).Error
	return genreIDs, err
}
