package http

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type HttpClient struct {
	client *http.Client
}

type HttpRequest struct {
	Url     string
	Request *http.Request
}

func NewHttpClient() *HttpClient {
	return &HttpClient{
		client: http.DefaultClient,
	}
}

func NewHttpClientWithTimeout(timeout time.Duration) *HttpClient {
	return &HttpClient{
		client: &http.Client{Timeout: timeout},
	}
}

func NewHttpRequest() *HttpRequest {
	return &HttpRequest{}
}

func (r *HttpRequest) CreateUrl(base string, path string, pathParam string) error {
	u, err := url.JoinPath(base, path, pathParam)
	if err != nil {
		return err
	}
	r.Url = u
	return nil
}
func (r *HttpRequest) CreateReqest(method string, url string, body io.Reader) error {
	if method == "" || url == "" {
		return fmt.Errorf("method or url is empty")
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	r.Request = req
	return nil
}

func (c *HttpClient) DoRequest(r *HttpRequest) (*http.Response, error) {
	if r == nil || r.Request == nil {
		return nil, fmt.Errorf("http request is nil")
	}

	if r.Request.URL != nil {
		if err := ValidateURL(r.Request.URL.String()); err != nil {
			return nil, fmt.Errorf("unsafe request url: %w", err)
		}
	}

	resp, err := c.client.Do(r.Request)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (r *HttpRequest) GetHttpMethodForEvent(eventName string) string {

	switch eventName {
	default:
		return ""
	}
}
