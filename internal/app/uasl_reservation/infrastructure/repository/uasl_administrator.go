package repository

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/repositoryIF"
)

type uaslAdministratorRepository struct {
	db *gorm.DB
}

func NewUaslAdministratorRepository(db *gorm.DB) repositoryIF.UaslAdministratorRepositoryIF {
	return &uaslAdministratorRepository{db: db}
}

func (r *uaslAdministratorRepository) FindByExAdministratorIDs(ctx context.Context, exAdminIDs []string) ([]*model.UaslAdministrator, error) {
	if len(exAdminIDs) == 0 {
		return []*model.UaslAdministrator{}, nil
	}

	var administrators []*model.UaslAdministrator

	err := r.db.WithContext(ctx).
		Where("ex_administrator_id IN ?", exAdminIDs).
		Find(&administrators).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find uasl administrators: %w", err)
	}

	return administrators, nil
}

func (r *uaslAdministratorRepository) FindByExUaslIDs(ctx context.Context, exUaslIDs []string) ([]*model.UaslAdministrator, error) {

	if len(exUaslIDs) == 0 {
		return []*model.UaslAdministrator{}, nil
	}

	var administrators []*model.UaslAdministrator

	var subQueries []string
	var args []interface{}

	subQueryTemplate := `(
			CASE jsonb_typeof(external_services)
				WHEN 'array' THEN EXISTS (
					SELECT 1 FROM jsonb_array_elements(external_services) AS elem
					WHERE elem->>'exUaslId' = ?
				)
				WHEN 'object' THEN external_services->>'exUaslId' = ?
				ELSE false
			END
		)`

	for _, id := range exUaslIDs {
		subQueries = append(subQueries, subQueryTemplate)
		args = append(args, id, id)
	}

	fullCondition := strings.Join(subQueries, " OR ")

	db := r.db.WithContext(ctx).Where(fullCondition, args...)

	if err := db.Find(&administrators).Error; err != nil {
		return nil, fmt.Errorf("failed to find uasl administrators by ex_uasl_ids: %w", err)
	}

	return administrators, nil
}

func (r *uaslAdministratorRepository) FindInternalAdministrators(ctx context.Context) ([]*model.UaslAdministrator, error) {
	var administrators []*model.UaslAdministrator

	err := r.db.WithContext(ctx).
		Where("is_internal = ?", true).
		Find(&administrators).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find internal administrators: %w", err)
	}

	return administrators, nil
}

func (r *uaslAdministratorRepository) FindExternalAdministrators(ctx context.Context) ([]*model.UaslAdministrator, error) {
	var administrators []*model.UaslAdministrator

	err := r.db.WithContext(ctx).
		Where("is_internal = ?", false).
		Find(&administrators).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find external administrators: %w", err)
	}

	return administrators, nil
}

func (r *uaslAdministratorRepository) UpsertBatch(ctx context.Context, items []*model.UaslAdministrator) error {
	if len(items) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "ex_administrator_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"business_number", "name", "external_services", "is_internal", "updated_at",
			}),
		}).
		Create(&items)

	if result.Error != nil {
		return fmt.Errorf("failed to upsert uasl administrators: %w", result.Error)
	}

	return nil
}
