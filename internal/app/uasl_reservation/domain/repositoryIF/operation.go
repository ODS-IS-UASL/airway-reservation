package repositoryIF

import "uasl-reservation/internal/app/uasl_reservation/domain/model"

type OperationRepositoryIF interface {
	FindByID(operationID string, operation *model.Operation) error
}
