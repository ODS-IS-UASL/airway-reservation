package gatewayIF

import (
	"context"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

type ConformityAssessmentGatewayIF interface {
	PostConformityAssessment(ctx context.Context, baseURL string, req model.ConformityAssessmentRequest) (*model.ConformityAssessmentResponse, error)
}
