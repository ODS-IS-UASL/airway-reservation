package gatewayIF

import (
	"context"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

type VehicleOpenAPIGatewayIF interface {
	ListReservations(ctx context.Context, baseURL string, req model.VehicleFetchRequest) ([]model.VehicleReservationInfo, error)

	Reserve(ctx context.Context, baseURL string, req model.VehicleReservationRequest) (model.ReservationHandle, error)

	Cancel(ctx context.Context, baseURL string, handle model.ReservationHandle, operatorID string) error

	GetAircraftReservationDetail(ctx context.Context, baseURL string, aircraftID string) (*model.VehicleReservationDetail, error)

	GetAircraftInfoDetail(ctx context.Context, baseURL string, aircraftID string, isRequiredPriceInfo bool) (*model.AircraftInfoDetail, error)

	FetchAircraftList(ctx context.Context, baseURL string, isRequiredPriceInfo bool) ([]*model.ExternalUaslResource, error)
}
