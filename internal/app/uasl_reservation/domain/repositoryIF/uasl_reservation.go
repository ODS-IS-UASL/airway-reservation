package repositoryIF

import (
	"context"
	"time"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/pkg/value"
)

type UaslReservationRepositoryIF interface {
	FetchAll(uaslReservations *[]model.UaslReservation) error
	FindReservedByMonth(yearMonth time.Time) ([]*model.UaslReservation, error)
	FindByID(uaslReservationID value.ModelID) (*model.UaslReservation, error)
	FindByRequestID(requestID value.ModelID) (*model.UaslReservation, []*model.UaslReservation, error)
	FindByUaslSectionIDs(uaslSectionIDs []string, baseAt *value.NullTime) ([]model.UaslReservation, error)
	InsertOne(uaslReservation *model.UaslReservation) (*model.UaslReservation, error)
	InsertBatch(uaslReservations []*model.UaslReservation) ([]*model.UaslReservation, error)
	UpdateOne(uaslReservation *model.UaslReservation) (*model.UaslReservation, error)
	UpdateBatch(uaslReservations []*model.UaslReservation) ([]*model.UaslReservation, error)
	ExpirePendingParentsAndChildren(expiredBeforeUTC time.Time, limit int) ([]model.ExpiredUaslReservation, error)
	ExpirePendingByRequestID(requestID value.ModelID, expiredBeforeUTC time.Time) (*model.ExpiredUaslReservation, error)
	DeleteOne(uaslReservationID value.ModelID) (value.ModelID, error)
	DeleteByRequestID(requestID value.ModelID) (int, error)
	CheckConflictReservations(queries []model.UaslCheckRequest) ([]model.UaslReservation, error)
	ListByRequestIDs(requestIDs []value.ModelID) ([]*model.UaslReservationListItem, error)
	ListByOperator(
		ctx context.Context,
		operatorID string,
		exAdministratorIDs []string,
		page int32,
		perPage int32,
	) ([]*model.UaslReservationListItem, int64, error)
	ListAdmin(
		ctx context.Context,
		page int32,
		perPage int32,
	) ([]*model.UaslReservationListItem, int64, error)

	FindChildrenByParentID(parentID value.ModelID) ([]*model.UaslReservation, error)

	FindParentsByRequestID(requestID value.ModelID) ([]*model.UaslReservation, error)
}
