package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	gswmodel "uasl-reservation/external/uasl/vehicle/model"
	"uasl-reservation/internal/app/uasl_reservation/application/converter"
	gatewayIF "uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/infrastructure/gateway/mock"
	httpClient "uasl-reservation/internal/pkg/http"
	"uasl-reservation/internal/pkg/logger"
	"uasl-reservation/internal/pkg/myerror"
)

const (
	pathAircraftReserveList   = "/asset/api/aircraft/reserve/list"
	pathAircraftReserveCreate = "/asset/api/aircraft/reserve"
	pathAircraftReserveDelete = "/asset/api/aircraft/reserve/%s"
	pathAircraftReserveDetail = "/asset/api/aircraft/reserve/detail/%s"
	pathAircraftInfoDetail    = "/asset/api/aircraft/info/detail/%s"
	pathAircraftInfoList      = "/asset/api/aircraft/info/list"

	vehicleDefaultTimeout = 30 * time.Second
)

type vehicleOpenAPIGateway struct {
	client    *httpClient.HttpClient
	baseURL   string
	converter *converter.VehicleReservationConverter
}

func (g *vehicleOpenAPIGateway) resolveBaseURL(baseURL string) string {
	if baseURL != "" {
		return baseURL
	}
	return g.baseURL
}

func NewVehicleOpenAPIGateway() gatewayIF.VehicleOpenAPIGatewayIF {
	if os.Getenv("APP_ENV") == "local" {
		logger.LogInfo("Using vehicle mock gateway (APP_ENV=local)")
		return mock.NewVehicleOpenAPIMockGateway()
	}
	baseURL := os.Getenv("RESOURCE_BASE_URL")
	return &vehicleOpenAPIGateway{
		client:    httpClient.NewHttpClientWithTimeout(vehicleDefaultTimeout),
		baseURL:   baseURL,
		converter: converter.NewVehicleReservationConverter(),
	}
}

func (g *vehicleOpenAPIGateway) newJSONRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
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
	return httpReq, nil
}

func (g *vehicleOpenAPIGateway) doRequest(httpReq *http.Request) (*http.Response, error) {
	if g.client != nil {
		return g.client.DoRequest(&httpClient.HttpRequest{Request: httpReq})
	}
	hc := httpClient.NewHttpClientWithTimeout(vehicleDefaultTimeout)
	return hc.DoRequest(&httpClient.HttpRequest{Request: httpReq})
}

func (g *vehicleOpenAPIGateway) ListReservations(ctx context.Context, baseURL string, req model.VehicleFetchRequest) ([]model.VehicleReservationInfo, error) {

	listReq := g.converter.ToListReservationsRequest(req)

	query := url.Values{}
	query.Set("aircraftId", listReq.AircraftID)
	if listReq.TimeFrom != "" {
		query.Set("timeFrom", listReq.TimeFrom)
	}
	if listReq.TimeTo != "" {
		query.Set("timeTo", listReq.TimeTo)
	}

	requestURL := fmt.Sprintf("%s%s?%s", strings.TrimRight(g.resolveBaseURL(baseURL), "/"), pathAircraftReserveList, query.Encode())

	httpReq, err := g.newJSONRequest(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		logger.LogError("[vehicle] ListReservations: http call failed - url=%s, error=%v", requestURL, err)
		return nil, fmt.Errorf("http call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logger.LogError("[vehicle] ListReservations: unexpected status=%d, body=%s", resp.StatusCode, string(bodyBytes))
		return nil, myerror.NewHTTPError(resp.StatusCode, "GSW API request failed", string(bodyBytes))
	}

	var listResp gswmodel.AircraftReserveInfoListResponseDto
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		logger.LogError("[vehicle] ListReservations: decode failed - error=%v", err)
		return nil, fmt.Errorf("decode response: %w", err)
	}

	allReservations := make([]model.VehicleReservationInfo, 0, len(listResp.Data))
	for _, item := range listResp.Data {
		startAt, err := time.Parse(time.RFC3339, item.ReservationTimeFrom)
		if err != nil {
			logger.LogError("[vehicle] ListReservations: failed to parse start time - reservation_id=%s, value=%s, error=%v",
				item.AircraftReservationID, item.ReservationTimeFrom, err)
			continue
		}
		endAt, err := time.Parse(time.RFC3339, item.ReservationTimeTo)
		if err != nil {
			logger.LogError("[vehicle] ListReservations: failed to parse end time - reservation_id=%s, value=%s, error=%v",
				item.AircraftReservationID, item.ReservationTimeTo, err)
			continue
		}
		allReservations = append(allReservations, model.VehicleReservationInfo{
			ReservationID: &item.AircraftReservationID,
			VehicleID:     item.AircraftID,
			VehicleName:   item.AircraftName,
			StartAt:       startAt,
			EndAt:         endAt,
		})
	}

	return allReservations, nil
}

