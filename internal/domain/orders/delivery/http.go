package delivery

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/martinmanurung/cinestream/internal/domain/orders"
	"github.com/martinmanurung/cinestream/internal/domain/orders/usecase"
	"github.com/martinmanurung/cinestream/pkg/constant"
	"github.com/martinmanurung/cinestream/pkg/response"
)

// OrderHandler handles HTTP requests for order operations
type OrderHandler struct {
	ctx          context.Context
	orderUsecase usecase.OrderUsecase
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(ctx context.Context, orderUsecase usecase.OrderUsecase) *OrderHandler {
	return &OrderHandler{
		ctx:          ctx,
		orderUsecase: orderUsecase,
	}
}

// CreateOrder handles POST /api/v1/orders
// @Summary Create a new order to rent a movie
// @Tags Orders
// @Accept json
// @Produce json
// @Param request body orders.CreateOrderRequest true "Order Request"
// @Success 201 {object} response.Response{data=orders.CreateOrderResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/orders [post]
// @Security BearerAuth
func (h *OrderHandler) CreateOrder(c echo.Context) error {
	// Get user_ext_id from JWT context (set by middleware)
	userExtID, ok := c.Get(string(constant.CtxKeyUserExtID)).(string)
	if !ok || userExtID == "" {
		return response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
	}

	// Bind request
	var req orders.CreateOrderRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Invalid request body", nil)
	}

	// Validate request
	if err := c.Validate(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error(), nil)
	}

	// Create order using user_ext_id string directly
	result, err := h.orderUsecase.CreateOrder(userExtID, &req)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, http.StatusCreated, "Order created successfully", result)
}

// GetUserOrders handles GET /api/v1/orders/me
// @Summary Get current user's order history
// @Tags Orders
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} response.Response{data=orders.OrdersListWrapper}
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/orders/me [get]
// @Security BearerAuth
func (h *OrderHandler) GetUserOrders(c echo.Context) error {
	// Get user_ext_id from JWT context
	userExtID, ok := c.Get(string(constant.CtxKeyUserExtID)).(string)
	if !ok || userExtID == "" {
		return response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 {
		limit = 10
	}

	// Get orders using user_ext_id string directly
	result, err := h.orderUsecase.GetUserOrders(userExtID, page, limit)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, http.StatusOK, "Orders retrieved successfully", result)
}

// GetAllOrders handles GET /api/v1/admin/orders
// @Summary Get all orders (Admin only)
// @Tags Orders
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param status query string false "Filter by payment status" Enums(PENDING, PAID, FAILED, EXPIRED)
// @Success 200 {object} response.Response{data=orders.OrdersListWrapper}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/admin/orders [get]
// @Security BearerAuth
func (h *OrderHandler) GetAllOrders(c echo.Context) error {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 {
		limit = 20
	}

	// Get status filter
	status := c.QueryParam("status")

	// Get all orders
	result, err := h.orderUsecase.GetAllOrders(page, limit, status)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, http.StatusOK, "Orders retrieved successfully", result)
}

// GetOrderDetail handles GET /api/v1/orders/:id
// @Summary Get order detail by ID
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path int true "Order ID"
// @Success 200 {object} response.Response{data=orders.OrderDetailResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/orders/{id} [get]
// @Security BearerAuth
func (h *OrderHandler) GetOrderDetail(c echo.Context) error {
	// Parse order ID
	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "Invalid order ID", nil)
	}

	// Get order detail
	result, err := h.orderUsecase.GetOrderDetail(orderID)
	if err != nil {
		return response.Error(c, http.StatusNotFound, err.Error(), nil)
	}

	return response.Success(c, http.StatusOK, "Order detail retrieved successfully", result)
}

// SimulatePaymentSuccess handles POST /api/v1/orders/:id/simulate-payment
// @Summary Simulate payment success for testing (Development only)
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path int true "Order ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/orders/{id}/simulate-payment [post]
// @Security BearerAuth
func (h *OrderHandler) SimulatePaymentSuccess(c echo.Context) error {
	// Parse order ID
	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "Invalid order ID", nil)
	}

	// Simulate payment success
	if err := h.orderUsecase.SimulatePaymentSuccess(orderID); err != nil {
		return response.Error(c, http.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, http.StatusOK, "Payment simulated successfully. Movie access granted!", nil)
}
