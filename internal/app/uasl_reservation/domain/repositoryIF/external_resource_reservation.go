package repositoryIF

import (
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/pkg/value"
)

type ExternalResourceReservationRepositoryIF interface {
	FindByRequestID(requestID value.ModelID) ([]model.ExternalResourceReservation, error)

	FindByExReservationIDs(exReservationIDs []string) ([]model.ExternalResourceReservation, error)

	InsertOne(reservation *model.ExternalResourceReservation) (*model.ExternalResourceReservation, error)

	InsertBatch(reservations []*model.ExternalResourceReservation) ([]*model.ExternalResourceReservation, error)

	UpdateBatch(reservations []*model.ExternalResourceReservation) ([]*model.ExternalResourceReservation, error)

	DeleteByExReservationID(exReservationID string) error

	DeleteByRequestID(requestID value.ModelID) error
}