func (g *vehicleOpenAPIGateway) Reserve(ctx context.Context, baseURL string, req model.VehicleReservationRequest) (model.ReservationHandle, error) {

	createReq := g.converter.ToCreateReservationRequest(req)

	body, err := json.Marshal(createReq)
	if err != nil {
		return model.ReservationHandle{}, fmt.Errorf("marshal request: %w", err)
	}

	requestURL := buildRequestURL(g.resolveBaseURL(baseURL), pathAircraftReserveCreate)

	httpReq, err := g.newJSONRequest(ctx, http.MethodPost, requestURL, body)
	if err != nil {
		return model.ReservationHandle{}, err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		logger.LogError("[vehicle] Reserve: http call failed - url=%s, error=%v", requestURL, err)
		return model.ReservationHandle{}, fmt.Errorf("http call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logger.LogError("[vehicle] Reserve: unexpected status=%d, body=%s", resp.StatusCode, string(bodyBytes))
		return model.ReservationHandle{}, myerror.NewHTTPError(resp.StatusCode, "GSW API request failed", string(bodyBytes))
	}

	var createResp gswmodel.AircraftReserveInfoResponseDto
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		logger.LogError("[vehicle] Reserve: decode failed - error=%v", err)
		return model.ReservationHandle{}, fmt.Errorf("decode response: %w", err)
	}

	handle, err := g.converter.ToReservationHandle(createResp, g.resolveBaseURL(baseURL))
	if err != nil {
		logger.LogError("[vehicle] Reserve: failed to convert handle - error=%v", err)
		return model.ReservationHandle{}, fmt.Errorf("failed to convert reservation response to handle: %w", err)
	}

	return handle, nil
}

func (g *vehicleOpenAPIGateway) Cancel(ctx context.Context, baseURL string, handle model.ReservationHandle, operatorID string) error {
	requestURL := fmt.Sprintf("%s%s?aircraftReservationIdFlag=true", g.resolveBaseURL(baseURL), fmt.Sprintf(pathAircraftReserveDelete, url.PathEscape(handle.ID)))

	logger.LogInfo("vehicle reservation cancel requested - url=%s, operator_id=%s, reservation_id=%s",
		requestURL, operatorID, handle.ID)

	httpReq, err := g.newJSONRequest(ctx, http.MethodDelete, requestURL, nil)
	if err != nil {
		logger.LogError("vehicle cancel: failed to build request - error: %v", err)
		return fmt.Errorf("vehicle cancel: build request: %w", err)
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		logger.LogError("vehicle cancel: HTTP request failed - url=%s, error: %v", requestURL, err)
		return fmt.Errorf("vehicle cancel: call failed: %w", err)
	}
	defer resp.Body.Close()

	logger.LogInfo("vehicle reservation cancel response - status=%d, reservation_id=%s", resp.StatusCode, handle.ID)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyResp, _ := io.ReadAll(resp.Body)
		logger.LogError("vehicle cancel: unexpected status - status=%d, reservation_id=%s, response_body=%s",
			resp.StatusCode, handle.ID, string(bodyResp))
		return myerror.NewHTTPError(resp.StatusCode, "External vehicle reservation cancellation failed", string(bodyResp))
	}

	logger.LogInfo("vehicle reservation cancelled successfully - reservation_id=%s, operator_id=%s", handle.ID, operatorID)
	return nil
}

