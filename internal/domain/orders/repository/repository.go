package repository

import (
	"time"

	"github.com/martinmanurung/cinestream/internal/domain/orders"
	"gorm.io/gorm"
)

// OrderRepository defines the interface for order data operations
type OrderRepository interface {
	CreateOrder(order *orders.Order) error
	FindOrderByID(orderID int64) (*orders.Order, error)
	FindOrdersByUserExtID(userExtID string, page, limit int) ([]orders.Order, int64, error)
	FindAllOrders(page, limit int, status string) ([]orders.Order, int64, error)
	UpdateOrderStatus(orderID int64, status orders.PaymentStatus, paidAt *time.Time) error
	UpdateOrderPaymentDetails(orderID int64, paymentRef, checkoutURL string, expiresAt *time.Time) error
	FindOrderByPaymentRef(paymentRef string) (*orders.Order, error)

	// User movie access operations
	CreateUserMovieAccess(access *orders.UserMovieAccess) error
	CheckUserAccess(userExtID string, movieID int64) (*orders.UserMovieAccess, error)
	FindUserAccessByOrderID(orderID int64) (*orders.UserMovieAccess, error)
}

type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

// CreateOrder creates a new order in the database
func (r *orderRepository) CreateOrder(order *orders.Order) error {
	return r.db.Create(order).Error
}

// FindOrderByID finds an order by ID with movie and user details
func (r *orderRepository) FindOrderByID(orderID int64) (*orders.Order, error) {
	var order orders.Order

	err := r.db.Table("orders").
		Select("orders.*, movies.title as movie_title, users.name as user_name, users.email as user_email").
		Joins("LEFT JOIN movies ON orders.movie_id = movies.id").
		Joins("LEFT JOIN users ON orders.user_ext_id = users.ext_id").
		Where("orders.id = ?", orderID).
		First(&order).Error

	if err != nil {
		return nil, err
	}

	return &order, nil
}

// FindOrdersByUserExtID finds all orders for a specific user with pagination
func (r *orderRepository) FindOrdersByUserExtID(userExtID string, page, limit int) ([]orders.Order, int64, error) {
	var ordersList []orders.Order
	var total int64

	offset := (page - 1) * limit

	// Count total orders
	if err := r.db.Model(&orders.Order{}).Where("user_ext_id = ?", userExtID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get orders with movie details
	err := r.db.Table("orders").
		Select("orders.*, movies.title as movie_title").
		Joins("LEFT JOIN movies ON orders.movie_id = movies.id").
		Where("orders.user_ext_id = ?", userExtID).
		Order("orders.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&ordersList).Error

	if err != nil {
		return nil, 0, err
	}

	return ordersList, total, nil
}

// FindAllOrders finds all orders with optional status filter and pagination
func (r *orderRepository) FindAllOrders(page, limit int, status string) ([]orders.Order, int64, error) {
	var ordersList []orders.Order
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&orders.Order{})

	// Apply status filter if provided
	if status != "" {
		query = query.Where("payment_status = ?", status)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get orders with movie and user details
	queryBuilder := r.db.Table("orders").
		Select("orders.*, movies.title as movie_title, users.name as user_name, users.email as user_email").
		Joins("LEFT JOIN movies ON orders.movie_id = movies.id").
		Joins("LEFT JOIN users ON orders.user_ext_id = users.ext_id")

	if status != "" {
		queryBuilder = queryBuilder.Where("orders.payment_status = ?", status)
	}

	err := queryBuilder.Order("orders.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&ordersList).Error

	if err != nil {
		return nil, 0, err
	}

	return ordersList, total, nil
}

// UpdateOrderStatus updates the payment status of an order
func (r *orderRepository) UpdateOrderStatus(orderID int64, status orders.PaymentStatus, paidAt *time.Time) error {
	updates := map[string]interface{}{
		"payment_status": status,
	}

	if paidAt != nil {
		updates["paid_at"] = paidAt
	}

	return r.db.Model(&orders.Order{}).
		Where("id = ?", orderID).
		Updates(updates).Error
}

// UpdateOrderPaymentDetails updates payment gateway reference, checkout URL, and expiration
func (r *orderRepository) UpdateOrderPaymentDetails(orderID int64, paymentRef, checkoutURL string, expiresAt *time.Time) error {
	updates := map[string]interface{}{
		"payment_gateway_ref": paymentRef,
		"checkout_url":        checkoutURL,
	}

	if expiresAt != nil {
		updates["expires_at"] = expiresAt
	}

	return r.db.Model(&orders.Order{}).
		Where("id = ?", orderID).
		Updates(updates).Error
}

// FindOrderByPaymentRef finds an order by payment gateway reference
func (r *orderRepository) FindOrderByPaymentRef(paymentRef string) (*orders.Order, error) {
	var order orders.Order

	err := r.db.Table("orders").
		Select("orders.*, movies.title as movie_title, users.name as user_name, users.email as user_email").
		Joins("LEFT JOIN movies ON orders.movie_id = movies.id").
		Joins("LEFT JOIN users ON orders.user_ext_id = users.ext_id").
		Where("orders.payment_gateway_ref = ?", paymentRef).
		First(&order).Error

	if err != nil {
		return nil, err
	}

	return &order, nil
}

// CreateUserMovieAccess creates a new user movie access record
func (r *orderRepository) CreateUserMovieAccess(access *orders.UserMovieAccess) error {
	return r.db.Create(access).Error
}

// CheckUserAccess checks if a user has access to a movie
func (r *orderRepository) CheckUserAccess(userExtID string, movieID int64) (*orders.UserMovieAccess, error) {
	var access orders.UserMovieAccess

	err := r.db.Where("user_ext_id = ? AND movie_id = ?", userExtID, movieID).
		Where("access_expires_at IS NULL OR access_expires_at > ?", time.Now()).
		First(&access).Error

	if err != nil {
		return nil, err
	}

	return &access, nil
}

// FindUserAccessByOrderID finds user movie access by order ID
func (r *orderRepository) FindUserAccessByOrderID(orderID int64) (*orders.UserMovieAccess, error) {
	var access orders.UserMovieAccess

	err := r.db.Where("order_id = ?", orderID).First(&access).Error
	if err != nil {
		return nil, err
	}

	return &access, nil
}
