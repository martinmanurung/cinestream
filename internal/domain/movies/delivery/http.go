package delivery

import (
	"context"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/martinmanurung/cinestream/internal/domain/movies"
	"github.com/martinmanurung/cinestream/pkg/response"
)

type MovieUsecase interface {
	UploadMovie(ctx context.Context, req movies.UploadMovieRequest, file multipart.File, fileHeader *multipart.FileHeader) (*movies.UploadMovieResponse, error)
	GetMovieList(ctx context.Context, page, limit int, genre string) (*movies.MovieListWithPagination, error)
	GetMovieDetail(ctx context.Context, movieID int64) (*movies.MovieDetailResponse, error)
	UpdateMovie(ctx context.Context, movieID int64, req movies.UpdateMovieRequest) error
	DeleteMovie(ctx context.Context, movieID int64) error
	GetAllMoviesAdmin(ctx context.Context, page, limit int, status string) (*movies.MovieListWithPagination, error)
}

type MovieHandler struct {
	ctx     context.Context
	usecase MovieUsecase
}

func NewMovieHandler(ctx context.Context, usecase MovieUsecase) *MovieHandler {
	return &MovieHandler{
		ctx:     ctx,
		usecase: usecase,
	}
}

// UploadMovie handles movie upload (Admin only)
// POST /api/v1/admin/movies
func (h *MovieHandler) UploadMovie(c echo.Context) error {
	ctx := h.ctx

	// Parse multipart form
	if err := c.Request().ParseMultipartForm(100 << 20); err != nil { // 100 MB max
		return response.Error(c, http.StatusBadRequest, "invalid_multipart_form", err.Error())
	}

	// Bind form data to request struct
	var req movies.UploadMovieRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "invalid_request_body", err.Error())
	}

	// Validate request
	if err := c.Validate(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
	}

	// Get video file from form
	file, fileHeader, err := c.Request().FormFile("videoFile")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "video_file_required", err.Error())
	}
	defer file.Close()

	// Validate file size (max 2GB)
	maxSize := int64(2 << 30) // 2GB
	if fileHeader.Size > maxSize {
		return response.Error(c, http.StatusBadRequest, "file_too_large", "maximum file size is 2GB")
	}

	// Call usecase
	result, err := h.usecase.UploadMovie(ctx, req, file, fileHeader)
	if err != nil {
		var apiErr *response.APIError
		if errors, ok := err.(*response.APIError); ok {
			apiErr = errors
			return response.Error(c, apiErr.Code, apiErr.Message, apiErr.Details)
		}
		return response.Error(c, http.StatusInternalServerError, "internal_server_error", err.Error())
	}

	return response.Success(c, http.StatusAccepted, result.Message, result)
}

// GetMovieList returns paginated list of movies (Public)
// GET /api/v1/movies?page=1&limit=12&genre=action
func (h *MovieHandler) GetMovieList(c echo.Context) error {
	ctx := h.ctx

	// Parse query params
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 12
	}

	genre := c.QueryParam("genre")

	// Call usecase
	result, err := h.usecase.GetMovieList(ctx, page, limit, genre)
	if err != nil {
		var apiErr *response.APIError
		if errors, ok := err.(*response.APIError); ok {
			apiErr = errors
			return response.Error(c, apiErr.Code, apiErr.Message, apiErr.Details)
		}
		return response.Error(c, http.StatusInternalServerError, "internal_server_error", err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":     "success",
		"data":       result.Movies,
		"pagination": result.Pagination,
	})
}

// GetMovieDetail returns detailed movie information (Public)
// GET /api/v1/movies/:id
func (h *MovieHandler) GetMovieDetail(c echo.Context) error {
	ctx := h.ctx

	// Parse movie ID from URL
	movieID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "invalid_movie_id", err.Error())
	}

	// Call usecase
	result, err := h.usecase.GetMovieDetail(ctx, movieID)
	if err != nil {
		var apiErr *response.APIError
		if errors, ok := err.(*response.APIError); ok {
			apiErr = errors
			return response.Error(c, apiErr.Code, apiErr.Message, apiErr.Details)
		}
		return response.Error(c, http.StatusInternalServerError, "internal_server_error", err.Error())
	}

	return response.Success(c, http.StatusOK, "success", result)
}

// UpdateMovie updates movie metadata (Admin only)
// PUT /api/v1/admin/movies/:id
func (h *MovieHandler) UpdateMovie(c echo.Context) error {
	ctx := h.ctx

	// Parse movie ID from URL
	movieID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "invalid_movie_id", err.Error())
	}

	// Bind request body
	var req movies.UpdateMovieRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "invalid_request_body", err.Error())
	}

	// Validate request
	if err := c.Validate(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
	}

	// Call usecase
	err = h.usecase.UpdateMovie(ctx, movieID, req)
	if err != nil {
		var apiErr *response.APIError
		if errors, ok := err.(*response.APIError); ok {
			apiErr = errors
			return response.Error(c, apiErr.Code, apiErr.Message, apiErr.Details)
		}
		return response.Error(c, http.StatusInternalServerError, "internal_server_error", err.Error())
	}

	return response.Success(c, http.StatusOK, "movie_updated_successfully", nil)
}

// DeleteMovie deletes a movie (Admin only)
// DELETE /api/v1/admin/movies/:id
func (h *MovieHandler) DeleteMovie(c echo.Context) error {
	ctx := h.ctx

	// Parse movie ID from URL
	movieID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "invalid_movie_id", err.Error())
	}

	// Call usecase
	err = h.usecase.DeleteMovie(ctx, movieID)
	if err != nil {
		var apiErr *response.APIError
		if errors, ok := err.(*response.APIError); ok {
			apiErr = errors
			return response.Error(c, apiErr.Code, apiErr.Message, apiErr.Details)
		}
		return response.Error(c, http.StatusInternalServerError, "internal_server_error", err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// GetAllMoviesAdmin returns all movies with any status (Admin only)
// GET /api/v1/admin/movies?page=1&limit=12&status=PENDING
func (h *MovieHandler) GetAllMoviesAdmin(c echo.Context) error {
	ctx := h.ctx

	// Parse query params
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 12
	}

	status := c.QueryParam("status") // PENDING, PROCESSING, READY, FAILED

	// Call usecase
	result, err := h.usecase.GetAllMoviesAdmin(ctx, page, limit, status)
	if err != nil {
		var apiErr *response.APIError
		if errors, ok := err.(*response.APIError); ok {
			apiErr = errors
			return response.Error(c, apiErr.Code, apiErr.Message, apiErr.Details)
		}
		return response.Error(c, http.StatusInternalServerError, "internal_server_error", err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":     "success",
		"data":       result.Movies,
		"pagination": result.Pagination,
	})
}

