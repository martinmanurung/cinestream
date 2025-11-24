package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/martinmanurung/cinestream/internal/domain/users"
	"github.com/martinmanurung/cinestream/pkg/jwt"
	"github.com/martinmanurung/cinestream/pkg/response"
	"github.com/segmentio/ksuid"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	CreateNewUser(ctx context.Context, user users.User) error
	FindUserByEmail(ctx context.Context, email string) (*users.User, error)
	FindUserByExtID(ctx context.Context, extID string) (*users.User, error)
	FindUserByID(ctx context.Context, userID int) (*users.User, error)
	CreateRefreshToken(ctx context.Context, token users.UserRefreshToken) error
	FindRefreshToken(ctx context.Context, tokenHash string) (*users.UserRefreshToken, error)
	DeleteRefreshToken(ctx context.Context, tokenHash string) error
}

type Usecase struct {
	repo       UserRepository
	jwtService *jwt.JWTService
}

func NewUsecase(repo UserRepository, jwtService *jwt.JWTService) *Usecase {
	return &Usecase{
		repo:       repo,
		jwtService: jwtService,
	}
}

func (u Usecase) RegisterUser(ctx context.Context, payload users.UserRegisterRequest) (*users.UserRegisterResponse, error) {
	val, err := u.repo.FindUserByEmail(ctx, payload.Email)
	if err != nil {
		return nil, response.InternalServerError(err)
	}

	if val != nil {
		return nil, response.NewError(http.StatusConflict, "email_already_exists", nil)
	}

	if payload.Password == "" {
		return nil, response.NewError(http.StatusBadRequest, "password_required", nil)
	}

	hashPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, response.InternalServerError(err)
	}

	extID := "user_" + ksuid.New().String()

	user := users.User{
		ExtID:     extID,
		Name:      payload.Name,
		Email:     payload.Email,
		Password:  string(hashPassword),
		Role:      "USER",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := u.repo.CreateNewUser(ctx, user); err != nil {
		return nil, err
	}

	return &users.UserRegisterResponse{
		ExtID: extID,
		Name:  payload.Name,
		Email: payload.Email,
	}, nil
}

func (u Usecase) LoginUser(ctx context.Context, payload users.UserLoginRequest) (*users.UserLoginResponse, error) {
	// Find user by email
	user, err := u.repo.FindUserByEmail(ctx, payload.Email)
	if err != nil {
		return nil, response.InternalServerError(err)
	}

	if user == nil {
		return nil, response.NewError(http.StatusUnauthorized, "invalid_credentials", nil)
	}

	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(payload.Password))
	if err != nil {
		return nil, response.NewError(http.StatusUnauthorized, "invalid_credentials", nil)
	}

	// Generate JWT access token
	token, err := u.jwtService.GenerateToken(user.ExtID, user.Role)
	if err != nil {
		return nil, response.InternalServerError(err)
	}

	// Generate refresh token (32 bytes random string)
	refreshTokenBytes := make([]byte, 32)
	if _, err := rand.Read(refreshTokenBytes); err != nil {
		return nil, response.InternalServerError(err)
	}
	refreshToken := hex.EncodeToString(refreshTokenBytes)

	// Hash refresh token using SHA256 for storage
	hash := sha256.Sum256([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Store refresh token with 7 days expiry
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	refreshTokenRecord := users.UserRefreshToken{
		UserExtID: user.ExtID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	if err := u.repo.CreateRefreshToken(ctx, refreshTokenRecord); err != nil {
		return nil, response.InternalServerError(err)
	}

	return &users.UserLoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User: users.UserProfile{
			ExtID: user.ExtID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}

func (u Usecase) GetUserProfile(ctx context.Context, userExtID string) (*users.UserProfile, error) {
	user, err := u.repo.FindUserByExtID(ctx, userExtID)
	if err != nil {
		return nil, response.InternalServerError(err)
	}

	if user == nil {
		return nil, response.NewError(http.StatusNotFound, "user_not_found", nil)
	}

	return &users.UserProfile{
		ExtID: user.ExtID,
		Name:  user.Name,
		Email: user.Email,
		Role:  user.Role,
	}, nil
}

func (u Usecase) Logout(ctx context.Context, refreshToken string) error {
	// Hash the incoming refresh token to match stored hash
	hash := sha256.Sum256([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Verify token exists and not expired
	storedToken, err := u.repo.FindRefreshToken(ctx, tokenHash)
	if err != nil {
		return response.InternalServerError(err)
	}

	if storedToken == nil {
		return response.NewError(http.StatusUnauthorized, "invalid_refresh_token", nil)
	}

	// Delete the refresh token
	if err := u.repo.DeleteRefreshToken(ctx, tokenHash); err != nil {
		return response.InternalServerError(err)
	}

	return nil
}

func (u Usecase) RefreshToken(ctx context.Context, refreshToken string) (*users.RefreshTokenResponse, error) {
	// Hash the incoming refresh token to match stored hash
	hash := sha256.Sum256([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Find and verify token exists and not expired
	storedToken, err := u.repo.FindRefreshToken(ctx, tokenHash)
	if err != nil {
		return nil, response.InternalServerError(err)
	}

	if storedToken == nil {
		return nil, response.NewError(http.StatusUnauthorized, "invalid_or_expired_refresh_token", nil)
	}

	// Get user data to generate new access token
	user, err := u.repo.FindUserByExtID(ctx, storedToken.UserExtID)
	if err != nil {
		return nil, response.InternalServerError(err)
	}

	if user == nil {
		return nil, response.NewError(http.StatusNotFound, "user_not_found", nil)
	}

	// Generate new access token (JWT, 1 hour expiry)
	accessToken, err := u.jwtService.GenerateToken(user.ExtID, user.Role)
	if err != nil {
		return nil, response.InternalServerError(err)
	}

	return &users.RefreshTokenResponse{
		AccessToken: accessToken,
	}, nil
}
