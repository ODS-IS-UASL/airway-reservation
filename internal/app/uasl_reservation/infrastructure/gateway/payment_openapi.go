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

	ouranosModel "uasl-reservation/external/uasl/payment/model"
	"uasl-reservation/internal/app/uasl_reservation/application/converter"
	gatewayIF "uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	mock "uasl-reservation/internal/app/uasl_reservation/infrastructure/gateway/mock"
	httpClient "uasl-reservation/internal/pkg/http"
	"uasl-reservation/internal/pkg/logger"
)

const (
	paymentDefaultTimeout                 = 30 * time.Second
	pathTransactionEligibilityNonFeeModel = "/api/v1/data-exchange/non-fee-model/transaction/eligibility"
	pathTransactionConfirmNonFeeModel     = "/api/v1/data-exchange/non-fee-model/confirm"
	paymentUserAgent                      = "uasl-reservations/1.0.0"
	paymentAcceptLanguage                 = "ja-JP"
)

type paymentGateway struct {
	baseURL   string
	apiKey    string
	client    *httpClient.HttpClient
	converter *converter.PaymentConverter
}

func NewPaymentGateway() gatewayIF.PaymentGatewayIF {
	baseURL := os.Getenv("PAYMENT_API_BASE_URL")
	apiKey := os.Getenv("PAYMENT_API_KEY")

	if os.Getenv("APP_ENV") == "local" {
		logger.LogInfo("Using payment mock gateway (APP_ENV=local)")
		return mock.NewPaymentGatewayLocalMock()
	}

	return &paymentGateway{
		baseURL:   baseURL,
		apiKey:    apiKey,
		client:    httpClient.NewHttpClientWithTimeout(paymentDefaultTimeout),
		converter: converter.NewPaymentConverter(),
	}
}

func (g *paymentGateway) newPaymentRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", paymentUserAgent)
	httpReq.Header.Set("Accept-Language", paymentAcceptLanguage)
	httpReq.Header.Set("X-TrackingId", uuid.New().String())

	if authToken, ok := getAuthorizationFromContext(ctx); ok {
		httpReq.Header.Set("Authorization", authToken)
	}

	if g.apiKey != "" {
		httpReq.Header.Set("x-payment-api-key", g.apiKey)
	} else {
		logger.LogError("PAYMENT_API_KEY not configured")
	}

	return httpReq, nil
}

func (g *paymentGateway) doRequest(httpReq *http.Request) (*http.Response, error) {
	if g.client != nil {
		return g.client.DoRequest(&httpClient.HttpRequest{Request: httpReq})
	}
	hc := httpClient.NewHttpClientWithTimeout(paymentDefaultTimeout)
	return hc.DoRequest(&httpClient.HttpRequest{Request: httpReq})
}

func (g *paymentGateway) CheckTransactionEligibility(
	ctx context.Context,
	req model.TransactionEligibilityRequest,
) (*model.TransactionEligibilityResponse, error) {
	logger.LogInfo(fmt.Sprintf("CheckTransactionEligibility called: provider=%s, consumer=%s, amount=%d",
		req.ProviderID, req.ConsumerID, req.Amount))

	externalReq := g.converter.ToExternalTransactionEligibilityRequest(req)

	reqBody, err := json.Marshal(externalReq)
	if err != nil {
		logger.LogError("Failed to marshal transaction eligibility request", "error", err.Error())
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	requestURL := buildRequestURL(g.baseURL, pathTransactionEligibilityNonFeeModel)
	httpReq, err := g.newPaymentRequest(ctx, http.MethodPost, requestURL, reqBody)
	if err != nil {
		logger.LogError("Failed to create HTTP request", "error", err.Error())
		return nil, err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		logger.LogError("Failed to call payment API", "error", err.Error())
		return nil, fmt.Errorf("failed to call payment API: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.LogError("Failed to read response body", "error", err.Error())
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.LogError("Payment API returned error status",
			"status_code", resp.StatusCode,
			"body", string(bodyBytes))
		return nil, fmt.Errorf("payment API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var externalResponse ouranosModel.TransactionEligibilityResponse
	if err := json.Unmarshal(bodyBytes, &externalResponse); err != nil {
		logger.LogError("Failed to unmarshal response",
			"error", err.Error(),
			"body", string(bodyBytes))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	internalResponse := g.converter.ToInternalTransactionEligibilityResponse(externalResponse)

	logger.LogInfo(fmt.Sprintf("CheckTransactionEligibility completed: status=%s, detail=%s",
		internalResponse.Status, internalResponse.Detail))

	return &internalResponse, nil
}

func (g *paymentGateway) ConfirmTransaction(
	ctx context.Context,
	baseURL string,
	req model.TransactionConfirmRequest,
) (*model.TransactionConfirmResponse, error) {
	logger.LogInfo(fmt.Sprintf("ConfirmTransaction called: provider=%s, consumer=%s, amount=%d",
		req.ProviderID, req.ConsumerID, req.Amount))

	externalReq := g.converter.ToExternalTransactionConfirmRequest(req)

	reqBody, err := json.Marshal(externalReq)
	if err != nil {
		logger.LogError("Failed to marshal transaction confirm request", "error", err.Error())
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resolvedBaseURL := g.baseURL
	if baseURL != "" {
		resolvedBaseURL = baseURL
	}
	requestURL := fmt.Sprintf("%s%s", strings.TrimRight(resolvedBaseURL, "/"), pathTransactionConfirmNonFeeModel)

	httpReq, err := g.newPaymentRequest(ctx, http.MethodPost, requestURL, reqBody)
	if err != nil {
		logger.LogError("Failed to create HTTP request for confirm", "error", err.Error())
		return nil, err
	}

	resp, err := g.doRequest(httpReq)
	if err != nil {
		logger.LogError("Failed to call payment confirm API", "error", err.Error())
		return nil, fmt.Errorf("failed to call payment confirm API: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.LogError("Failed to read confirm response body", "error", err.Error())
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.LogError("Payment confirm API returned error status",
			"status_code", resp.StatusCode,
			"body", string(bodyBytes))
		return nil, fmt.Errorf("payment confirm API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var externalResponse ouranosModel.TransactionConfirmResponse
	if err := json.Unmarshal(bodyBytes, &externalResponse); err != nil {
		logger.LogError("Failed to unmarshal confirm response",
			"error", err.Error(),
			"body", string(bodyBytes))
		return nil, fmt.Errorf("failed to decode confirm response: %w", err)
	}

	internalResponse := g.converter.ToInternalTransactionConfirmResponse(externalResponse)

	logger.LogInfo(fmt.Sprintf("ConfirmTransaction completed: status=%s, detail=%s",
		internalResponse.Status, internalResponse.Detail))

	return &internalResponse, nil
}
