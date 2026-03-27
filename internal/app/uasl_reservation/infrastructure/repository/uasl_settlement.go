package repository

import (
	"errors"
	"fmt"
	"time"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/repositoryIF"
	"uasl-reservation/internal/pkg/logger"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type uaslSettlementRepository struct {
	db *gorm.DB
}

func NewUaslSettlementRepository(db *gorm.DB) repositoryIF.UaslSettlementRepositoryIF {
	return &uaslSettlementRepository{db}
}

func (r *uaslSettlementRepository) UpsertBatch(settlements []*model.UaslSettlement) error {
	if len(settlements) == 0 {
		return nil
	}

	logger.LogInfo("UpsertBatch: upserting %d UaslSettlement records", len(settlements))
	if err := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "ex_administrator_id"},
			{Name: "operator_id"},
			{Name: "target_year_month"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"uasl_reservation_ids",
			"total_amount",
			"tax_rate",
			"updated_at",
		}),
		Where: clause.Where{
			Exprs: []clause.Expression{
				clause.Expr{SQL: "uasl_reservation.uasl_settlements.submitted_at IS NULL"},
			},
		},
	}).Create(&settlements).Error; err != nil {
		logger.LogError("UpsertBatch: failed to upsert UaslSettlement records: %v", err)
		return err
	}
	return nil
}

func (r *uaslSettlementRepository) UpdatePaymentConfirmedAt(id string, confirmedAt time.Time) error {
	logger.LogInfo("UpdatePaymentConfirmedAt: updating settlement id=%s", id)
	result := r.db.Model(&model.UaslSettlement{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"payment_confirmed_at": confirmedAt,
			"updated_at":           time.Now().UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected < 1 {
		return fmt.Errorf("settlement with id %s does not exist", id)
	}
	return nil
}

func (r *uaslSettlementRepository) UpdateSubmittedAt(id string, submittedAt time.Time) error {
	logger.LogInfo("UpdateSubmittedAt: updating settlement id=%s", id)
	result := r.db.Model(&model.UaslSettlement{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"submitted_at": submittedAt,
			"updated_at":   time.Now().UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected < 1 {
		return fmt.Errorf("settlement with id %s does not exist", id)
	}
	return nil
}

func (r *uaslSettlementRepository) FindByTargetYearMonth(yearMonth time.Time) ([]*model.UaslSettlement, error) {
	var settlements []*model.UaslSettlement
	if err := r.db.Where("target_year_month = ?", yearMonth).Find(&settlements).Error; err != nil {
		return nil, err
	}
	return settlements, nil
}

func (r *uaslSettlementRepository) FindUnsubmittedByTargetYearMonth(yearMonth time.Time) ([]*model.UaslSettlement, error) {
	var settlements []*model.UaslSettlement
	if err := r.db.Where("target_year_month = ? AND submitted_at IS NULL", yearMonth).Find(&settlements).Error; err != nil {
		return nil, err
	}
	return settlements, nil
}

func (r *uaslSettlementRepository) FindByID(id string) (*model.UaslSettlement, error) {
	var settlement model.UaslSettlement
	if err := r.db.First(&settlement, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &settlement, nil
}
