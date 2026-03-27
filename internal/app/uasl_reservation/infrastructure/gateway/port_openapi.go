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

	gswmodel "uasl-reservation/external/uasl/port/model"
	"uasl-reservation/internal/app/uasl_reservation/application/converter"
	gatewayIF "uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/infrastructure/gateway/mock"
	httpClient "uasl-reservation/internal/pkg/http"
	"uasl-reservation/internal/pkg/logger"
	"uasl-reservation/internal/pkg/myerror"
)

const (
	pathPortReserveList   = "/asset/api/droneport/reserve/list"
	pathPortReserveCreate = "/asset/api/droneport/reserve"
	pathPortReserveDelete = "/asset/api/droneport/reserve/%s"
	pathPortReserveDetail = "/asset/api/droneport/reserve/detail/%s"
	pathPortInfoList      = "/asset/api/droneport/info/list"
	pathPortInfoDetail    = "/asset/api/droneport/info/detail/%s"

	portDefaultTimeout = 30 * time.Second
)

type portOpenAPIGateway struct {
	client    *httpClient.HttpClient
	baseURL   string
	converter *converter.PortReservationConverter
}

func NewPortOpenAPIGateway() gatewayIF.PortOpenAPIGatewayIF {
	if os.Getenv("APP_ENV") == "local" {
		logger.LogInfo("Using port mock gateway (APP_ENV=local)")
		return mock.NewPortOpenAPIMockGateway()
	}
	baseURL := os.Getenv("RESOURCE_BASE_URL")
	return &portOpenAPIGateway{
		client:    httpClient.NewHttpClientWithTimeout(portDefaultTimeout),
		baseURL:   baseURL,
		converter: converter.NewPortReservationConverter(),
	}
}

func (g *portOpenAPIGateway) resolveBaseURL(baseURL string) string {
	if baseURL != "" {
		return baseURL
	}
	return g.baseURL
}

func (g *portOpenAPIGateway) newJSONRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
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

func (g *portOpenAPIGateway) doRequest(httpReq *http.Request) (*http.Response, error) {
	if g.client != nil {
		return g.client.DoRequest(&httpClient.HttpRequest{Request: httpReq})
	}
	hc := httpClient.NewHttpClientWithTimeout(portDefaultTimeout)
	return hc.DoRequest(&httpClient.HttpRequest{Request: httpReq})
}

func (g *portOpenAPIGateway) ListReservations(ctx context.Context, baseURL string, req model.PortFetchRequest) ([]model.PortReservationInfo, error) {
	listReq := g.converter.ToListReservationsRequest(req)

	query := url.Values{}
	query.Set("dronePortId", listReq.DronePortID)
	if listReq.TimeFrom != "" {
		query.Set("timeFrom", listReq.TimeFrom)
	}
	if listReq.TimeTo != "" {
		query.Set("timeTo", listReq.TimeTo)
	}

	requestURL := fmt.Sprintf("%s%s?%s", strings.TrimRight(g.resolveBaseURL(baseURL), "/"), pathPortReserveList, query.Encode())

	httpReq, err := g.newJSONRequest(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		logger.LogError("[port] ListReservations: http call failed - url=%s, error=%v", requestURL, err)
		return nil, fmt.Errorf("http call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logger.LogError("[port] ListReservations: unexpected status=%d, body=%s", resp.StatusCode, string(bodyBytes))
		return nil, myerror.NewHTTPError(resp.StatusCode, "GSW API request failed", string(bodyBytes))
	}

	var listResp gswmodel.DronePortReserveInfoListResponseDto
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		logger.LogError("[port] ListReservations: decode failed - error=%v", err)
		return nil, fmt.Errorf("decode response: %w", err)
	}

	allReservations := make([]model.PortReservationInfo, 0, len(listResp.Data))
	for _, item := range listResp.Data {
		startAt, err := time.Parse(time.RFC3339, item.ReservationTimeFrom)
		if err != nil {
			logger.LogError("[port] ListReservations: failed to parse start time - reservation_id=%s, value=%s, error=%v",
				item.DronePortReservationID, item.ReservationTimeFrom, err)
			continue
		}
		endAt, err := time.Parse(time.RFC3339, item.ReservationTimeTo)
		if err != nil {
			logger.LogError("[port] ListReservations: failed to parse end time - reservation_id=%s, value=%s, error=%v",
				item.DronePortReservationID, item.ReservationTimeTo, err)
			continue
		}
		allReservations = append(allReservations, model.PortReservationInfo{
			ReservationID: &item.DronePortReservationID,
			PortID:        item.DronePortID,
			PortName:      item.DronePortName,
			UsageType:     0,
			StartAt:       startAt,
			EndAt:         endAt,
		})
	}

	return allReservations, nil
}

