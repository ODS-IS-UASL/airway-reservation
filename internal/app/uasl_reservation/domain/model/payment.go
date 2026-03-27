package model

import (
	"time"
)

type TransactionEligibilityRequest struct {
	ProviderID        string
	ConsumerID        string
	PaymentServiceID  string
	TaxClassification string
	TaxRate           float64
	CompletedAt       time.Time
	Amount            int
}

type TransactionEligibilityResponse struct {
	Status string
	Detail string
}

type TransactionConfirmRequest struct {
	ProviderID        string    `json:"provider_id"`
	ConsumerID        string    `json:"consumer_id"`
	PaymentServiceID  string    `json:"payment_service_id"`
	TaxClassification string    `json:"tax_classification"`
	TaxRate           float64   `json:"tax_rate"`
	CompletedAt       time.Time `json:"completed_at"`
	Amount            int       `json:"amount"`
}

type TransactionConfirmResponse struct {
	Status string `json:"status"`
	Detail string `json:"detail"`
}
