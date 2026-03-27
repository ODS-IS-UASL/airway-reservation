package mock

import (
	"context"
	"fmt"
	"strings"

	gatewayIF "uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/pkg/logger"

	"google.golang.org/grpc/metadata"
)

func logMockAuthorization(ctx context.Context, gatewayName string) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		logger.LogInfo("%s mock received no metadata", gatewayName)
		return
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		logger.LogInfo("%s mock received no Authorization header", gatewayName)
		return
	}

	token := authHeaders[0]
	if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = "Bearer " + token
	}
	logger.LogInfo("%s mock received Authorization header", "authorization", token)
}

type flightPlanInfoMockGateway struct{}

func NewFlightPlanInfoMockGateway() gatewayIF.FlightPlanInfoOpenAPIGatewayIF {
	return &flightPlanInfoMockGateway{}
}
func (g *flightPlanInfoMockGateway) Fetch(ctx context.Context, req model.FlightPlanInfoRequest) (model.FlightPlanInfoResponse, error) {
	logMockAuthorization(ctx, "flightPlanInfo")
	return model.FlightPlanInfoResponse{}, nil
}

type conformityAssessmentGatewayLocalMock struct{}

func NewConformityAssessmentGatewayLocalMock() gatewayIF.ConformityAssessmentGatewayIF {
	return &conformityAssessmentGatewayLocalMock{}
}
func (g *conformityAssessmentGatewayLocalMock) PostConformityAssessment(ctx context.Context, baseURL string, req model.ConformityAssessmentRequest) (*model.ConformityAssessmentResponse, error) {
	logMockAuthorization(ctx, "conformityAssessment")
	return &model.ConformityAssessmentResponse{}, nil
}

type ouranosProxyGatewayLocalMock struct{}

func NewOuranosProxyGatewayLocalMock() gatewayIF.OuranosProxyGatewayIF {
	return &ouranosProxyGatewayLocalMock{}
}
func (g *ouranosProxyGatewayLocalMock) CreateUaslReservation(ctx context.Context, baseURL string, domains []*model.UaslReservation, ports []model.PortReservationRequest, originAdministratorID string, originUaslID string, vehicleDetail *model.VehicleDetailInfo, ignoreFlightPlanConflict bool) (*model.UaslReservationResponse, error) {
	logMockAuthorization(ctx, "ouranosProxy")
	return &model.UaslReservationResponse{}, nil
}
func (g *ouranosProxyGatewayLocalMock) ConfirmUaslReservation(ctx context.Context, baseURL string, requestID string, isInterConnect bool) (*model.UaslReservationResponse, error) {
	logMockAuthorization(ctx, "ouranosProxy")
	return &model.UaslReservationResponse{}, nil
}
func (g *ouranosProxyGatewayLocalMock) CancelUaslReservation(ctx context.Context, baseURL string, requestID string, isInterConnect bool, action string) (*model.UaslReservationResponse, error) {
	logMockAuthorization(ctx, "ouranosProxy")
	return &model.UaslReservationResponse{}, nil
}
func (g *ouranosProxyGatewayLocalMock) DeleteUaslReservation(ctx context.Context, baseURL string, requestID string) error {
	logMockAuthorization(ctx, "ouranosProxy")
	return nil
}
func (g *ouranosProxyGatewayLocalMock) GetAvailability(ctx context.Context, baseURL string, sections []model.AvailabilitySection, vehicleIDs []string, portIDs []string) ([]model.AvailabilityItem, []model.VehicleAvailabilityItem, []model.PortAvailabilityItem, error) {
	logMockAuthorization(ctx, "ouranosProxy")
	return []model.AvailabilityItem{}, []model.VehicleAvailabilityItem{}, []model.PortAvailabilityItem{}, nil
}
func (g *ouranosProxyGatewayLocalMock) GetReservationsByRequestIDs(ctx context.Context, baseURL string, requestIDs []string) ([]model.ExternalReservationListItem, error) {
	logMockAuthorization(ctx, "ouranosProxy")
	return []model.ExternalReservationListItem{}, nil
}
func (g *ouranosProxyGatewayLocalMock) EstimateUaslReservation(ctx context.Context, baseURL string, req model.ExternalEstimateRequest) (*model.ExternalEstimateResponse, error) {
	logMockAuthorization(ctx, "ouranosProxy")
	return &model.ExternalEstimateResponse{}, nil
}
func (g *ouranosProxyGatewayLocalMock) SendReservationCompleted(ctx context.Context, baseURL string, req model.DestinationReservationNotification) error {
	logMockAuthorization(ctx, "ouranosProxy")
	return nil
}

