package orders

import "time"

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "PENDING"
	PaymentStatusPaid    PaymentStatus = "PAID"
	PaymentStatusFailed  PaymentStatus = "FAILED"
	PaymentStatusExpired PaymentStatus = "EXPIRED"
)

// Order represents an order in the system
type Order struct {
	ID                int64         `json:"id" gorm:"primaryKey;autoIncrement"`
	UserExtID         string        `json:"user_ext_id" gorm:"not null;index;column:user_ext_id"`
	MovieID           int64         `json:"movie_id" gorm:"not null;index"`
	Amount            float64       `json:"amount" gorm:"type:decimal(10,2);not null"`
	PaymentStatus     PaymentStatus `json:"payment_status" gorm:"type:enum('PENDING','PAID','FAILED','EXPIRED');default:'PENDING';not null"`
	PaymentGatewayRef *string       `json:"payment_gateway_ref,omitempty" gorm:"unique"`
	CheckoutURL       *string       `json:"checkout_url,omitempty" gorm:"type:text"`
	PaidAt            *time.Time    `json:"paid_at,omitempty"`
	ExpiresAt         *time.Time    `json:"expires_at,omitempty"`
	CreatedAt         time.Time     `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time     `json:"updated_at" gorm:"autoUpdateTime"`

	// Relations (not persisted in database, loaded via joins/preload)
	MovieTitle string `json:"movie_title,omitempty" gorm:"-"`
	UserName   string `json:"user_name,omitempty" gorm:"-"`
	UserEmail  string `json:"user_email,omitempty" gorm:"-"`
}

// TableName specifies the table name for Order model
func (Order) TableName() string {
	return "orders"
}

// UserMovieAccess represents user's access rights to a movie after purchase
type UserMovieAccess struct {
	ID              int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserExtID       string     `json:"user_ext_id" gorm:"not null;index;column:user_ext_id"`
	MovieID         int64      `json:"movie_id" gorm:"not null;index"`
	OrderID         int64      `json:"order_id" gorm:"not null;unique"`
	AccessGrantedAt time.Time  `json:"access_granted_at" gorm:"autoCreateTime"`
	AccessExpiresAt *time.Time `json:"access_expires_at,omitempty"` // NULL = permanent access
	CreatedAt       time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for UserMovieAccess model
func (UserMovieAccess) TableName() string {
	return "user_movie_access"
}

// CreateOrderRequest represents the request to create a new order
type CreateOrderRequest struct {
	MovieID int64 `json:"movie_id" validate:"required,gt=0"`
}

// CreateOrderResponse represents the response after creating an order
type CreateOrderResponse struct {
	OrderID     int64   `json:"order_id"`
	CheckoutURL string  `json:"checkout_url"`
	Amount      float64 `json:"amount"`
	Message     string  `json:"message"`
}

// OrderListResponse represents a single order in list view
type OrderListResponse struct {
	ID                int64         `json:"id"`
	MovieID           int64         `json:"movie_id"`
	MovieTitle        string        `json:"movie_title"`
	Amount            float64       `json:"amount"`
	PaymentStatus     PaymentStatus `json:"payment_status"`
	PaymentGatewayRef string        `json:"payment_gateway_ref,omitempty"`
	PaidAt            *time.Time    `json:"paid_at,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
}

// OrderDetailResponse represents detailed order information
type OrderDetailResponse struct {
	ID                int64         `json:"id"`
	UserExtID         string        `json:"user_ext_id"`
	UserName          string        `json:"user_name,omitempty"`
	UserEmail         string        `json:"user_email,omitempty"`
	MovieID           int64         `json:"movie_id"`
	MovieTitle        string        `json:"movie_title"`
	Amount            float64       `json:"amount"`
	PaymentStatus     PaymentStatus `json:"payment_status"`
	PaymentGatewayRef string        `json:"payment_gateway_ref,omitempty"`
	CheckoutURL       string        `json:"checkout_url,omitempty"`
	PaidAt            *time.Time    `json:"paid_at,omitempty"`
	ExpiresAt         *time.Time    `json:"expires_at,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// OrdersListWrapper wraps the list of orders with pagination
type OrdersListWrapper struct {
	Orders     []OrderListResponse `json:"orders"`
	Pagination PaginationMeta      `json:"pagination"`
}

// PaginationMeta contains pagination metadata
type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	TotalPages  int   `json:"total_pages"`
	TotalItems  int64 `json:"total_items"`
	PerPage     int   `json:"per_page"`
}

// StreamURLResponse represents the response for streaming URL request
type StreamURLResponse struct {
	HLSURL          string     `json:"hls_url"`
	AccessExpiresAt *time.Time `json:"access_expires_at,omitempty"`
	Message         string     `json:"message"`
}
