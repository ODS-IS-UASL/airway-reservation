package repositoryIF

import (
	"context"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

type ExternalUaslDefinitionRepositoryIF interface {
	FindByExSectionID(ctx context.Context, exSectionID string) (*model.ExternalUaslDefinition, error)

	FindByExSectionIDs(ctx context.Context, exSectionIDs []string) ([]*model.ExternalUaslDefinition, error)

	FindByExUaslIDs(ctx context.Context, exUaslIDs []string) ([]*model.ExternalUaslDefinition, error)

	UpsertBatch(ctx context.Context, items []*model.ExternalUaslDefinition) error

	HasAnyExternalDefinitions(ctx context.Context) (bool, error)

	FindExistingExUaslIDs(ctx context.Context, exUaslIDs []string) ([]string, error)
}
