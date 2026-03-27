package converter

import (
	"os"
	"time"

	ouranosModel "uasl-reservation/external/uasl/payment/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

const defaultPaymentServiceID = "169f5226-4644-4d1e-ac36-14999e73601f"

type PaymentConverter struct{}

func NewPaymentConverter() *PaymentConverter {
	return &PaymentConverter{}
}

func (c *PaymentConverter) ToTransactionEligibilityRequest(
	providerID string,
	consumerID string,
	amount int,
) model.TransactionEligibilityRequest {
	paymentServiceID := os.Getenv("PAYMENT_SERVICE_ID")
	if paymentServiceID == "" {
		paymentServiceID = defaultPaymentServiceID
	}

	return model.TransactionEligibilityRequest{
		ProviderID:        providerID,
		ConsumerID:        consumerID,
		PaymentServiceID:  paymentServiceID,
		TaxClassification: "taxable",
		TaxRate:           0.1,
		CompletedAt:       time.Now(),
		Amount:            amount,
	}
}

func (c *PaymentConverter) ToExternalTransactionEligibilityRequest(
	req model.TransactionEligibilityRequest,
) ouranosModel.TransactionEligibilityRequest {
	return ouranosModel.TransactionEligibilityRequest{
		ProviderID:        req.ProviderID,
		ConsumerID:        req.ConsumerID,
		PaymentServiceID:  req.PaymentServiceID,
		TaxClassification: req.TaxClassification,
		TaxRate:           req.TaxRate,
		CompletedAt:       req.CompletedAt,
		Amount:            req.Amount,
	}
}

func (c *PaymentConverter) ToInternalTransactionEligibilityResponse(
	resp ouranosModel.TransactionEligibilityResponse,
) model.TransactionEligibilityResponse {
	return model.TransactionEligibilityResponse{
		Status: resp.Status,
		Detail: resp.Detail,
	}
}

func (c *PaymentConverter) ToExternalTransactionConfirmRequest(
	req model.TransactionConfirmRequest,
) ouranosModel.TransactionConfirmRequest {
	return ouranosModel.TransactionConfirmRequest{
		ProviderID:        req.ProviderID,
		ConsumerID:        req.ConsumerID,
		PaymentServiceID:  req.PaymentServiceID,
		TaxClassification: req.TaxClassification,
		TaxRate:           req.TaxRate,
		CompletedAt:       req.CompletedAt,
		Amount:            req.Amount,
	}
}

func (c *PaymentConverter) ToInternalTransactionConfirmResponse(
	resp ouranosModel.TransactionConfirmResponse,
) model.TransactionConfirmResponse {
	return model.TransactionConfirmResponse{
		Status: resp.Status,
		Detail: resp.Detail,
	}
}
