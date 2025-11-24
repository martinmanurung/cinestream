package delivery

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/martinmanurung/cinestream/internal/domain/movies"
	"github.com/martinmanurung/cinestream/pkg/response"
)

type GenreUsecase interface {
	GetAllGenres(ctx context.Context) (*movies.GenreListResponse, error)
	CreateGenre(ctx context.Context, req movies.GenreRequest) (*movies.Genre, error)
	DeleteGenre(ctx context.Context, genreID int) error
}

type GenreHandler struct {
	ctx     context.Context
	usecase GenreUsecase
}

func NewGenreHandler(ctx context.Context, usecase GenreUsecase) *GenreHandler {
	return &GenreHandler{
		ctx:     ctx,
		usecase: usecase,
	}
}

// GetAllGenres returns all available genres (Public)
// GET /api/v1/genres
func (h *GenreHandler) GetAllGenres(c echo.Context) error {
	ctx := h.ctx

	result, err := h.usecase.GetAllGenres(ctx)
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

// CreateGenre creates a new genre (Admin only)
// POST /api/v1/admin/genres
func (h *GenreHandler) CreateGenre(c echo.Context) error {
	ctx := h.ctx

	var req movies.GenreRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "invalid_request_body", err.Error())
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
	}

	result, err := h.usecase.CreateGenre(ctx, req)
	if err != nil {
		var apiErr *response.APIError
		if errors, ok := err.(*response.APIError); ok {
			apiErr = errors
			return response.Error(c, apiErr.Code, apiErr.Message, apiErr.Details)
		}
		return response.Error(c, http.StatusInternalServerError, "internal_server_error", err.Error())
	}

	return response.Success(c, http.StatusCreated, "genre_created", result)
}

// DeleteGenre deletes a genre (Admin only)
// DELETE /api/v1/admin/genres/:id
func (h *GenreHandler) DeleteGenre(c echo.Context) error {
	ctx := h.ctx

	genreID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "invalid_genre_id", err.Error())
	}

	err = h.usecase.DeleteGenre(ctx, genreID)
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
