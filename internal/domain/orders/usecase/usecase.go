package usecase

import (
	"fmt"
	"math"
	"time"

	"github.com/martinmanurung/cinestream/internal/domain/orders"
	orderRepository "github.com/martinmanurung/cinestream/internal/domain/orders/repository"
	"github.com/martinmanurung/cinestream/internal/platform/payment"
	"gorm.io/gorm"
)

// MovieRepository defines minimal movie repository interface needed by order usecase
type MovieRepository interface {
	FindMovieByID(movieID int64) (map[string]interface{}, error)
	GetMovieHLSURL(movieID int64) (string, error)
}

// UserRepository defines minimal user repository interface needed by order usecase
type UserRepository interface {
	FindUserByExtID(userExtID string) (map[string]interface{}, error)
}

// OrderUsecase defines the interface for order business logic
type OrderUsecase interface {
	CreateOrder(userExtID string, req *orders.CreateOrderRequest) (*orders.CreateOrderResponse, error)
	GetUserOrders(userExtID string, page, limit int) (*orders.OrdersListWrapper, error)
	GetAllOrders(page, limit int, status string) (*orders.OrdersListWrapper, error)
	GetOrderDetail(orderID int64) (*orders.OrderDetailResponse, error)
	CheckStreamAccess(userExtID string, movieID int64) (*orders.StreamURLResponse, error)
	SimulatePaymentSuccess(orderID int64) error // For development/testing
}

type orderUsecase struct {
	orderRepo      orderRepository.OrderRepository
	movieRepo      MovieRepository
	userRepo       UserRepository
	paymentService payment.PaymentService
}

// NewOrderUsecase creates a new order usecase
func NewOrderUsecase(
	orderRepo orderRepository.OrderRepository,
	movieRepo MovieRepository,
	userRepo UserRepository,
	paymentService payment.PaymentService,
) OrderUsecase {
	return &orderUsecase{
		orderRepo:      orderRepo,
		movieRepo:      movieRepo,
		userRepo:       userRepo,
		paymentService: paymentService,
	}
}

