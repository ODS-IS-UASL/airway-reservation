package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	dipsmodel "uasl-reservation/external/uasl/dips/model"
	"uasl-reservation/internal/app/uasl_reservation/application/converter"
	gatewayIF "uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/infrastructure/gateway/mock"
	httpClient "uasl-reservation/internal/pkg/http"
	"uasl-reservation/internal/pkg/logger"
)

const (
	pathFlightPlanInfo = "/flightPlanInfoReceiver"

	dipsDefaultTimeout = 5 * time.Second
)

type flightPlanInfoOpenAPIGateway struct {
	client    *httpClient.HttpClient
	baseURL   string
	converter *converter.FlightPlanInfoConverter
}

func NewFlightPlanInfoOpenAPIGateway() gatewayIF.FlightPlanInfoOpenAPIGatewayIF {
	baseURL := os.Getenv("DIPS_FPR_BASE_URL")
	if os.Getenv("APP_ENV") == "local" {
		logger.LogInfo("Using DIPS mock gateway (APP_ENV=local)")
		return mock.NewFlightPlanInfoMockGateway()
	}
	return &flightPlanInfoOpenAPIGateway{
		client:    httpClient.NewHttpClientWithTimeout(dipsDefaultTimeout),
		baseURL:   baseURL,
		converter: converter.NewFlightPlanInfoConverter(),
	}
}

func (g *flightPlanInfoOpenAPIGateway) newJSONRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewBuffer(body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	return httpReq, nil
}

func (g *flightPlanInfoOpenAPIGateway) doRequest(httpReq *http.Request) (*http.Response, error) {
	if g.client != nil {
		return g.client.DoRequest(&httpClient.HttpRequest{Request: httpReq})
	}
	hc := httpClient.NewHttpClientWithTimeout(dipsDefaultTimeout)
	return hc.DoRequest(&httpClient.HttpRequest{Request: httpReq})
}

func (g *flightPlanInfoOpenAPIGateway) Fetch(
	ctx context.Context,
	req model.FlightPlanInfoRequest,
) (model.FlightPlanInfoResponse, error) {

	logger.LogDebug("[FlightPlanInfoGateway] Fetch start: req=%+v", req)

	externalReq := g.converter.ToExternalRequest(req)

	logger.LogDebug("[FlightPlanInfoGateway] converted to external request: externalReq=%+v", externalReq)

	body, err := json.Marshal(externalReq)
	if err != nil {

		logger.LogDebug("[FlightPlanInfoGateway] json.Marshal failed: err=%v", err)
		return model.FlightPlanInfoResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	logger.LogDebug("[FlightPlanInfoGateway] request body marshaled: body=%s", string(body))

	requestURL := buildRequestURL(g.baseURL, pathFlightPlanInfo)

	logger.LogDebug("[FlightPlanInfoGateway] request URL: %s", requestURL)

	httpReq, err := g.newJSONRequest(ctx, http.MethodPost, requestURL, body)
	if err != nil {

		logger.LogDebug("[FlightPlanInfoGateway] newJSONRequest failed: err=%v", err)
		return model.FlightPlanInfoResponse{}, fmt.Errorf("build request: %w", err)
	}

	logger.LogDebug("[FlightPlanInfoGateway] HTTP request created: method=%s url=%s", httpReq.Method, httpReq.URL)

	logger.LogDebug("[FlightPlanInfoGateway] sending HTTP request...")
	resp, err := g.doRequest(httpReq)
	if err != nil {

		logger.LogDebug("[FlightPlanInfoGateway] doRequest failed: err=%v", err)
		return model.FlightPlanInfoResponse{}, fmt.Errorf("http call failed: %w", err)
	}
	defer resp.Body.Close()

	logger.LogDebug("[FlightPlanInfoGateway] HTTP response received: status=%d", resp.StatusCode)

	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {

		logger.LogDebug("[FlightPlanInfoGateway] io.ReadAll failed: err=%v", readErr)
		return model.FlightPlanInfoResponse{}, fmt.Errorf("read response body: %w", readErr)
	}

	logger.LogDebug("[FlightPlanInfoGateway] response body: %s", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {

		logger.LogDebug("[FlightPlanInfoGateway] unexpected status: status=%d body=%s", resp.StatusCode, string(bodyBytes))
		return model.FlightPlanInfoResponse{}, fmt.Errorf(
			"status=%d body=%s",
			resp.StatusCode,
			string(bodyBytes),
		)
	}

	var flightPlanResp dipsmodel.FlightPlanInfoResponse
	if err := json.Unmarshal(bodyBytes, &flightPlanResp); err != nil {

		logger.LogDebug("[FlightPlanInfoGateway] json.Unmarshal failed: err=%v body=%s", err, string(bodyBytes))
		return model.FlightPlanInfoResponse{}, fmt.Errorf("decode response: %w", err)
	}

	logger.LogDebug("[FlightPlanInfoGateway] response unmarshaled: flightPlanResp=%+v", flightPlanResp)

	domainResp := g.converter.ToDomainResponse(flightPlanResp)

	logger.LogDebug("[FlightPlanInfoGateway] Fetch done: domainResp=%+v", domainResp)

	return domainResp, nil
}
