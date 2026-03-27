package gatewayIF

import (
	"context"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

type OuranosDiscoveryGatewayIF interface {
	FindResourceFromDiscoveryService(ctx context.Context, req model.FindResourceRequest) ([]model.UaslServiceURL, error)
}
