package myerror

import "fmt"

type HTTPError struct {
	statusCode int
	message    string
	body       string
}

func NewHTTPError(statusCode int, message string, body string) *HTTPError {
	return &HTTPError{
		statusCode: statusCode,
		message:    message,
		body:       body,
	}
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s (body: %s)", e.statusCode, e.message, e.body)
}

func (e *HTTPError) StatusCode() int {
	return e.statusCode
}
