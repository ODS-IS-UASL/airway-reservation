package baseValidator

import (
	"fmt"
	"reflect"
	"strings"

	"uasl-reservation/internal/pkg/myerror"
	"uasl-reservation/internal/pkg/value"

	"github.com/go-playground/validator"
)

func New() (*validator.Validate, error) {
	valid := validator.New()
	err := valid.RegisterValidation("model-id", modelIDValidation)
	if err != nil {
		return nil, err
	}
	err = valid.RegisterValidation("datetime", datetimeValidation)
	if err != nil {
		return nil, err
	}
	err = valid.RegisterValidation("update-pointer-uuid", updatePointerUUIDValidation)
	if err != nil {
		return nil, err
	}

	useJsonFieldName(valid)
	return valid, nil
}
func useJsonFieldName(validate *validator.Validate) {
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}
func modelIDValidation(fl validator.FieldLevel) bool {
	id := fl.Field().String()
	_, err := value.NewModelIDFromUUIDString(id)
	return err == nil
}
func datetimeValidation(fl validator.FieldLevel) bool {
	dtStr := fl.Field().String()
	_, err := value.NewTimestampFromString(dtStr)
	return err == nil
}

// Update時に使う任意項目uuid用
// nilと空文字以外の時にUUIDの形式をチェックする
func updatePointerUUIDValidation(fl validator.FieldLevel) bool {
	field := fl.Field()

	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return true
		}

		field = field.Elem()
	}

	if field.Kind() != reflect.String {
		return false
	}
	id := field.String()

	if id == "" {
		return true
	}

	_, err := value.NewModelIDFromUUIDString(id)
	return err == nil
}

func CustomErrorMessage(err error) error {
	if err == nil {
		return nil
	}
	for _, fieldErr := range err.(validator.ValidationErrors) {
		switch fieldErr.ActualTag() {
		case "required":
			return myerror.Wrap(myerror.BadRequest, err, fmt.Sprintf("%s is required", fieldErr.Field()))
		case "email":
			return myerror.Wrap(myerror.BadRequest, err, fmt.Sprintf("%s is invalid email format", fieldErr.Field()))
		case "max":
			if fieldErr.Kind() == reflect.String {
				return myerror.Wrap(myerror.BadRequest, err,
					fmt.Sprintf("%s must be %s characters or less", fieldErr.Field(), fieldErr.Param()))
			}
			if fieldErr.Kind() == reflect.Float64 {
				return myerror.Wrap(myerror.BadRequest, err,
					fmt.Sprintf("%s must be less than or equal to %s", fieldErr.Field(), fieldErr.Param()))
			}
		case "min":
			return myerror.Wrap(myerror.BadRequest, err,
				fmt.Sprintf("%s must be greater than or equal to %s", fieldErr.Field(), fieldErr.Param()))

		case "model-id", "update-pointer-uuid":
			return myerror.Wrap(myerror.BadRequest, err,
				fmt.Sprintf("%s %s is invalid", fieldErr.Field(), fieldErr.Value()))
		case "oneof":
			return myerror.Wrap(myerror.BadRequest, err, fmt.Sprintf("%s is invalid input value", fieldErr.Field()))
		case "numeric":
			return myerror.Wrap(myerror.BadRequest, err, fmt.Sprintf("%s is not a numeric", fieldErr.Field()))
		case "datetime":
			return myerror.Wrap(myerror.BadRequest, err, fmt.Sprintf("%s is not a RFC3339 format", fieldErr.Field()))
		}
	}
	return err
}