func (g *portOpenAPIGateway) Reserve(ctx context.Context, baseURL string, req model.PortReservationRequest) (model.ReservationHandle, error) {

	createReq := g.converter.ToCreateReservationRequest(req)

	body, err := json.Marshal(createReq)
	if err != nil {
		return model.ReservationHandle{}, fmt.Errorf("marshal request: %w", err)
	}

	requestURL := buildRequestURL(g.resolveBaseURL(baseURL), pathPortReserveCreate)

	httpReq, err := g.newJSONRequest(ctx, http.MethodPost, requestURL, body)
	if err != nil {
		return model.ReservationHandle{}, err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		logger.LogError("[port] Reserve: http call failed - url=%s, error=%v", requestURL, err)
		return model.ReservationHandle{}, fmt.Errorf("http call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logger.LogError("[port] Reserve: unexpected status=%d, body=%s", resp.StatusCode, string(bodyBytes))
		return model.ReservationHandle{}, myerror.NewHTTPError(resp.StatusCode, "GSW API request failed", string(bodyBytes))
	}

	var createResp gswmodel.DronePortReserveInfoRegisterListResponseDto
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		logger.LogError("[port] Reserve: decode failed - error=%v", err)
		return model.ReservationHandle{}, fmt.Errorf("decode response: %w", err)
	}

	handle, err := g.converter.ToReservationHandle(createResp, g.resolveBaseURL(baseURL))
	if err != nil {
		logger.LogError("[port] Reserve: failed to convert handle - error=%v", err)
		return model.ReservationHandle{}, fmt.Errorf("failed to convert reservation response to handle: %w", err)
	}

	return handle, nil
}

func (g *portOpenAPIGateway) Cancel(ctx context.Context, baseURL string, handle model.ReservationHandle, operatorID string) error {
	requestURL := fmt.Sprintf("%s%s?dronePortReservationIdFlag=true", g.resolveBaseURL(baseURL), fmt.Sprintf(pathPortReserveDelete, url.PathEscape(handle.ID)))

	logger.LogInfo("port reservation cancel requested - url=%s, operator_id=%s, reservation_id=%s",
		requestURL, operatorID, handle.ID)

	httpReq, err := g.newJSONRequest(ctx, http.MethodDelete, requestURL, nil)
	if err != nil {
		logger.LogError("port cancel: failed to build request - error: %v", err)
		return fmt.Errorf("port cancel: build request: %w", err)
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		logger.LogError("port cancel: HTTP request failed - url=%s, error: %v", requestURL, err)
		return fmt.Errorf("port cancel: call failed: %w", err)
	}
	defer resp.Body.Close()

	logger.LogInfo("port reservation cancel response - status=%d, reservation_id=%s", resp.StatusCode, handle.ID)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyResp, _ := io.ReadAll(resp.Body)
		logger.LogError("port cancel: unexpected status - status=%d, reservation_id=%s, response_body=%s",
			resp.StatusCode, handle.ID, string(bodyResp))
		return myerror.NewHTTPError(resp.StatusCode, "External port reservation cancellation failed", string(bodyResp))
	}

	logger.LogInfo("port reservation cancelled successfully - reservation_id=%s, operator_id=%s", handle.ID, operatorID)
	return nil
}

