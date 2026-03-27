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

	conformityAssessmentModel "uasl-reservation/external/uasl/uasl_design/model"
	"uasl-reservation/internal/app/uasl_reservation/application/converter"
	gatewayIF "uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/infrastructure/gateway/mock"
	httpClient "uasl-reservation/internal/pkg/http"
	"uasl-reservation/internal/pkg/logger"
	"uasl-reservation/internal/pkg/myerror"
)

const (
	pathUaslList       = "/uaslDesign/uasl"
	uaslDefaultTimeout = 30 * time.Second
)

type uaslDesignOpenAPIGateway struct {
	client  *httpClient.HttpClient
	baseURL string
}

func NewUaslDesignOpenAPIGateway() gatewayIF.UaslDesignGatewayIF {
	if os.Getenv("APP_ENV") == "local" {
		logger.LogInfo("Using uasl design mock gateway (APP_ENV=local)")
		return mock.NewUaslDesignOpenAPIMockGateway()
	}
	baseURL := os.Getenv("DESIGN_BASE_URL")
	return &uaslDesignOpenAPIGateway{
		client:  httpClient.NewHttpClientWithTimeout(uaslDefaultTimeout),
		baseURL: baseURL,
	}
}

func (g *uaslDesignOpenAPIGateway) resolveBaseURL(externalBaseURL string) string {
	if externalBaseURL != "" {
		return externalBaseURL
	}
	return g.baseURL
}

func (g *uaslDesignOpenAPIGateway) doRequest(httpReq *http.Request) (*http.Response, error) {
	if g.client != nil {
		return g.client.DoRequest(&httpClient.HttpRequest{Request: httpReq})
	}
	hc := httpClient.NewHttpClientWithTimeout(uaslDefaultTimeout)
	return hc.DoRequest(&httpClient.HttpRequest{Request: httpReq})
}

func (g *uaslDesignOpenAPIGateway) newJSONRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
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

func (g *uaslDesignOpenAPIGateway) FetchUaslList(ctx context.Context, baseURL string, isInternal bool) (*model.UaslBulkData, error) {
	base := g.resolveBaseURL(baseURL)
	requestURL := fmt.Sprintf("%s%s?all=true", strings.TrimRight(base, "/"), pathUaslList)

	httpReq, err := g.newJSONRequest(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, myerror.NewHTTPError(resp.StatusCode, "uasl design API error", string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var topEntity conformityAssessmentModel.UaslDesignTopEntity
	if err := json.Unmarshal(bodyBytes, &topEntity); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	bulkData := converter.ToUaslBulkDataFromDTO(topEntity, isInternal, base)

	return bulkData, nil
}

func (g *uaslDesignOpenAPIGateway) FetchUaslByID(ctx context.Context, baseURL string, uaslID string, isInternal bool) (*model.UaslBulkData, error) {
	base := g.resolveBaseURL(baseURL)
	requestURL := fmt.Sprintf("%s%s?uaslId=%s", strings.TrimRight(base, "/"), pathUaslList, uaslID)

	httpReq, err := g.newJSONRequest(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, myerror.NewHTTPError(resp.StatusCode, "uasl design API error", string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var topEntity conformityAssessmentModel.UaslDesignTopEntity
	if err := json.Unmarshal(bodyBytes, &topEntity); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	bulkData := converter.ToUaslBulkDataFromDTO(topEntity, isInternal, base)

	return bulkData, nil
}
