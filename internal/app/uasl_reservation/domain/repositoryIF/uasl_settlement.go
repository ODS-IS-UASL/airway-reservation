package repositoryIF

import (
	"time"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

type UaslSettlementRepositoryIF interface {
	UpsertBatch(settlements []*model.UaslSettlement) error
	UpdatePaymentConfirmedAt(id string, confirmedAt time.Time) error
	UpdateSubmittedAt(id string, submittedAt time.Time) error
	FindByTargetYearMonth(yearMonth time.Time) ([]*model.UaslSettlement, error)
	FindUnsubmittedByTargetYearMonth(yearMonth time.Time) ([]*model.UaslSettlement, error)
	FindByID(id string) (*model.UaslSettlement, error)
}
