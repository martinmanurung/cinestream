package delivery

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/martinmanurung/cinestream/internal/domain/orders/usecase"
	"github.com/martinmanurung/cinestream/pkg/constant"
	"github.com/martinmanurung/cinestream/pkg/response"
)

// StreamingHandler handles movie streaming requests
type StreamingHandler struct {
	ctx          context.Context
	orderUsecase usecase.OrderUsecase
}

// NewStreamingHandler creates a new streaming handler
func NewStreamingHandler(ctx context.Context, orderUsecase usecase.OrderUsecase) *StreamingHandler {
	return &StreamingHandler{
		ctx:          ctx,
		orderUsecase: orderUsecase,
	}
}

// GetStreamURL handles GET /api/v1/movies/:id/stream
// Returns HLS streaming URL if user has access
func (h *StreamingHandler) GetStreamURL(c echo.Context) error {
	// Get user_ext_id from JWT context
	userExtID, ok := c.Get(string(constant.CtxKeyUserExtID)).(string)
	if !ok || userExtID == "" {
		return response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
	}

	// Parse movie ID
	movieID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "Invalid movie ID", nil)
	}

	// Check access and get HLS URL using user_ext_id string directly
	streamResp, err := h.orderUsecase.CheckStreamAccess(userExtID, movieID)
	if err != nil {
		return response.Error(c, http.StatusForbidden, err.Error(), nil)
	}

	return response.Success(c, http.StatusOK, streamResp.Message, streamResp)
}
