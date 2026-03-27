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

	gswmodel "uasl-reservation/external/uasl/resource_price/model"
	conv "uasl-reservation/internal/app/uasl_reservation/application/converter"
	gatewayIF "uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	mock "uasl-reservation/internal/app/uasl_reservation/infrastructure/gateway/mock"
	httpClient "uasl-reservation/internal/pkg/http"
	"uasl-reservation/internal/pkg/logger"
)

const (
	pathResourcePriceList = "/asset/api/price/info/resourcePriceList"

	resourcePriceDefaultTimeout = 30 * time.Second
)

type gswPriceGateway struct {
	baseURL string
	client  *httpClient.HttpClient
}

func (g *gswPriceGateway) resolveBaseURL(baseURL string) string {
	if baseURL != "" {
		return baseURL
	}
	return g.baseURL
}

func NewResourcePriceGateway() gatewayIF.ResourcePriceGatewayIF {
	if os.Getenv("APP_ENV") == "local" {
		logger.LogInfo("Using resource price mock gateway (APP_ENV=local)")
		return mock.NewResourcePriceGatewayLocalMock()
	}
	baseURL := os.Getenv("RESOURCE_BASE_URL")
	return &gswPriceGateway{
		baseURL: baseURL,
		client:  httpClient.NewHttpClientWithTimeout(resourcePriceDefaultTimeout),
	}
}

func (g *gswPriceGateway) newJSONRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
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

func (g *gswPriceGateway) doRequest(httpReq *http.Request) (*http.Response, error) {
	if g.client != nil {
		return g.client.DoRequest(&httpClient.HttpRequest{Request: httpReq})
	}
	hc := httpClient.NewHttpClientWithTimeout(resourcePriceDefaultTimeout)
	return hc.DoRequest(&httpClient.HttpRequest{Request: httpReq})
}

func (g *gswPriceGateway) GetResourcePriceList(ctx context.Context, baseURL string, req model.ResourcePriceListRequest) (*model.ResourcePriceList, error) {
	extReq := conv.ToExternalResourcePriceRequest(req)

	params := url.Values{}
	params.Add("resourceId", extReq.ResourceID)
	if extReq.ResourceType != nil {
		params.Add("resourceType", fmt.Sprintf("%d", *extReq.ResourceType))
	}
	if extReq.PriceType != nil {
		params.Add("priceType", fmt.Sprintf("%d", *extReq.PriceType))
	}
	if extReq.EffectiveStartTime != "" {
		params.Add("effectiveStartTime", extReq.EffectiveStartTime)
	}
	if extReq.EffectiveEndTime != "" {
		params.Add("effectiveEndTime", extReq.EffectiveEndTime)
	}

	requestURL := fmt.Sprintf("%s%s?%s", strings.TrimRight(g.resolveBaseURL(baseURL), "/"), pathResourcePriceList, params.Encode())

	httpReq, err := g.newJSONRequest(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		logger.LogError("[price] GetResourcePriceList: http call failed - url=%s, error=%v", requestURL, err)
		return nil, fmt.Errorf("failed to call GSW price API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.LogError("[price] GetResourcePriceList: unexpected status=%d", resp.StatusCode)
		return nil, fmt.Errorf("GSW price API returned status %d", resp.StatusCode)
	}

	var response gswmodel.ResourcePriceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		logger.LogError("[price] GetResourcePriceList: decode failed - error=%v", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	domainList, err := conv.ToDomainResourcePriceList(&response)
	if err != nil {
		logger.LogError("[price] GetResourcePriceList: convert failed - error=%v", err)
		return nil, fmt.Errorf("failed to convert GSW response to domain model: %w", err)
	}

	return &domainList, nil
}
