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

	"github.com/google/uuid"

	"uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	mock "uasl-reservation/internal/app/uasl_reservation/infrastructure/gateway/mock"
	httpClient "uasl-reservation/internal/pkg/http"
	"uasl-reservation/internal/pkg/logger"
)

const (
	l3AuthDefaultTimeout   = 30 * time.Second
	l3AuthTokenPath        = "/auth/token/client"
	l3TokenIntrospectPath  = "/auth/token/introspect"
	l3AuthDefaultLanguage  = "ja-JP"
	l3AuthDefaultUserAgent = "monthly-settlement-worker/1.0.0"
)

type ouranosL3AuthGateway struct {
	authURL        string
	apiKey         string
	clientID       string
	clientSecret   string
	client         *httpClient.HttpClient
	cachedToken    string
	tokenExpiresAt time.Time
}

type l3AuthTokenResponse struct {
	Data struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	} `json:"data"`
}

type l3IntrospectResponse struct {
	Data struct {
		Active bool `json:"active"`
	} `json:"data"`
}

func NewOuranosL3AuthGateway() gatewayIF.OuranosL3AuthGatewayIF {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "local" {
		logger.LogInfo("Using L3 auth mock gateway (APP_ENV=%s)", appEnv)
		return mock.NewOuranosL3AuthGatewayLocalMock()
	}

	baseURL := os.Getenv("L3_BASE_URL")
	apiKey := os.Getenv("L3_API_KEY")
	clientID := os.Getenv("L3_CLIENT_ID")
	clientSecret := os.Getenv("L3_CLIENT_SECRET")

	if baseURL == "" {
		logger.LogError("L3_BASE_URL is not set, L3 auth gateway may not work properly")
	}

	authURL := strings.TrimRight(baseURL, "/") + l3AuthTokenPath

	return &ouranosL3AuthGateway{
		authURL:      authURL,
		apiKey:       apiKey,
		clientID:     clientID,
		clientSecret: clientSecret,
		client:       httpClient.NewHttpClientWithTimeout(l3AuthDefaultTimeout),
	}
}

func (g *ouranosL3AuthGateway) GetAccessToken(ctx context.Context) (string, error) {

	if g.cachedToken != "" && !g.tokenExpiresAt.IsZero() && time.Now().Before(g.tokenExpiresAt) {
		return g.cachedToken, nil
	}

	logger.LogInfo("L3認証トークン取得")

	reqBody, err := json.Marshal(map[string]string{
		"client_id":     g.clientID,
		"client_secret": g.clientSecret,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal L3 auth request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.authURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create L3 auth request: %w", err)
	}
	if err := g.applyCommonHeaders(httpReq); err != nil {
		return "", fmt.Errorf("failed to prepare L3 auth headers: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.DoRequest(&httpClient.HttpRequest{Request: httpReq})
	if err != nil {
		logger.LogError("L3認証: HTTP接続失敗: %v", err)
		return "", fmt.Errorf("failed to call L3 auth endpoint: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.LogError("L3認証: レスポンス読み取り失敗: %v", err)
		return "", fmt.Errorf("failed to read L3 auth response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.LogError("L3認証: ステータスエラー: status=%d, body=%s", resp.StatusCode, string(bodyBytes))
		return "", fmt.Errorf("L3 auth returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp l3AuthTokenResponse
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		logger.LogError("L3認証: レスポンスパース失敗: %v", err)
		return "", fmt.Errorf("failed to unmarshal L3 auth response: %w", err)
	}

	accessStr := strings.TrimSpace(tokenResp.Data.AccessToken)
	if accessStr == "" {
		logger.LogError("L3認証: access_tokenが空")
		return "", fmt.Errorf("access_token not found in L3 auth response")
	}

	expiresInSec := tokenResp.Data.ExpiresIn

	g.cachedToken = accessStr
	if expiresInSec > 0 {
		g.tokenExpiresAt = time.Now().Add(time.Duration(expiresInSec*9/10) * time.Second)
	} else {
		g.tokenExpiresAt = time.Now().Add(30 * time.Second)
	}

	logger.LogInfo("L3認証トークン取得成功")
	return accessStr, nil
}

func (g *ouranosL3AuthGateway) IntrospectToken(ctx context.Context, accessToken string) error {
	if accessToken == "" {
		return fmt.Errorf("access token is empty")
	}

	introspectURL, err := g.resolveIntrospectURL()
	if err != nil {
		return fmt.Errorf("failed to resolve introspect URL: %w", err)
	}

	reqBody, err := json.Marshal(map[string]string{
		"client_id":     g.clientID,
		"client_secret": g.clientSecret,
		"access_token":  accessToken,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal introspect request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, introspectURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create introspect request: %w", err)
	}
	if err := g.applyCommonHeaders(httpReq); err != nil {
		return fmt.Errorf("failed to prepare introspect headers: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.DoRequest(&httpClient.HttpRequest{Request: httpReq})
	if err != nil {
		return fmt.Errorf("failed to call token introspect endpoint: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("token introspect returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var introspectResp l3IntrospectResponse
	if err := json.Unmarshal(bodyBytes, &introspectResp); err != nil {
		return fmt.Errorf("failed to unmarshal introspect response: %w", err)
	}
	if !introspectResp.Data.Active {
		return fmt.Errorf("token is inactive")
	}

	logger.LogInfo("L3トークンイントロスペクション成功")
	return nil
}

func (g *ouranosL3AuthGateway) resolveIntrospectURL() (string, error) {
	parsed, err := url.Parse(g.authURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid OURANOS_L3_AUTH_URL")
	}
	parsed.Path = l3TokenIntrospectPath
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func (g *ouranosL3AuthGateway) applyCommonHeaders(req *http.Request) error {
	if strings.TrimSpace(g.apiKey) == "" {
		return fmt.Errorf("OURANOS_L3_API_KEY is not set")
	}
	req.Header.Set("API-Key", g.apiKey)
	req.Header.Set("User-Agent", l3AuthDefaultUserAgent)
	req.Header.Set("Accept-Language", l3AuthDefaultLanguage)
	req.Header.Set("X-TrackingID", uuid.New().String())
	return nil
}
