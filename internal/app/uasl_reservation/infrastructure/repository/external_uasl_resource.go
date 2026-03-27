package repository

import (
	"context"
	"fmt"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/repositoryIF"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type externalUaslResourceRepository struct {
	db *gorm.DB
}

func NewExternalUaslResourceRepository(db *gorm.DB) repositoryIF.ExternalUaslResourceRepositoryIF {
	return &externalUaslResourceRepository{db: db}
}

func (r *externalUaslResourceRepository) FindByResourceIDsAndType(resourceIDs []string, resourceType model.ExternalResourceType) ([]*model.ExternalUaslResource, error) {
	if len(resourceIDs) == 0 {
		return []*model.ExternalUaslResource{}, nil
	}

	var resources []*model.ExternalUaslResource
	err := r.db.Where("ex_resource_id IN ? AND resource_type = ?", resourceIDs, resourceType).
		Find(&resources).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find external uasl resources: %w", err)
	}

	return resources, nil
}

func (r *externalUaslResourceRepository) FindByExUaslSectionIDs(exUaslSectionIDs []string) ([]*model.ExternalUaslResource, error) {
	if len(exUaslSectionIDs) == 0 {
		return []*model.ExternalUaslResource{}, nil
	}

	var resources []*model.ExternalUaslResource
	err := r.db.Where("ex_uasl_section_id IN ?", exUaslSectionIDs).
		Find(&resources).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find external uasl resources by section IDs: %w", err)
	}

	return resources, nil
}

func (r *externalUaslResourceRepository) UpdateAircraftInfoByExResourceID(exResourceID string, aircraftInfoJSON []byte, priceInfo model.ExternalResourcePriceInfoList) error {
	updates := map[string]interface{}{
		"estimated_price_per_minute": priceInfo,
	}
	if len(aircraftInfoJSON) > 0 {
		jsonData := datatypes.JSON(aircraftInfoJSON)
		updates["aircraft_info"] = &jsonData
	}
	result := r.db.Model(&model.ExternalUaslResource{}).
		Where("ex_resource_id = ?", exResourceID).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update aircraft_info for ex_resource_id=%s: %w", exResourceID, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("external uasl resource not found: ex_resource_id=%s", exResourceID)
	}
	return nil
}

func (r *externalUaslResourceRepository) UpsertBatch(ctx context.Context, items []*model.ExternalUaslResource) error {
	if len(items) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "ex_resource_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name", "resource_id", "ex_uasl_section_id", "resource_type",
				"organization_id", "estimated_price_per_minute", "aircraft_info", "updated_at",
			}),
		}).
		Create(&items)

	if result.Error != nil {
		return fmt.Errorf("failed to upsert external uasl resources: %w", result.Error)
	}

	return nil
}

func (r *externalUaslResourceRepository) HasAnyExternalResources(ctx context.Context) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ExternalUaslResource{}).
		Limit(1).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check external resources existence: %w", err)
	}

	return count > 0, nil
}

func (r *externalUaslResourceRepository) HasResourceByID(ctx context.Context, exResourceID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ExternalUaslResource{}).
		Where("ex_resource_id = ?", exResourceID).
		Limit(1).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check resource existence: %w", err)
	}

	return count > 0, nil
}

func (r *externalUaslResourceRepository) FindExistingExResourceIDs(ctx context.Context, exResourceIDs []string) ([]string, error) {
	if len(exResourceIDs) == 0 {
		return []string{}, nil
	}

	var existingIDs []string
	err := r.db.WithContext(ctx).
		Model(&model.ExternalUaslResource{}).
		Where("ex_resource_id IN ?", exResourceIDs).
		Pluck("ex_resource_id", &existingIDs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find existing ex_resource_ids: %w", err)
	}

	return existingIDs, nil
}
