package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	ouranosModel "uasl-reservation/external/uasl/l2/model"
	conv "uasl-reservation/internal/app/uasl_reservation/application/converter"
	"uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	dmodel "uasl-reservation/internal/app/uasl_reservation/domain/model"
	mock "uasl-reservation/internal/app/uasl_reservation/infrastructure/gateway/mock"
	httpClient "uasl-reservation/internal/pkg/http"
	"uasl-reservation/internal/pkg/logger"
)

const ouranosProxyDefaultTimeout = 30 * time.Second

const (
	pathUaslReservations             = "/v1/uaslReservations"
	pathUaslReservationsByID         = "/v1/uaslReservations/%s"
	pathUaslReservationsCancel       = "/v1/uaslReservations/%s/cancel"
	pathAdminUaslReservationsRescind = "/v1/admin/uaslReservations/%s/rescind"
	pathUaslReservationsDelete       = "/v1/uaslReservations/%s"
	pathUaslReservationsConfirm      = "/v1/uaslReservations/%s/confirm"
	pathUaslReservationsAvailability = "/v1/uaslReservations/availability"
	pathUaslReservationsEstimate     = "/v1/uaslReservations/estimate"
	pathUaslReservationsCompleted    = "/v1/uaslReservations/completed"
)

type ouranosProxyGateway struct {
	apiKey  string
	baseURL string
	client  *httpClient.HttpClient
}

func NewOuranosProxyGateway() gatewayIF.OuranosProxyGatewayIF {

	if os.Getenv("APP_ENV") == "local" {
		logger.LogInfo("Using Ouranos proxy mock (APP_ENV=poc)")
		return mock.NewOuranosProxyGatewayLocalMock()
	}

	base := os.Getenv("L2_BASE_URL")
	apiKey := os.Getenv("RESERVATION_API_KEY")

	return &ouranosProxyGateway{
		baseURL: base,
		apiKey:  apiKey,
		client:  httpClient.NewHttpClientWithTimeout(ouranosProxyDefaultTimeout),
	}
}

func getAuthorizationFromContext(ctx context.Context) (string, bool) {
	type contextKey string
	const authorizationKey contextKey = "authorization"
	token, ok := ctx.Value(authorizationKey).(string)
	if !ok || token == "" {
		return "", false
	}

	if !strings.HasPrefix(token, "Bearer ") && !strings.HasPrefix(token, "bearer ") {
		token = "Bearer " + token
	}
	return token, true
}

func (g *ouranosProxyGateway) resolveBaseURL(baseURL string) string {
	if baseURL != "" {
		return baseURL
	}

	if g.baseURL != "" {
		return g.baseURL
	}

	return ""
}

func buildRequestURL(base, path string) string {
	return fmt.Sprintf("%s%s", strings.TrimRight(base, "/"), path)
}

func (g *ouranosProxyGateway) newJSONRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewBuffer(body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if authToken, ok := getAuthorizationFromContext(ctx); ok {
		httpReq.Header.Set("Authorization", authToken)
	}
	if g.apiKey != "" {
		httpReq.Header.Set("API-Key", g.apiKey)
	}
	httpReq.Header.Set("X-TrackingId", uuid.New().String())

	return httpReq, nil
}

func (g *ouranosProxyGateway) doRequest(httpReq *http.Request) (*http.Response, error) {
	if g.client != nil {
		return g.client.DoRequest(&httpClient.HttpRequest{Request: httpReq})
	}
	hc := httpClient.NewHttpClientWithTimeout(ouranosProxyDefaultTimeout)
	return hc.DoRequest(&httpClient.HttpRequest{Request: httpReq})
}

