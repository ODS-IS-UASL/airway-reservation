package repositoryIF

import (
	"context"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

type UaslAdministratorRepositoryIF interface {
	FindByExAdministratorIDs(ctx context.Context, exAdminIDs []string) ([]*model.UaslAdministrator, error)

	FindByExUaslIDs(ctx context.Context, exUaslIDs []string) ([]*model.UaslAdministrator, error)

	FindInternalAdministrators(ctx context.Context) ([]*model.UaslAdministrator, error)

	FindExternalAdministrators(ctx context.Context) ([]*model.UaslAdministrator, error)

	UpsertBatch(ctx context.Context, items []*model.UaslAdministrator) error
}