// CreateOrder creates a new order and initiates payment
func (u *orderUsecase) CreateOrder(userExtID string, req *orders.CreateOrderRequest) (*orders.CreateOrderResponse, error) {
	// 1. Get movie details and price
	movie, err := u.movieRepo.FindMovieByID(req.MovieID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("movie not found")
		}
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}

	price, ok := movie["price"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid movie price")
	}

	// 2. Get user details
	user, err := u.userRepo.FindUserByExtID(userExtID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	userEmail, _ := user["email"].(string)
	userName, _ := user["name"].(string)

	// 3. Create order record with PENDING status
	order := &orders.Order{
		UserExtID:     userExtID,
		MovieID:       req.MovieID,
		Amount:        price,
		PaymentStatus: orders.PaymentStatusPending,
	}

	if err := u.orderRepo.CreateOrder(order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// 4. Create payment transaction with Midtrans
	checkoutURL, paymentRef, err := u.paymentService.CreateTransaction(
		order.ID,
		price,
		userEmail,
		userName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment transaction: %w", err)
	}

	// 5. Update order with payment details
	expiresAt := time.Now().Add(24 * time.Hour) // Payment link expires in 24 hours

	if err := u.orderRepo.UpdateOrderPaymentDetails(order.ID, paymentRef, checkoutURL, &expiresAt); err != nil {
		return nil, fmt.Errorf("failed to update order payment details: %w", err)
	}

	// 6. Return response
	return &orders.CreateOrderResponse{
		OrderID:     order.ID,
		CheckoutURL: checkoutURL,
		Amount:      price,
		Message:     "Order created successfully. Please proceed to payment.",
	}, nil
}

// GetUserOrders retrieves all orders for a specific user with pagination
func (u *orderUsecase) GetUserOrders(userExtID string, page, limit int) (*orders.OrdersListWrapper, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	ordersList, total, err := u.orderRepo.FindOrdersByUserExtID(userExtID, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get user orders: %w", err)
	}

	// Map to response DTOs
	orderResponses := make([]orders.OrderListResponse, len(ordersList))
	for i, order := range ordersList {
		paymentRef := ""
		if order.PaymentGatewayRef != nil {
			paymentRef = *order.PaymentGatewayRef
		}

		orderResponses[i] = orders.OrderListResponse{
			ID:                order.ID,
			MovieID:           order.MovieID,
			MovieTitle:        order.MovieTitle,
			Amount:            order.Amount,
			PaymentStatus:     order.PaymentStatus,
			PaymentGatewayRef: paymentRef,
			PaidAt:            order.PaidAt,
			CreatedAt:         order.CreatedAt,
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return &orders.OrdersListWrapper{
		Orders: orderResponses,
		Pagination: orders.PaginationMeta{
			CurrentPage: page,
			TotalPages:  totalPages,
			TotalItems:  total,
			PerPage:     limit,
		},
	}, nil
}

// GetAllOrders retrieves all orders (admin) with optional status filter and pagination
func (u *orderUsecase) GetAllOrders(page, limit int, status string) (*orders.OrdersListWrapper, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	ordersList, total, err := u.orderRepo.FindAllOrders(page, limit, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get all orders: %w", err)
	}

	// Map to response DTOs
	orderResponses := make([]orders.OrderListResponse, len(ordersList))
	for i, order := range ordersList {
		paymentRef := ""
		if order.PaymentGatewayRef != nil {
			paymentRef = *order.PaymentGatewayRef
		}

		orderResponses[i] = orders.OrderListResponse{
			ID:                order.ID,
			MovieID:           order.MovieID,
			MovieTitle:        order.MovieTitle,
			Amount:            order.Amount,
			PaymentStatus:     order.PaymentStatus,
			PaymentGatewayRef: paymentRef,
			PaidAt:            order.PaidAt,
			CreatedAt:         order.CreatedAt,
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return &orders.OrdersListWrapper{
		Orders: orderResponses,
		Pagination: orders.PaginationMeta{
			CurrentPage: page,
			TotalPages:  totalPages,
			TotalItems:  total,
			PerPage:     limit,
		},
	}, nil
}

// GetOrderDetail retrieves detailed information about an order
func (u *orderUsecase) GetOrderDetail(orderID int64) (*orders.OrderDetailResponse, error) {
	order, err := u.orderRepo.FindOrderByID(orderID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	paymentRef := ""
	if order.PaymentGatewayRef != nil {
		paymentRef = *order.PaymentGatewayRef
	}

	checkoutURL := ""
	if order.CheckoutURL != nil {
		checkoutURL = *order.CheckoutURL
	}

	return &orders.OrderDetailResponse{
		ID:                order.ID,
		UserExtID:         order.UserExtID,
		UserName:          order.UserName,
		UserEmail:         order.UserEmail,
		MovieID:           order.MovieID,
		MovieTitle:        order.MovieTitle,
		Amount:            order.Amount,
		PaymentStatus:     order.PaymentStatus,
		PaymentGatewayRef: paymentRef,
		CheckoutURL:       checkoutURL,
		PaidAt:            order.PaidAt,
		ExpiresAt:         order.ExpiresAt,
		CreatedAt:         order.CreatedAt,
		UpdatedAt:         order.UpdatedAt,
	}, nil
}

// CheckStreamAccess checks if user has access to stream a movie
func (u *orderUsecase) CheckStreamAccess(userExtID string, movieID int64) (*orders.StreamURLResponse, error) {
	// 1. Check if user has active access
	access, err := u.orderRepo.CheckUserAccess(userExtID, movieID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("access denied: you need to rent this movie first")
		}
		return nil, fmt.Errorf("failed to check access: %w", err)
	}

	// 2. Get HLS URL from movie
	hlsURL, err := u.movieRepo.GetMovieHLSURL(movieID)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie stream URL: %w", err)
	}

	// 3. Return stream URL
	message := "Access granted. Enjoy your movie!"
	if access.AccessExpiresAt != nil {
		message = fmt.Sprintf("Access granted until %s", access.AccessExpiresAt.Format("2006-01-02 15:04:05"))
	}

	return &orders.StreamURLResponse{
		HLSURL:          hlsURL,
		AccessExpiresAt: access.AccessExpiresAt,
		Message:         message,
	}, nil
}

// SimulatePaymentSuccess simulates a successful payment (for development/testing only)
// This method updates order status to PAID and grants movie access to the user
func (u *orderUsecase) SimulatePaymentSuccess(orderID int64) error {
	// 1. Get order details
	order, err := u.orderRepo.FindOrderByID(orderID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("order not found")
		}
		return fmt.Errorf("failed to get order: %w", err)
	}

	// 2. Check if already paid
	if order.PaymentStatus == orders.PaymentStatusPaid {
		return fmt.Errorf("order already paid")
	}

	// 3. Update order status to PAID
	now := time.Now()
	if err := u.orderRepo.UpdateOrderStatus(orderID, orders.PaymentStatusPaid, &now); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// 4. Grant user access to the movie
	access := &orders.UserMovieAccess{
		UserExtID:       order.UserExtID,
		MovieID:         order.MovieID,
		OrderID:         orderID,
		AccessGrantedAt: now,
		AccessExpiresAt: nil, // Permanent access (or set expiration as needed)
	}

	if err := u.orderRepo.CreateUserMovieAccess(access); err != nil {
		return fmt.Errorf("failed to grant movie access: %w", err)
	}

	fmt.Printf("INFO - Simulated payment success for order %d, granted access to user %s for movie %d\n",
		orderID, order.UserExtID, order.MovieID)

	return nil
}
