package uaslValidator

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator"
)

type CreateUaslReservationInput struct {
	ExUaslSections []string `json:"exUaslSections"`
	AcceptedAt     string   `json:"acceptedAt"`
	ReservedBy     string   `json:"reservedBy"`
	AirspaceID     string   `json:"airspaceId"`
	Status         string   `json:"status"`
}

type UpdateUaslReservationInput struct {
	UaslReservationID string `json:"uaslReservationId"`
	Status            string `json:"status"`
}

func New(modelType interface{}) (*validator.Validate, error) {
	valid := validator.New()
	valid.RegisterStructValidation(validateUaslStruct, modelType)
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

func CustomErrorMessage(err error) error {
	if err == nil {
		return nil
	}
	for _, fieldErr := range err.(validator.ValidationErrors) {
		switch fieldErr.Tag() {
		case "datetime-format":
			return fmt.Errorf("%s の形式が不正です", fieldErr.Field())
		case "datetime-after-now":
			return fmt.Errorf("start_at は現在時刻以降である必要があります")
		case "datetime-order":
			return fmt.Errorf("end_at は start_at より後である必要があります")
		case "required":
			return fmt.Errorf("%s フィールドが存在しません", fieldErr.Field())
		}
	}
	return err
}

func validateUaslStruct(sl validator.StructLevel) {
	validUaslDates(sl)
}

func validUaslDates(sl validator.StructLevel) {
	param := sl.Current().Interface()

	startAtStr := reflect.ValueOf(param).FieldByName("StartAt").String()
	endAtStr := reflect.ValueOf(param).FieldByName("EndAt").String()
	acceptedAtStr := reflect.ValueOf(param).FieldByName("AcceptedAt").String()

	startAt, err := time.Parse(time.RFC3339, startAtStr)
	if err != nil {
		sl.ReportError(param, "startAt", "StartAt", "datetime-format", "")
		return
	}

	endAt, err := time.Parse(time.RFC3339, endAtStr)
	if err != nil {
		sl.ReportError(endAt, "endAt", "EndAt", "datetime-format", "")
		return
	}

	acceptedAt, err := time.Parse(time.RFC3339, acceptedAtStr)
	if err != nil {
		sl.ReportError(acceptedAt, "acceptedAt", "AcceptedAt", "datetime-format", "")
		return
	}

	if startAt.Before(time.Now()) {
		sl.ReportError(startAt, "startAt", "StartAt", "datetime-after-now", "")
	}

	if endAt.Before(startAt) {
		sl.ReportError(endAt, "endAt", "EndAt", "datetime-order", "")
	}
}