func (g *portOpenAPIGateway) GetDronePortReservationDetail(ctx context.Context, baseURL string, dronePortReservationID string) (*model.PortReservationDetail, error) {
	requestURL := buildRequestURL(g.resolveBaseURL(baseURL), fmt.Sprintf(pathPortReserveDetail, dronePortReservationID))

	httpReq, err := g.newJSONRequest(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		logger.LogError("[port] GetDronePortReservationDetail: http call failed - error=%v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.LogError("[port] GetDronePortReservationDetail: unexpected status=%d, body=%s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("droneport reservation detail api status=%d body=%s", resp.StatusCode, string(body))
	}

	var dto gswmodel.DronePortReserveInfoDetailResponseDto
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		logger.LogError("[port] GetDronePortReservationDetail: decode failed - error=%v", err)
		return nil, fmt.Errorf("failed to decode droneport reservation detail response: %w", err)
	}

	conv := converter.NewPortReservationConverter()
	detail, err := conv.ToPortReservationDetail(dto)
	if err != nil {
		logger.LogError("[port] GetDronePortReservationDetail: convert failed - error=%v", err)
		return nil, fmt.Errorf("failed to convert droneport reservation detail to domain model: %w", err)
	}

	return detail, nil
}

func (g *portOpenAPIGateway) FetchDronePortList(ctx context.Context, baseURL string, isRequiredPriceInfo bool) ([]*model.ExternalUaslResource, error) {
	query := url.Values{}
	if isRequiredPriceInfo {
		query.Set("isRequiredPriceInfo", "true")
	} else {
		query.Set("isRequiredPriceInfo", "false")
	}
	requestURL := fmt.Sprintf("%s%s?%s", strings.TrimRight(g.resolveBaseURL(baseURL), "/"), pathPortInfoList, query.Encode())

	httpReq, err := g.newJSONRequest(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("droneport info list: build request: %w", err)
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		logger.LogError("[port] FetchDronePortList: http call failed - error=%v", err)
		return nil, fmt.Errorf("droneport info list: http call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyResp, _ := io.ReadAll(resp.Body)
		logger.LogError("[port] FetchDronePortList: unexpected status=%d, body=%s", resp.StatusCode, string(bodyResp))
		return nil, myerror.NewHTTPError(resp.StatusCode, "droneport info list API error", string(bodyResp))
	}

	bodyResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("droneport info list: read response body: %w", err)
	}

	var externalResp gswmodel.DronePortInfoListResponseDto
	if err := json.Unmarshal(bodyResp, &externalResp); err != nil {
		logger.LogError("[port] FetchDronePortList: unmarshal failed - error=%v, body=%s", err, string(bodyResp))
		return nil, fmt.Errorf("droneport info list: unmarshal response: %w", err)
	}

	logger.LogInfo("FetchDronePortList succeeded", "droneport_count", len(externalResp.Data))

	result := converter.ToExternalUaslResourceFromPort(&externalResp)

	return result, nil
}

func (g *portOpenAPIGateway) GetDronePortInfoDetail(ctx context.Context, dronePortID string) (*model.DronePortInfo, error) {

	requestURL := buildRequestURL(g.baseURL, fmt.Sprintf(pathPortInfoDetail, url.PathEscape(dronePortID)))

	httpReq, err := g.newJSONRequest(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("droneport info detail: build request: %w", err)
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		logger.LogError("[port] GetDronePortInfoDetail: http call failed - droneport_id=%s, error=%v", dronePortID, err)
		return nil, fmt.Errorf("droneport info detail: http call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyResp, _ := io.ReadAll(resp.Body)
		logger.LogError("[port] GetDronePortInfoDetail: unexpected status=%d, droneport_id=%s, body=%s",
			resp.StatusCode, dronePortID, string(bodyResp))
		return nil, myerror.NewHTTPError(resp.StatusCode, "droneport info detail API error", string(bodyResp))
	}

	bodyResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("droneport info detail: read response body: %w", err)
	}

	var dto gswmodel.DronePortInfoDetailResponseDto
	if err := json.Unmarshal(bodyResp, &dto); err != nil {
		logger.LogError("[port] GetDronePortInfoDetail: unmarshal failed - error=%v, body=%s", err, string(bodyResp))
		return nil, fmt.Errorf("droneport info detail: unmarshal response: %w", err)
	}

	return &model.DronePortInfo{
		DronePortID:   dto.DronePortID,
		DronePortName: dto.DronePortName,
		ActiveStatus:  dto.ActiveStatus,
	}, nil
}
