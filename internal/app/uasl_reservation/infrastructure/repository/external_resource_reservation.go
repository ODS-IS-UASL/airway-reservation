package repository

import (
	"fmt"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/repositoryIF"
	"uasl-reservation/internal/pkg/value"

	"gorm.io/gorm"
)

type externalResourceReservationRepository struct {
	db *gorm.DB
}

func NewExternalResourceReservationRepository(db *gorm.DB) repositoryIF.ExternalResourceReservationRepositoryIF {
	return &externalResourceReservationRepository{db: db}
}

func (repo *externalResourceReservationRepository) FindByRequestID(requestID value.ModelID) ([]model.ExternalResourceReservation, error) {
	reservations := make([]model.ExternalResourceReservation, 0)

	err := repo.db.
		Table("? AS err", gorm.Expr((&model.ExternalResourceReservation{}).TableName())).
		Select(`err.*,
			(SELECT name
			 FROM ?
			 WHERE ex_resource_id = err.ex_resource_id
			 AND resource_type::text = err.resource_type::text
			 LIMIT 1) as resource_name`, gorm.Expr((&model.ExternalUaslResource{}).TableName())).
		Where("err.request_id = ?", requestID.ToString()).
		Scan(&reservations).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find external resource reservations with resource names: %w", err)
	}

	return reservations, nil
}

func (repo *externalResourceReservationRepository) FindByExReservationIDs(exReservationIDs []string) ([]model.ExternalResourceReservation, error) {
	if len(exReservationIDs) == 0 {
		return []model.ExternalResourceReservation{}, nil
	}

	reservations := make([]model.ExternalResourceReservation, 0)

	err := repo.db.
		Table("? AS err", gorm.Expr((&model.ExternalResourceReservation{}).TableName())).
		Select(`err.*,
			(SELECT name
			 FROM ?
			 WHERE ex_resource_id = err.ex_resource_id
			 AND resource_type::text = err.resource_type::text
			 LIMIT 1) as resource_name`, gorm.Expr((&model.ExternalUaslResource{}).TableName())).
		Where("err.ex_reservation_id IN ?", exReservationIDs).
		Scan(&reservations).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find external resource reservations by ex_reservation_ids: %w", err)
	}

	return reservations, nil
}

func (repo *externalResourceReservationRepository) InsertOne(
	reservation *model.ExternalResourceReservation,
) (*model.ExternalResourceReservation, error) {
	if err := reservation.Validate(); err != nil {
		return nil, fmt.Errorf("invalid external resource reservation: %w", err)
	}

	if err := repo.db.Create(reservation).Error; err != nil {
		return nil, fmt.Errorf("failed to insert external resource reservation: %w", err)
	}

	return reservation, nil
}

func (repo *externalResourceReservationRepository) InsertBatch(
	reservations []*model.ExternalResourceReservation,
) ([]*model.ExternalResourceReservation, error) {
	if len(reservations) == 0 {
		return reservations, nil
	}

	for i, reservation := range reservations {
		if err := reservation.Validate(); err != nil {
			return nil, fmt.Errorf("invalid external resource reservation at index %d: %w", i, err)
		}
	}

	if err := repo.db.Create(reservations).Error; err != nil {
		return nil, fmt.Errorf("failed to batch insert external resource reservations: %w", err)
	}

	return reservations, nil
}

func (repo *externalResourceReservationRepository) UpdateBatch(
	reservations []*model.ExternalResourceReservation,
) ([]*model.ExternalResourceReservation, error) {
	if len(reservations) == 0 {
		return reservations, nil
	}

	for i, reservation := range reservations {
		if err := reservation.Validate(); err != nil {
			return nil, fmt.Errorf("invalid external resource reservation at index %d: %w", i, err)
		}
	}

	for _, reservation := range reservations {
		if err := repo.db.Save(reservation).Error; err != nil {
			return nil, fmt.Errorf("failed to update external resource reservation: %w", err)
		}
	}

	return reservations, nil
}

func (repo *externalResourceReservationRepository) DeleteByExReservationID(
	exReservationID string,
) error {
	result := repo.db.
		Where("ex_reservation_id = ?", exReservationID).
		Delete(&model.ExternalResourceReservation{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete external resource reservation by external reservation ID: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no external resource reservation found for external reservation ID: %s", exReservationID)
	}

	return nil
}

func (repo *externalResourceReservationRepository) DeleteByRequestID(requestID value.ModelID) error {
	return repo.db.Where("request_id = ?", requestID.ToString()).Delete(&model.ExternalResourceReservation{}).Error
}