func (g *ouranosProxyGateway) CreateUaslReservation(ctx context.Context, baseURL string, childDomains []*dmodel.UaslReservation, ports []dmodel.PortReservationRequest, originAdministratorID string, originUaslID string, vehicleDetail *dmodel.VehicleDetailInfo, ignoreFlightPlanConflict bool) (*dmodel.UaslReservationResponse, error) {
	base := g.resolveBaseURL(baseURL)

	extReq := conv.ToExternalUaslReservationRequest(childDomains, ports, originAdministratorID, originUaslID, vehicleDetail, ignoreFlightPlanConflict)

	reqBody, err := json.Marshal(extReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	httpReq, err := g.newJSONRequest(ctx, http.MethodPost, buildRequestURL(base, pathUaslReservations), reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call ouranos proxy API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		var errBody map[string]interface{}
		_ = json.Unmarshal(bodyBytes, &errBody)
		return &dmodel.UaslReservationResponse{Error: &dmodel.ResponseError{
			Code:    fmt.Sprintf("http:%d", resp.StatusCode),
			Message: string(bodyBytes),
			Details: errBody,
		}}, nil
	}

	var extResp ouranosModel.UaslReservationResponse

	if err := json.NewDecoder(resp.Body).Decode(&extResp); err != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return &dmodel.UaslReservationResponse{Error: &dmodel.ResponseError{
			Code:    "decode_error",
			Message: err.Error(),
			Details: map[string]interface{}{"body": string(bodyBytes)},
		}}, nil
	}

	return conv.ToUaslReservationResponse(&extResp), nil
}

func (g *ouranosProxyGateway) CancelUaslReservation(ctx context.Context, baseURL string, requestID string, isInterConnect bool, action string) (*dmodel.UaslReservationResponse, error) {
	base := g.resolveBaseURL(baseURL)

	cancelReq := map[string]interface{}{
		"isInterConnect": isInterConnect,
	}

	reqBody, err := json.Marshal(cancelReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cancel request: %w", err)
	}

	path := fmt.Sprintf(pathUaslReservationsCancel, requestID)
	if strings.EqualFold(action, "RESCINDED") {
		path = fmt.Sprintf(pathAdminUaslReservationsRescind, requestID)
	}
	httpReq, err := g.newJSONRequest(ctx, http.MethodPut, buildRequestURL(base, path), reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call ouranos proxy API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ouranos proxy API returned status %d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var extResp ouranosModel.UaslReservationResponse
	if err := json.NewDecoder(resp.Body).Decode(&extResp); err != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return &dmodel.UaslReservationResponse{Error: &dmodel.ResponseError{
			Code:    "decode_error",
			Message: err.Error(),
			Details: map[string]interface{}{"body": string(bodyBytes)},
		}}, nil
	}

	return conv.ToUaslReservationResponse(&extResp), nil
}

