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

	safetymodel "uasl-reservation/external/uasl/conformity_assessment/model"
	"uasl-reservation/internal/app/uasl_reservation/application/converter"
	"uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	mock "uasl-reservation/internal/app/uasl_reservation/infrastructure/gateway/mock"
	httpClient "uasl-reservation/internal/pkg/http"
	"uasl-reservation/internal/pkg/logger"
)

const conformityAssessmentDefaultTimeout = 30 * time.Second

const (
	pathConformityAssessment = "/safetyManagement/conformity-assessment"
)

type conformityAssessmentGateway struct {
	baseURL   string
	client    *httpClient.HttpClient
	converter *converter.ConformityAssessmentConverter
}

func NewConformityAssessmentGateway() gatewayIF.ConformityAssessmentGatewayIF {
	baseURL := os.Getenv("SAFETY_BASE_URL")

	if os.Getenv("APP_ENV") == "local" && baseURL == "" {
		logger.LogInfo("Using ConformityAssessment gateway mock (APP_ENV=local, SAFETY_BASE_URL not set)")
		return mock.NewConformityAssessmentGatewayLocalMock()
	}

	return &conformityAssessmentGateway{
		baseURL:   baseURL,
		client:    httpClient.NewHttpClientWithTimeout(conformityAssessmentDefaultTimeout),
		converter: converter.NewConformityAssessmentConverter(),
	}
}

func (g *conformityAssessmentGateway) resolveBaseURL(baseURL string) string {
	if baseURL != "" {
		return baseURL
	}
	return g.baseURL
}

func (g *conformityAssessmentGateway) newJSONRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
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

func (g *conformityAssessmentGateway) doRequest(httpReq *http.Request) (*http.Response, error) {
	return g.client.DoRequest(&httpClient.HttpRequest{Request: httpReq})
}

func (g *conformityAssessmentGateway) PostConformityAssessment(ctx context.Context, baseURL string, req model.ConformityAssessmentRequest) (*model.ConformityAssessmentResponse, error) {
	logger.LogInfo("ConformityAssessment API: PostConformityAssessment uaslSectionId=%s", req.UaslSectionID)

	externalReq := g.converter.ToExternalRequest(req)

	reqBody, err := json.Marshal(externalReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal conformity assessment request: %w", err)
	}
	requestURL := buildRequestURL(g.resolveBaseURL(baseURL), pathConformityAssessment)
	httpReq, err := g.newJSONRequest(ctx, http.MethodPost, requestURL, reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call ConformityAssessment conformity assessment API: %w", err)
	}
	defer resp.Body.Close()
	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read ConformityAssessment conformity assessment response body: %w", readErr)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ConformityAssessment conformity assessment API returned status %d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var externalResp safetymodel.ConformityAssessmentResponse
	if err := json.Unmarshal(bodyBytes, &externalResp); err != nil {
		return nil, fmt.Errorf("failed to decode conformity assessment response: %w", err)
	}

	result := g.converter.ToInternalResponse(externalResp)

	return &result, nil
}
