package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/martinmanurung/cinestream/pkg/constant"
	"github.com/martinmanurung/cinestream/pkg/response"
)

// AdminOnly middleware checks if the user has ADMIN role
func AdminOnly() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get user role from context (set by JWT middleware)
			role := c.Get(string(constant.CtxKeyUserRole))

			if role == nil {
				return response.Error(c, http.StatusUnauthorized, "unauthorized", "missing role information")
			}

			userRole, ok := role.(string)
			if !ok || userRole != "ADMIN" {
				return response.Error(c, http.StatusForbidden, "forbidden", "admin access required")
			}

			return next(c)
		}
	}
}
