package gatewayIF

import (
	"context"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

type OuranosProxyGatewayIF interface {
	CreateUaslReservation(ctx context.Context, baseURL string, childDomains []*model.UaslReservation, ports []model.PortReservationRequest, originAdministratorID string, originUaslID string, vehicleDetail *model.VehicleDetailInfo, ignoreFlightPlanConflict bool) (*model.UaslReservationResponse, error)

	ConfirmUaslReservation(ctx context.Context, baseURL string, requestID string, isInterConnect bool) (*model.UaslReservationResponse, error)

	CancelUaslReservation(ctx context.Context, baseURL string, requestID string, isInterConnect bool, action string) (*model.UaslReservationResponse, error)

	DeleteUaslReservation(ctx context.Context, baseURL string, requestID string) error

	GetAvailability(ctx context.Context, baseURL string, sections []model.AvailabilitySection, vehicleIDs []string, portIDs []string) ([]model.AvailabilityItem, []model.VehicleAvailabilityItem, []model.PortAvailabilityItem, error)

	GetReservationsByRequestIDs(ctx context.Context, baseURL string, requestIDs []string) ([]model.ExternalReservationListItem, error)

	EstimateUaslReservation(ctx context.Context, baseURL string, req model.ExternalEstimateRequest) (*model.ExternalEstimateResponse, error)

	SendReservationCompleted(ctx context.Context, baseURL string, payload model.DestinationReservationNotification) error
}