type ouranosL3AuthGatewayLocalMock struct{}

func NewOuranosL3AuthGatewayLocalMock() gatewayIF.OuranosL3AuthGatewayIF {
	return &ouranosL3AuthGatewayLocalMock{}
}
func (g *ouranosL3AuthGatewayLocalMock) GetAccessToken(context.Context) (string, error) {
	return "mock-token", nil
}
func (g *ouranosL3AuthGatewayLocalMock) IntrospectToken(context.Context, string) error { return nil }

type ouranosDiscoveryGatewayLocalMock struct{}

func NewOuranosDiscoveryGatewayLocalMock() gatewayIF.OuranosDiscoveryGatewayIF {
	return &ouranosDiscoveryGatewayLocalMock{}
}
func (g *ouranosDiscoveryGatewayLocalMock) FindResourceFromDiscoveryService(ctx context.Context, req model.FindResourceRequest) ([]model.UaslServiceURL, error) {
	logMockAuthorization(ctx, "ouranosDiscovery")
	return []model.UaslServiceURL{}, nil
}

type paymentGatewayLocalMock struct{}

func NewPaymentGatewayLocalMock() gatewayIF.PaymentGatewayIF { return &paymentGatewayLocalMock{} }
func (g *paymentGatewayLocalMock) CheckTransactionEligibility(ctx context.Context, req model.TransactionEligibilityRequest) (*model.TransactionEligibilityResponse, error) {
	logMockAuthorization(ctx, "payment")
	return &model.TransactionEligibilityResponse{Status: "OK"}, nil
}
func (g *paymentGatewayLocalMock) ConfirmTransaction(ctx context.Context, baseURL string, req model.TransactionConfirmRequest) (*model.TransactionConfirmResponse, error) {
	logMockAuthorization(ctx, "payment")
	return &model.TransactionConfirmResponse{Status: "success"}, nil
}

type portOpenAPIMockGateway struct{}

func NewPortOpenAPIMockGateway() gatewayIF.PortOpenAPIGatewayIF { return &portOpenAPIMockGateway{} }
func (g *portOpenAPIMockGateway) ListReservations(ctx context.Context, baseURL string, req model.PortFetchRequest) ([]model.PortReservationInfo, error) {
	logMockAuthorization(ctx, "port")
	return []model.PortReservationInfo{}, nil
}
func (g *portOpenAPIMockGateway) Reserve(ctx context.Context, baseURL string, req model.PortReservationRequest) (model.ReservationHandle, error) {
	logMockAuthorization(ctx, "port")
	return model.NewReservationHandle("mock-port-reservation", model.ResourceTypePort, "")
}
func (g *portOpenAPIMockGateway) Cancel(ctx context.Context, baseURL string, handle model.ReservationHandle, operatorID string) error {
	logMockAuthorization(ctx, "port")
	return nil
}
func (g *portOpenAPIMockGateway) GetDronePortReservationDetail(ctx context.Context, baseURL string, reservationID string) (*model.PortReservationDetail, error) {
	logMockAuthorization(ctx, "port")
	return &model.PortReservationDetail{}, nil
}
func (g *portOpenAPIMockGateway) GetDronePortInfoDetail(ctx context.Context, portID string) (*model.DronePortInfo, error) {
	logMockAuthorization(ctx, "port")
	return &model.DronePortInfo{ActiveStatus: model.DronePortActiveStatusAvailable}, nil
}
func (g *portOpenAPIMockGateway) FetchDronePortList(ctx context.Context, baseURL string, includeHidden bool) ([]*model.ExternalUaslResource, error) {
	logMockAuthorization(ctx, "port")
	return []*model.ExternalUaslResource{}, nil
}

