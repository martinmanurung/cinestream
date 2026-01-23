package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestID := c.Request().Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = uuid.New().String()
				c.Request().Header.Set("X-Request-Id", requestID)
			}

			// Set request ID to response header
			c.Response().Header().Set("X-Request-Id", requestID)

			// Create logger with request_id and store in context
			logger := log.With().
				Str("request_id", requestID).
				Logger()

			// Store logger in context for handlers to use
			c.Set("logger", &logger)

			// Log incoming request
			logger.Info().
				Str("method", c.Request().Method).
				Str("path", c.Request().URL.Path).
				Str("remote_ip", c.RealIP()).
				Msg("Incoming request")

			return next(c)
		}
	}
}

// GetLogger retrieves the logger from echo context
// If not found, returns the default logger
func GetLogger(c echo.Context) *zerolog.Logger {
	if logger, ok := c.Get("logger").(*zerolog.Logger); ok {
		return logger
	}
	return &log.Logger
}

// GetRequestID retrieves the request ID from echo context
func GetRequestID(c echo.Context) string {
	return c.Request().Header.Get("X-Request-Id")
}