func (g *vehicleOpenAPIGateway) GetAircraftReservationDetail(ctx context.Context, baseURL string, aircraftReservationID string) (*model.VehicleReservationDetail, error) {

	requestURL := buildRequestURL(g.resolveBaseURL(baseURL), fmt.Sprintf(pathAircraftReserveDetail, aircraftReservationID))

	httpReq, err := g.newJSONRequest(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		logger.LogError("[vehicle] GetAircraftReservationDetail: http call failed - error=%v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.LogError("[vehicle] GetAircraftReservationDetail: unexpected status=%d, body=%s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("aircraft reservation detail api status=%d body=%s", resp.StatusCode, string(body))
	}

	var dto gswmodel.AircraftReserveInfoDetailResponseDto
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		logger.LogError("[vehicle] GetAircraftReservationDetail: decode failed - error=%v", err)
		return nil, fmt.Errorf("failed to decode aircraft reservation detail response: %w", err)
	}

	conv := converter.NewVehicleReservationConverter()
	detail, err := conv.ToVehicleReservationDetail(dto)
	if err != nil {
		logger.LogError("[vehicle] GetAircraftReservationDetail: convert failed - error=%v", err)
		return nil, fmt.Errorf("failed to convert aircraft reservation detail to domain model: %w", err)
	}

	return detail, nil
}

func (g *vehicleOpenAPIGateway) GetAircraftInfoDetail(ctx context.Context, baseURL string, aircraftID string, isRequiredPriceInfo bool) (*model.AircraftInfoDetail, error) {

	query := url.Values{}
	if isRequiredPriceInfo {
		query.Set("isRequiredPriceInfo", "true")
	} else {
		query.Set("isRequiredPriceInfo", "false")
	}
	requestURL := fmt.Sprintf("%s%s?%s",
		strings.TrimRight(g.resolveBaseURL(baseURL), "/"),
		fmt.Sprintf(pathAircraftInfoDetail, url.PathEscape(aircraftID)),
		query.Encode(),
	)

	httpReq, err := g.newJSONRequest(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("aircraft info detail: build request: %w", err)
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		logger.LogError("[vehicle] GetAircraftInfoDetail: http call failed - error=%v", err)
		return nil, fmt.Errorf("aircraft info detail: http call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.LogError("[vehicle] GetAircraftInfoDetail: unexpected status=%d, body=%s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("aircraft info detail api status=%d body=%s", resp.StatusCode, string(body))
	}

	var dto gswmodel.AircraftInfoDetailResponseDto
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		logger.LogError("[vehicle] GetAircraftInfoDetail: decode failed - error=%v", err)
		return nil, fmt.Errorf("aircraft info detail: decode response: %w", err)
	}

	conv := converter.NewVehicleReservationConverter()
	detail := conv.ToAircraftInfoDetail(dto)
	return detail, nil
}

func (g *vehicleOpenAPIGateway) FetchAircraftList(ctx context.Context, baseURL string, isRequiredPriceInfo bool) ([]*model.ExternalUaslResource, error) {

	query := url.Values{}
	if isRequiredPriceInfo {
		query.Set("isRequiredPriceInfo", "true")
	} else {
		query.Set("isRequiredPriceInfo", "false")
	}
	requestURL := fmt.Sprintf("%s%s?%s", strings.TrimRight(g.resolveBaseURL(baseURL), "/"), pathAircraftInfoList, query.Encode())

	httpReq, err := g.newJSONRequest(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("aircraft info list: build request: %w", err)
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		logger.LogError("[vehicle] FetchAircraftList: http call failed - error=%v", err)
		return nil, fmt.Errorf("aircraft info list: http call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyResp, _ := io.ReadAll(resp.Body)
		logger.LogError("[vehicle] FetchAircraftList: unexpected status=%d, body=%s", resp.StatusCode, string(bodyResp))
		return nil, myerror.NewHTTPError(resp.StatusCode, "aircraft info list API error", string(bodyResp))
	}

	bodyResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("aircraft info list: read response body: %w", err)
	}

	var externalResp gswmodel.AircraftInfoSearchListResponseDto
	if err := json.Unmarshal(bodyResp, &externalResp); err != nil {
		logger.LogError("[vehicle] FetchAircraftList: unmarshal failed - error=%v, body=%s", err, string(bodyResp))
		return nil, fmt.Errorf("aircraft info list: unmarshal response: %w", err)
	}

	logger.LogInfo("FetchAircraftList succeeded", "aircraft_count", len(externalResp.Data))

	result := converter.ToExternalUaslResourceFromAircraft(&externalResp)
	return result, nil
}
