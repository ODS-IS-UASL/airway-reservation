package gatewayIF

import (
	"context"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

type PortOpenAPIGatewayIF interface {
	ListReservations(ctx context.Context, baseURL string, req model.PortFetchRequest) ([]model.PortReservationInfo, error)

	Reserve(ctx context.Context, baseURL string, req model.PortReservationRequest) (model.ReservationHandle, error)

	Cancel(ctx context.Context, baseURL string, handle model.ReservationHandle, operatorID string) error

	GetDronePortReservationDetail(ctx context.Context, baseURL string, portID string) (*model.PortReservationDetail, error)

	GetDronePortInfoDetail(ctx context.Context, dronePortID string) (*model.DronePortInfo, error)

	FetchDronePortList(ctx context.Context, baseURL string, isRequiredPriceInfo bool) ([]*model.ExternalUaslResource, error)
}
