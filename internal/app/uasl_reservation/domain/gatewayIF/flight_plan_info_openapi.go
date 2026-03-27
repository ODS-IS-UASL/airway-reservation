package gatewayIF

import (
	"context"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

type FlightPlanInfoOpenAPIGatewayIF interface {
	Fetch(ctx context.Context, req model.FlightPlanInfoRequest) (model.FlightPlanInfoResponse, error)
}
