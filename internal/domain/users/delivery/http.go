package delivery

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/martinmanurung/cinestream/internal/domain/users"
	"github.com/martinmanurung/cinestream/pkg/constant"
	"github.com/martinmanurung/cinestream/pkg/middleware"
	"github.com/martinmanurung/cinestream/pkg/response"
)

type UserUsecase interface {
	RegisterUser(ctx context.Context, payload users.UserRegisterRequest) (*users.UserRegisterResponse, error)
	LoginUser(ctx context.Context, payload users.UserLoginRequest) (*users.UserLoginResponse, error)
	GetUserProfile(ctx context.Context, userExtID string) (*users.UserProfile, error)
	Logout(ctx context.Context, refreshToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (*users.RefreshTokenResponse, error)
}

type Handler struct {
	ctx     context.Context
	usecase UserUsecase
}

func NewHandler(ctx context.Context, usecase UserUsecase) *Handler {
	return &Handler{
		ctx:     ctx,
		usecase: usecase,
	}
}

func (h *Handler) RegisterUser(c echo.Context) error {
	logger := middleware.GetLogger(c)
	ctx := h.ctx

	logger.Info().Msg("Starting user registration")

	var req users.UserRegisterRequest

	if err := c.Bind(&req); err != nil {
		logger.Error().Err(err).Msg("Failed to bind request")
		return response.Error(c, http.StatusBadRequest, "invalid_request_body", err.Error())
	}

	if err := c.Validate(&req); err != nil {
		logger.Warn().Err(err).Msg("Validation failed")
		return response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
	}

	result, err := h.usecase.RegisterUser(ctx, req)
	if err != nil {
		var apiErr *response.APIError
		if errors, ok := err.(*response.APIError); ok {
			apiErr = errors
			logger.Error().
				Err(err).
				Msg("Failed to register user")
			return response.Error(c, apiErr.Code, apiErr.Message, apiErr.Details)
		}
		logger.Error().Err(err).Msg("Internal server error during registration")
		return response.Error(c, http.StatusInternalServerError, "internal_server_error", err.Error())
	}

	logger.Info().
		Msg("User registered successfully")

	return response.Success(c, http.StatusCreated, "user_registered_successfully", result)
}

func (h *Handler) LoginUser(c echo.Context) error {
	logger := middleware.GetLogger(c)
	ctx := h.ctx

	logger.Info().Msg("User login attempt")

	var req users.UserLoginRequest

	if err := c.Bind(&req); err != nil {
		logger.Error().Err(err).Msg("Failed to bind login request")
		return response.Error(c, http.StatusBadRequest, "invalid_request_body", err.Error())
	}

	if err := c.Validate(&req); err != nil {
		logger.Warn().Err(err).Msg("Login validation failed")
		return response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
	}

	result, err := h.usecase.LoginUser(ctx, req)
	if err != nil {
		var apiErr *response.APIError
		if errors, ok := err.(*response.APIError); ok {
			apiErr = errors
			logger.Warn().
				Msg("Login failed")
			return response.Error(c, apiErr.Code, apiErr.Message, apiErr.Details)
		}
		logger.Error().Err(err).Msg("Internal server error during login")
		return response.Error(c, http.StatusInternalServerError, "internal_server_error", err.Error())
	}

	logger.Info().
		Msg("User logged in successfully")

	return response.Success(c, http.StatusOK, "login_successful", result)
}

func (h *Handler) GetMe(c echo.Context) error {
	ctx := h.ctx

	// Extract user_ext_id from echo context and set to standard context
	userExtID := c.Get(string(constant.CtxKeyUserExtID))
	ctx = context.WithValue(ctx, constant.CtxKeyUserExtID, userExtID)

	// Get user ext_id from context
	extID, ok := userExtID.(string)
	if !ok || extID == "" {
		return response.Error(c, http.StatusUnauthorized, "unauthorized", "invalid token")
	}

	result, err := h.usecase.GetUserProfile(ctx, extID)
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

func (h *Handler) Logout(c echo.Context) error {
	ctx := h.ctx
	var req users.LogoutRequest

	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "invalid_request_body", err.Error())
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
	}

	err := h.usecase.Logout(ctx, req.RefreshToken)
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

func (h *Handler) RefreshToken(c echo.Context) error {
	ctx := h.ctx
	var req users.RefreshTokenRequest

	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "invalid_request_body", err.Error())
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
	}

	result, err := h.usecase.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		var apiErr *response.APIError
		if errors, ok := err.(*response.APIError); ok {
			apiErr = errors
			return response.Error(c, apiErr.Code, apiErr.Message, apiErr.Details)
		}
		return response.Error(c, http.StatusInternalServerError, "internal_server_error", err.Error())
	}

	return response.Success(c, http.StatusOK, "token_refreshed_successfully", result)
}
