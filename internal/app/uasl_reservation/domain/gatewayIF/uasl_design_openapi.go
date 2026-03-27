package gatewayIF

import (
	"context"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

type UaslDesignGatewayIF interface {
	FetchUaslList(ctx context.Context, baseURL string, isInternal bool) (*model.UaslBulkData, error)

	FetchUaslByID(ctx context.Context, baseURL string, uaslID string, isInternal bool) (*model.UaslBulkData, error)
}
