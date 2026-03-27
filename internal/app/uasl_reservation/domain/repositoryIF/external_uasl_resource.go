package repositoryIF

import (
	"context"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

type ExternalUaslResourceRepositoryIF interface {
	FindByResourceIDsAndType(exResourceIDs []string, resourceType model.ExternalResourceType) ([]*model.ExternalUaslResource, error)

	FindByExUaslSectionIDs(exUaslSectionIDs []string) ([]*model.ExternalUaslResource, error)

	UpdateAircraftInfoByExResourceID(exResourceID string, aircraftInfoJSON []byte, priceInfo model.ExternalResourcePriceInfoList) error

	UpsertBatch(ctx context.Context, items []*model.ExternalUaslResource) error

	HasAnyExternalResources(ctx context.Context) (bool, error)

	HasResourceByID(ctx context.Context, exResourceID string) (bool, error)

	FindExistingExResourceIDs(ctx context.Context, exResourceIDs []string) ([]string, error)
}
