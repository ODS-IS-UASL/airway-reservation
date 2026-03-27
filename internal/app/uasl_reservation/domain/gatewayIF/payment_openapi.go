package gatewayIF

import (
	"context"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

type PaymentGatewayIF interface {
	CheckTransactionEligibility(ctx context.Context, req model.TransactionEligibilityRequest) (*model.TransactionEligibilityResponse, error)

	ConfirmTransaction(ctx context.Context, baseURL string, req model.TransactionConfirmRequest) (*model.TransactionConfirmResponse, error)
}
