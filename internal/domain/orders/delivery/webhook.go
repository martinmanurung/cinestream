package delivery

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/martinmanurung/cinestream/internal/domain/orders"
	orderRepository "github.com/martinmanurung/cinestream/internal/domain/orders/repository"
	"github.com/martinmanurung/cinestream/internal/platform/payment"
	"github.com/martinmanurung/cinestream/pkg/response"
)

// WebhookHandler handles payment gateway webhooks
type WebhookHandler struct {
	ctx            context.Context
	orderRepo      orderRepository.OrderRepository
	paymentService payment.PaymentService
	serverKey      string
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(
	ctx context.Context,
	orderRepo orderRepository.OrderRepository,
	paymentService payment.PaymentService,
	serverKey string,
) *WebhookHandler {
	return &WebhookHandler{
		ctx:            ctx,
		orderRepo:      orderRepo,
		paymentService: paymentService,
		serverKey:      serverKey,
	}
}

// MidtransNotification represents the webhook payload from Midtrans
type MidtransNotification struct {
	TransactionStatus string `json:"transaction_status"`
	OrderID           string `json:"order_id"`
	GrossAmount       string `json:"gross_amount"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
	PaymentType       string `json:"payment_type"`
	TransactionID     string `json:"transaction_id"`
	FraudStatus       string `json:"fraud_status"`
	TransactionTime   string `json:"transaction_time"`
}

// HandlePaymentWebhook handles POST /api/v1/webhooks/payment
// @Summary Handle payment notification from Midtrans
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param notification body MidtransNotification true "Payment Notification"
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/webhooks/payment [post]
func (h *WebhookHandler) HandlePaymentWebhook(c echo.Context) error {
	// 1. Parse webhook payload
	var notification MidtransNotification
	if err := c.Bind(&notification); err != nil {
		log.Printf("[WEBHOOK] Failed to parse notification: %v", err)
		return response.Error(c, http.StatusBadRequest, "Invalid notification payload", nil)
	}

	log.Printf("[WEBHOOK] Received notification for order: %s, status: %s",
		notification.OrderID, notification.TransactionStatus)

	// 2. Verify signature to ensure request is authentic
	isValid := h.paymentService.VerifySignature(
		notification.OrderID,
		notification.StatusCode,
		notification.GrossAmount,
		h.serverKey,
		notification.SignatureKey,
	)

	if !isValid {
		log.Printf("[WEBHOOK] Invalid signature for order: %s", notification.OrderID)
		return response.Error(c, http.StatusUnauthorized, "Invalid signature", nil)
	}

	log.Printf("[WEBHOOK] Signature verified for order: %s", notification.OrderID)

	// 3. Find order by payment gateway reference
	order, err := h.orderRepo.FindOrderByPaymentRef(notification.OrderID)
	if err != nil {
		log.Printf("[WEBHOOK] Order not found: %s, error: %v", notification.OrderID, err)
		return response.Error(c, http.StatusNotFound, "Order not found", nil)
	}

	log.Printf("[WEBHOOK] Found order ID: %d for payment ref: %s", order.ID, notification.OrderID)

	// 4. Process based on transaction status
	switch notification.TransactionStatus {
	case "capture", "settlement":
		// Payment successful
		if notification.FraudStatus == "accept" || notification.FraudStatus == "" {
			if err := h.handleSuccessfulPayment(order); err != nil {
				log.Printf("[WEBHOOK] Failed to process successful payment: %v", err)
				return response.Error(c, http.StatusInternalServerError, err.Error(), nil)
			}
			log.Printf("[WEBHOOK] Successfully processed payment for order: %d", order.ID)
		}

	case "pending":
		// Payment pending, no action needed
		log.Printf("[WEBHOOK] Payment pending for order: %d", order.ID)

	case "deny", "cancel", "expire":
		// Payment failed or cancelled
		now := time.Now()
		if err := h.orderRepo.UpdateOrderStatus(order.ID, orders.PaymentStatusFailed, &now); err != nil {
			log.Printf("[WEBHOOK] Failed to update failed order status: %v", err)
			return response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		}
		log.Printf("[WEBHOOK] Payment failed/cancelled for order: %d, status: %s",
			order.ID, notification.TransactionStatus)
	}

	// 5. Return 200 OK to acknowledge receipt
	return response.Success(c, http.StatusOK, "Notification processed", nil)
}

// handleSuccessfulPayment processes a successful payment
func (h *WebhookHandler) handleSuccessfulPayment(order *orders.Order) error {
	// 1. Update order status to PAID
	now := time.Now()
	if err := h.orderRepo.UpdateOrderStatus(order.ID, orders.PaymentStatusPaid, &now); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	log.Printf("[WEBHOOK] Updated order %d status to PAID", order.ID)

	// 2. Create user movie access with 48-hour expiry
	expiresAt := now.Add(48 * time.Hour)
	access := &orders.UserMovieAccess{
		UserExtID:       order.UserExtID,
		MovieID:         order.MovieID,
		OrderID:         order.ID,
		AccessGrantedAt: now,
		AccessExpiresAt: &expiresAt,
	}

	if err := h.orderRepo.CreateUserMovieAccess(access); err != nil {
		return fmt.Errorf("failed to create user movie access: %w", err)
	}

	log.Printf("[WEBHOOK] Created movie access for user %s, movie %d, expires at %s",
		order.UserExtID, order.MovieID, expiresAt.Format("2006-01-02 15:04:05"))

	return nil
}
