package payment

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
)

// PaymentService defines the interface for payment operations
type PaymentService interface {
	CreateTransaction(orderID int64, amount float64, userEmail, userName string) (string, string, error)
	VerifySignature(orderID, statusCode, grossAmount, serverKey string, signatureKey string) bool
}

type midtransService struct {
	client       snap.Client
	serverKey    string
	isProduction bool
}

// NewMidtransService creates a new Midtrans payment service
func NewMidtransService(serverKey, clientKey string, isProduction bool) PaymentService {
	var client snap.Client
	client.New(serverKey, midtrans.Sandbox)

	if isProduction {
		client.New(serverKey, midtrans.Production)
	}

	return &midtransService{
		client:       client,
		serverKey:    serverKey,
		isProduction: isProduction,
	}
}

// CreateTransaction creates a new payment transaction with Midtrans
func (s *midtransService) CreateTransaction(orderID int64, amount float64, userEmail, userName string) (string, string, error) {
	// Generate unique order ID for Midtrans
	orderIDStr := fmt.Sprintf("ORD-%d", orderID)

	// Create Snap request
	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  orderIDStr,
			GrossAmt: int64(amount),
		},
		CustomerDetail: &midtrans.CustomerDetails{
			Email: userEmail,
			FName: userName,
		},
		EnabledPayments: snap.AllSnapPaymentType,
		Items: &[]midtrans.ItemDetails{
			{
				ID:    orderIDStr,
				Price: int64(amount),
				Qty:   1,
				Name:  "Movie Rental",
			},
		},
	}

	// Create transaction
	snapResp, midtransErr := s.client.CreateTransaction(req)

	if midtransErr != nil {
		return "", "", fmt.Errorf("failed to create midtrans transaction: %w", midtransErr)
	}

	if snapResp == nil {
		return "", "", fmt.Errorf("midtrans returned nil response")
	}

	// Validate response
	if snapResp.Token == "" {
		return "", "", fmt.Errorf("midtrans returned empty token")
	}
	if snapResp.RedirectURL == "" {
		return "", "", fmt.Errorf("midtrans returned empty redirect URL")
	}

	return snapResp.RedirectURL, snapResp.Token, nil
}

// VerifySignature verifies the webhook signature from Midtrans
// Formula: SHA512(order_id+status_code+gross_amount+ServerKey)
func (s *midtransService) VerifySignature(orderID, statusCode, grossAmount, serverKey string, signatureKey string) bool {
	// Create signature string
	signatureString := orderID + statusCode + grossAmount + serverKey

	// Hash with SHA512
	hash := sha512.New()
	hash.Write([]byte(signatureString))
	expectedSignature := hex.EncodeToString(hash.Sum(nil))

	return expectedSignature == signatureKey
}
