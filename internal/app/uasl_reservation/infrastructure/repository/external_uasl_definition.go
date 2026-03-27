package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/repositoryIF"
)

type externalUaslDefinitionRepository struct {
	db *gorm.DB
}

func NewExternalUaslDefinitionRepository(db *gorm.DB) repositoryIF.ExternalUaslDefinitionRepositoryIF {
	return &externalUaslDefinitionRepository{db: db}
}

func (r *externalUaslDefinitionRepository) FindByExSectionID(ctx context.Context, exSectionID string) (*model.ExternalUaslDefinition, error) {
	var definition model.ExternalUaslDefinition

	err := r.db.WithContext(ctx).
		Where("ex_uasl_section_id = ?", exSectionID).
		First(&definition).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("external uasl definition not found for ex_section_id=%s: %w", exSectionID, err)
		}
		return nil, fmt.Errorf("failed to find external uasl definition: %w", err)
	}

	return &definition, nil
}

func (r *externalUaslDefinitionRepository) FindByExSectionIDs(ctx context.Context, exSectionIDs []string) ([]*model.ExternalUaslDefinition, error) {
	if len(exSectionIDs) == 0 {
		return []*model.ExternalUaslDefinition{}, nil
	}

	var definitions []*model.ExternalUaslDefinition

	err := r.db.WithContext(ctx).
		Where("ex_uasl_section_id IN ?", exSectionIDs).
		Find(&definitions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find external uasl definitions: %w", err)
	}

	return definitions, nil
}

func (r *externalUaslDefinitionRepository) FindByExUaslIDs(ctx context.Context, exUaslIDs []string) ([]*model.ExternalUaslDefinition, error) {
	if len(exUaslIDs) == 0 {
		return []*model.ExternalUaslDefinition{}, nil
	}

	var definitions []*model.ExternalUaslDefinition

	err := r.db.WithContext(ctx).
		Where("ex_uasl_id IN ?", exUaslIDs).
		Find(&definitions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find external uasl definitions: %w", err)
	}

	return definitions, nil
}

func (r *externalUaslDefinitionRepository) UpsertBatch(ctx context.Context, items []*model.ExternalUaslDefinition) error {
	if len(items) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "ex_uasl_section_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"ex_uasl_id", "ex_administrator_id", "geometry", "point_ids",
				"flight_purpose", "price_info", "price_timezone", "price_version",
				"status", "updated_at",
			}),
		}).
		Create(&items)

	if result.Error != nil {
		return fmt.Errorf("failed to upsert external uasl definitions: %w", result.Error)
	}

	return nil
}

func (r *externalUaslDefinitionRepository) HasAnyExternalDefinitions(ctx context.Context) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ExternalUaslDefinition{}).
		Limit(1).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check uasl definition existence: %w", err)
	}

	return count > 0, nil
}

func (r *externalUaslDefinitionRepository) FindExistingExUaslIDs(ctx context.Context, exUaslIDs []string) ([]string, error) {
	if len(exUaslIDs) == 0 {
		return []string{}, nil
	}

	var existingIDs []string
	err := r.db.WithContext(ctx).
		Model(&model.ExternalUaslDefinition{}).
		Where("ex_uasl_id IN ?", exUaslIDs).
		Pluck("ex_uasl_id", &existingIDs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find existing ex_uasl_ids: %w", err)
	}

	return existingIDs, nil
}
