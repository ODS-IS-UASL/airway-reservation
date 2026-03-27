package repository

import (
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/repositoryIF"

	"gorm.io/gorm"
)

type operationRepository struct {
	db *gorm.DB
}

func NewOperationRepository(db *gorm.DB) repositoryIF.OperationRepositoryIF {
	return &operationRepository{db: db}
}

func (r *operationRepository) FindByID(operationID string, operation *model.Operation) error {
	return r.db.First(operation, "id = ?", operationID).Error
}
