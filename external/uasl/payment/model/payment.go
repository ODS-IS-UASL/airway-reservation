package model

import (
	"time"
)

type TransactionEligibilityRequest struct {
	ProviderID        string    `json:"provider_id"`
	ConsumerID        string    `json:"consumer_id"`
	PaymentServiceID  string    `json:"payment_service_id"`
	TaxClassification string    `json:"tax_classification"`
	TaxRate           float64   `json:"tax_rate"`
	CompletedAt       time.Time `json:"completed_at"`
	Amount            int       `json:"amount"`
}

type TransactionEligibilityResponse struct {
	Status string `json:"status"`
	Detail string `json:"detail"`
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
