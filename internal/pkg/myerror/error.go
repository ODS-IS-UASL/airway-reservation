package myerror

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

var ErrRecordNotFound = errors.New("record not found")

type CustomError struct {
	code        Code
	err         error
	infoMessage string // infoMessage show as client error message
}

// Unwrapメソッドを追加してエラーチェーンをサポート
func (e CustomError) Unwrap() error {
	return e.err
}

func (e CustomError) Error() string {
	return e.infoMessage
}
func Cause(err error) error {
	return errors.Cause(err)
}

func Errorf(c Code, format string, a ...interface{}) error {
	err := errors.Errorf(format, a...)
	return CustomError{
		code:        c,
		err:         err,
		infoMessage: err.Error(),
	}
}

func Wrap(c Code, err error, infoMessage string) error {
	return CustomError{
		code:        c,
		err:         errors.Wrap(err, infoMessage),
		infoMessage: infoMessage,
	}
}

func GetCode(err error) Code {
	var e CustomError
	if errors.As(err, &e) {
		return e.code
	}
	return Internal
}

func StackTrace(err error) string {
	var e CustomError
	if errors.As(err, &e) {
		return fmt.Sprintf("%+v\n", e.err)
	}
	return ""
}

func ToHTTPStatus(err error) int {
	switch GetCode(err) {
	case BadRequest:
		return http.StatusBadRequest
	case NotFound:
		return http.StatusNotFound
	case Auth:
		return http.StatusUnauthorized
	case Forbidden:
		return http.StatusForbidden
	case Conflict:
		return http.StatusConflict
	case Timeout:
		return http.StatusRequestTimeout
	case TooManyRequests:
		return http.StatusTooManyRequests
	case Connection:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func ConvertToHTTPError(err error) (int, string) {
	statusCode := ToHTTPStatus(err)
	return statusCode, err.Error()
}
