package jwt

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/martinmanurung/cinestream/pkg/constant"
	"github.com/martinmanurung/cinestream/pkg/response"
)

type MyClaims struct {
	UserExtID string `json:"user_ext_id"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

type JWTService struct {
	UserExtID    string
	SignatureKey []byte
}

func NewJWTService(secretKey string) *JWTService {
	return &JWTService{
		SignatureKey: []byte(secretKey),
	}
}

func (j *JWTService) GenerateToken(userExtID string, role string) (string, error) {
	if userExtID == "" {
		return "", errors.New("user_ext_id cannot be empty")
	}

	if j.SignatureKey == nil {
		return "", errors.New("signature_key cannot be empty")
	}

	claims := MyClaims{
		UserExtID: userExtID,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.SignatureKey)
}

func (j *JWTService) ValidateToken(tokenStr string) (*MyClaims, error) {
	// Remove "Bearer " prefix if exists
	if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
		tokenStr = tokenStr[7:]
	}

	token, err := jwt.ParseWithClaims(tokenStr, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("invalid signing method")
		}
		return j.SignatureKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*MyClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (j *JWTService) JWTMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := c.Request().Header.Get(echo.HeaderAuthorization)
			if token == "" {
				return response.Error(c, 401, "unauthorized", "missing authorization token")
			}

			claims, err := j.ValidateToken(token)
			if err != nil {
				return response.Error(c, 401, "unauthorized", err.Error())
			}

			c.Set(string(constant.CtxKeyUserExtID), claims.UserExtID)
			c.Set(string(constant.CtxKeyUserRole), claims.Role)
			return next(c)
		}
	}
}

// GetUserExtIDFromContext extracts user_ext_id from echo context
func GetUserExtIDFromContext(c echo.Context) (string, error) {
	userExtID, ok := c.Get(string(constant.CtxKeyUserExtID)).(string)
	if !ok || userExtID == "" {
		return "", errors.New("user_ext_id not found in context")
	}
	return userExtID, nil
}

// SetUserExtIDToContext sets user_ext_id to standard context
func SetUserExtIDToContext(ctx context.Context, userExtID string) context.Context {
	return context.WithValue(ctx, constant.CtxKeyUserExtID, userExtID)
}

// GetUserExtIDFromStdContext extracts user_ext_id from standard context
func GetUserExtIDFromStdContext(ctx context.Context) (string, error) {
	userExtID, ok := ctx.Value(constant.CtxKeyUserExtID).(string)
	if !ok || userExtID == "" {
		return "", errors.New("user_ext_id not found in context")
	}
	return userExtID, nil
}
