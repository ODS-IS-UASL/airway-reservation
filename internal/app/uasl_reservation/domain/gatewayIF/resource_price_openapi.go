package gatewayIF

import (
	"context"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

type ResourcePriceGatewayIF interface {
	GetResourcePriceList(ctx context.Context, baseURL string, req model.ResourcePriceListRequest) (*model.ResourcePriceList, error)
}
