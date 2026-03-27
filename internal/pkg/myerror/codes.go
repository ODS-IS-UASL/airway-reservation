package myerror

import (
	"net/http"
)

type Code int

const (
	// 汎用エラー
	Internal Code = http.StatusInternalServerError

	// HTTP ステータスに準拠したエラー
	BadRequest      Code = http.StatusBadRequest      // 400 Bad Request
	BadParams       Code = http.StatusBadRequest      // 400 Bad Request
	Auth            Code = http.StatusUnauthorized    // 401 Unauthorized
	Forbidden       Code = http.StatusForbidden       // 403 Forbidden
	NotFound        Code = http.StatusNotFound        // 404 Not Found
	Conflict        Code = http.StatusConflict        // 409 Conflict
	Timeout         Code = http.StatusRequestTimeout  // 408 Request Timeout
	TooManyRequests Code = http.StatusTooManyRequests // 429 Too Many Requests

	// その他エラーコード
	Unknown    Code = Internal
	Database   Code = Internal
	Connection Code = http.StatusServiceUnavailable // 503
	Grpc       Code = Internal
)

var CodeToMessage = map[Code]string{
	BadRequest:      "Bad Request",
	Internal:        "Internal Server Error",
	Auth:            "Unauthorized",
	NotFound:        "Not Found",
	Conflict:        "Conflict",
	Forbidden:       "Forbidden",
	Timeout:         "Request Timeout",
	TooManyRequests: "Too Many Requests",
}