func (g *ouranosProxyGateway) DeleteUaslReservation(ctx context.Context, baseURL string, requestID string) error {
	base := g.resolveBaseURL(baseURL)

	path := fmt.Sprintf(pathUaslReservationsDelete, requestID)
	httpReq, err := g.newJSONRequest(ctx, http.MethodDelete, buildRequestURL(base, path), nil)
	if err != nil {
		return err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call ouranos proxy API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ouranos proxy API returned status %d body=%s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (g *ouranosProxyGateway) GetAvailability(ctx context.Context, baseURL string, sections []dmodel.AvailabilitySection, vehicleIDs []string, portIDs []string) ([]dmodel.AvailabilityItem, []dmodel.VehicleAvailabilityItem, []dmodel.PortAvailabilityItem, error) {
	base := g.resolveBaseURL(baseURL)

	extReq := conv.ToExternalAvailabilityRequest(sections, vehicleIDs, portIDs)

	reqBody, err := json.Marshal(extReq)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to marshal availability request: %w", err)
	}

	httpReq, err := g.newJSONRequest(ctx, http.MethodPost, buildRequestURL(base, pathUaslReservationsAvailability), reqBody)
	if err != nil {
		return nil, nil, nil, err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to call ouranos proxy availability API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, nil, nil, fmt.Errorf("ouranos proxy availability API returned status %d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var extResp ouranosModel.AvailabilityResponse
	if err := json.NewDecoder(resp.Body).Decode(&extResp); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode availability response: %w", err)
	}

	sectionItems, vehicleItems, portItems := conv.ToAvailabilityItems(&extResp.Result)
	return sectionItems, vehicleItems, portItems, nil
}

func (g *ouranosProxyGateway) GetReservationsByRequestIDs(ctx context.Context, baseURL string, requestIDs []string) ([]dmodel.ExternalReservationListItem, error) {
	base := g.resolveBaseURL(baseURL)

	internalReq := dmodel.ExternalReservationSearchRequest{
		RequestIDs: requestIDs,
	}
	extReq := conv.ToExternalReservationSearchRequest(internalReq)

	reqBody, err := json.Marshal(extReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal reservation search request: %w", err)
	}

	httpReq, err := g.newJSONRequest(ctx, http.MethodPost, buildRequestURL(base, pathUaslReservationsAvailability), reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call ouranos proxy reservation search API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ouranos proxy reservation search API returned status %d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var extResp ouranosModel.SearchByRequestIDsResponse
	if err := json.NewDecoder(resp.Body).Decode(&extResp); err != nil {
		return nil, fmt.Errorf("failed to decode reservation search response: %w", err)
	}

	return conv.ToInternalExternalReservationListItems(extResp.Result), nil
}

func (g *ouranosProxyGateway) ConfirmUaslReservation(ctx context.Context, baseURL string, requestID string, isInterConnect bool) (*dmodel.UaslReservationResponse, error) {
	base := g.resolveBaseURL(baseURL)

	extReq := conv.ToExternalUaslConfirmRequest(isInterConnect)

	reqBody, err := json.Marshal(extReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal confirm request: %w", err)
	}

	path := fmt.Sprintf(pathUaslReservationsConfirm, requestID)
	httpReq, err := g.newJSONRequest(ctx, http.MethodPut, buildRequestURL(base, path), reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call ouranos proxy confirm API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		var errBody map[string]interface{}
		_ = json.Unmarshal(bodyBytes, &errBody)
		return &dmodel.UaslReservationResponse{Error: &dmodel.ResponseError{
			Code:    fmt.Sprintf("http:%d", resp.StatusCode),
			Message: string(bodyBytes),
			Details: errBody,
		}}, nil
	}

	var extResp ouranosModel.UaslReservationResponse
	if err := json.NewDecoder(resp.Body).Decode(&extResp); err != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return &dmodel.UaslReservationResponse{Error: &dmodel.ResponseError{
			Code:    "decode_error",
			Message: err.Error(),
			Details: map[string]interface{}{"body": string(bodyBytes)},
		}}, nil
	}

	return conv.ToUaslReservationResponse(&extResp), nil
}

func (g *ouranosProxyGateway) EstimateUaslReservation(ctx context.Context, baseURL string, req dmodel.ExternalEstimateRequest) (*dmodel.ExternalEstimateResponse, error) {
	base := g.resolveBaseURL(baseURL)

	extReq := conv.ToExternalEstimateRequest(req)

	reqBody, err := json.Marshal(extReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal estimate request: %w", err)
	}

	httpReq, err := g.newJSONRequest(ctx, http.MethodPost, buildRequestURL(base, pathUaslReservationsEstimate), reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call ouranos proxy estimate API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ouranos proxy estimate API returned status %d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var extResp ouranosModel.EstimateResponse
	if err := json.NewDecoder(resp.Body).Decode(&extResp); err != nil {
		return nil, fmt.Errorf("failed to decode estimate response: %w", err)
	}

	return conv.ToInternalEstimateResponse(&extResp), nil
}

func (g *ouranosProxyGateway) SendReservationCompleted(ctx context.Context, baseURL string, payload dmodel.DestinationReservationNotification) error {
	base := g.resolveBaseURL(baseURL)

	extReq := conv.ToExternalDestinationReservationNotification(payload)

	reqBody, err := json.Marshal(extReq)
	if err != nil {
		return fmt.Errorf("failed to marshal send request: %w", err)
	}

	httpReq, err := g.newJSONRequest(ctx, http.MethodPost, buildRequestURL(base, pathUaslReservationsCompleted), reqBody)
	if err != nil {
		return err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
