package response

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

type SuccessResponse struct {
	Status  string      `json:"status"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Status  string      `json:"status"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Errors  interface{} `json:"errors,omitempty"`
}

func Success(c echo.Context, code int, message string, data interface{}) error {
	return c.JSON(code, SuccessResponse{
		Status:  "success",
		Code:    code,
		Message: message,
		Data:    data,
	})
}

func Error(c echo.Context, code int, message string, errDetails interface{}) error {
	return c.JSON(code, ErrorResponse{
		Status:  "error",
		Code:    code,
		Message: message,
		Errors:  errDetails,
	})
}

type APIError struct {
	Code    int
	Message string
	Details interface{}
}

func (e *APIError) Error() string {
	return e.Message
}

func NewError(code int, message string, details interface{}) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

func CustomErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		Error(c, apiErr.Code, apiErr.Message, apiErr.Details)
		return
	}

	var echoErr *echo.HTTPError
	if errors.As(err, &echoErr) {
		var msg string
		if s, ok := echoErr.Message.(string); ok {
			msg = s
		} else {
			msg = "An error occurred" // Fallback
		}
		Error(c, echoErr.Code, msg, nil)
		return
	}
	c.Logger().Error(err)
	Error(c, http.StatusInternalServerError, "Internal Server Error", nil)
}

func InternalServerError(err error) error {
	return &APIError{
		Code:    http.StatusInternalServerError,
		Message: "internal_server_error",
		Details: err,
	}
}