type resourcePriceGatewayLocalMock struct{}

func NewResourcePriceGatewayLocalMock() gatewayIF.ResourcePriceGatewayIF {
	return &resourcePriceGatewayLocalMock{}
}
func (g *resourcePriceGatewayLocalMock) GetResourcePriceList(ctx context.Context, baseURL string, req model.ResourcePriceListRequest) (*model.ResourcePriceList, error) {
	logMockAuthorization(ctx, "resourcePrice")
	return &model.ResourcePriceList{}, nil
}

type uaslDesignOpenAPIMockGateway struct{}

func NewUaslDesignOpenAPIMockGateway() gatewayIF.UaslDesignGatewayIF {
	return &uaslDesignOpenAPIMockGateway{}
}
func (g *uaslDesignOpenAPIMockGateway) FetchUaslList(ctx context.Context, baseURL string, includeHidden bool) (*model.UaslBulkData, error) {
	logMockAuthorization(ctx, "uaslDesign")
	return &model.UaslBulkData{}, nil
}
func (g *uaslDesignOpenAPIMockGateway) FetchUaslByID(ctx context.Context, baseURL string, uaslID string, includeHidden bool) (*model.UaslBulkData, error) {
	logMockAuthorization(ctx, "uaslDesign")
	if uaslID == "" {
		return nil, fmt.Errorf("uaslID is required")
	}
	return &model.UaslBulkData{}, nil
}

type vehicleOpenAPIMockGateway struct{}

func NewVehicleOpenAPIMockGateway() gatewayIF.VehicleOpenAPIGatewayIF {
	return &vehicleOpenAPIMockGateway{}
}
func (g *vehicleOpenAPIMockGateway) ListReservations(ctx context.Context, baseURL string, req model.VehicleFetchRequest) ([]model.VehicleReservationInfo, error) {
	logMockAuthorization(ctx, "vehicle")
	return []model.VehicleReservationInfo{}, nil
}
func (g *vehicleOpenAPIMockGateway) Reserve(ctx context.Context, baseURL string, req model.VehicleReservationRequest) (model.ReservationHandle, error) {
	logMockAuthorization(ctx, "vehicle")
	return model.NewReservationHandle("mock-vehicle-reservation", model.ResourceTypeVehicle, "")
}
func (g *vehicleOpenAPIMockGateway) Cancel(ctx context.Context, baseURL string, handle model.ReservationHandle, operatorID string) error {
	logMockAuthorization(ctx, "vehicle")
	return nil
}
func (g *vehicleOpenAPIMockGateway) GetAircraftReservationDetail(ctx context.Context, baseURL string, reservationID string) (*model.VehicleReservationDetail, error) {
	logMockAuthorization(ctx, "vehicle")
	return &model.VehicleReservationDetail{}, nil
}
func (g *vehicleOpenAPIMockGateway) GetAircraftInfoDetail(ctx context.Context, baseURL string, aircraftID string, includeHidden bool) (*model.AircraftInfoDetail, error) {
	logMockAuthorization(ctx, "vehicle")
	return &model.AircraftInfoDetail{}, nil
}
func (g *vehicleOpenAPIMockGateway) FetchAircraftList(ctx context.Context, baseURL string, includeHidden bool) ([]*model.ExternalUaslResource, error) {
	logMockAuthorization(ctx, "vehicle")
	return []*model.ExternalUaslResource{}, nil
}
