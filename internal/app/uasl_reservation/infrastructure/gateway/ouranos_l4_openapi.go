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

	extModel "uasl-reservation/external/uasl/l4/model"
	"uasl-reservation/internal/app/uasl_reservation/application/converter"
	"uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/infrastructure/gateway/mock"
	httpClient "uasl-reservation/internal/pkg/http"
	"uasl-reservation/internal/pkg/logger"
)

const ouranosDefaultTimeout = 30 * time.Second

const (
	pathDiscoveryOperations = "/api/v1/uasl/operations"
)

type ouranosDiscoveryGateway struct {
	l4BaseURL string
	client    *httpClient.HttpClient
}

func NewOuranosDiscoveryGateway() gatewayIF.OuranosDiscoveryGatewayIF {
	if os.Getenv("APP_ENV") == "local" {
		logger.LogInfo("Using Ouranos discovery mock (APP_ENV=local)")
		return mock.NewOuranosDiscoveryGatewayLocalMock()
	}

	l4Base := os.Getenv("L4_BASE_URL")

	return &ouranosDiscoveryGateway{
		l4BaseURL: l4Base,
		client:    httpClient.NewHttpClientWithTimeout(ouranosDefaultTimeout),
	}
}

func (g *ouranosDiscoveryGateway) doRequest(httpReq *http.Request) (*http.Response, error) {
	if g.client != nil {
		return g.client.DoRequest(&httpClient.HttpRequest{Request: httpReq})
	}
	hc := httpClient.NewHttpClientWithTimeout(ouranosDefaultTimeout)
	return hc.DoRequest(&httpClient.HttpRequest{Request: httpReq})
}

func (g *ouranosDiscoveryGateway) newJSONRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
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

func (g *ouranosDiscoveryGateway) FindResourceFromDiscoveryService(ctx context.Context, req model.FindResourceRequest) ([]model.UaslServiceURL, error) {
	requestURL := buildRequestURL(g.l4BaseURL, pathDiscoveryOperations)

	extReq := converter.ToFindResourceRequest(req)
	bodyBytes, err := json.Marshal(extReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal findResource request: %w", err)
	}

	httpReq, err := g.newJSONRequest(ctx, http.MethodPost, requestURL, bodyBytes)
	if err != nil {
		return nil, err
	}

	hasAuth := false
	if _, ok := getAuthorizationFromContext(ctx); ok {
		hasAuth = true
	}
	logger.LogInfo("FindResourceFromDiscoveryService: calling findResource url=%s method=%s has_auth=%t body_bytes=%d service_name=%s lat=%f lng=%f radius_km=%f", requestURL, http.MethodPost, hasAuth, len(bodyBytes), req.ServiceName, req.Lat, req.Lng, req.RadiusKm)

	resp, err := g.doRequest(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Discovery Service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Discovery Service returned status=%d body=%s", resp.StatusCode, string(body))
	}

	var dsResp extModel.FindResourceResponse
	if err := json.NewDecoder(resp.Body).Decode(&dsResp); err != nil {
		return nil, fmt.Errorf("failed to decode Discovery Service response: %w", err)
	}

	if dsResp.Error != nil {
		return nil, fmt.Errorf("Discovery Service error: code=%s message=%s", dsResp.Error.Code, dsResp.Error.Message)
	}

	urls := converter.ToUaslServiceURLs(dsResp, req.ServiceName)

	return urls, nil
}
