package validator

import (
	"context"
	"fmt"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/pkg/myvalidator/baseValidator"
)

type portReservation struct{}

func NewPortReservation() *portReservation {
	return &portReservation{}
}

func (portReservation) ReservationRequest(ctx context.Context, req model.PortReservationRequest) error {
	type valid struct {
		PortID     string `json:"drone_port_id" validate:"required"`
		AircraftID string `json:"aircraft_id" validate:"omitempty"`
		UsageType  int32  `json:"usage_type" validate:"required,min=1,max=3"`
		OperatorID string `json:"operator_id" validate:"omitempty"`
	}

	v := valid{
		PortID:    req.PortID,
		UsageType: req.UsageType,
	}

	if req.AircraftID != nil {
		v.AircraftID = *req.AircraftID
	}
	if req.OperatorID != nil {
		v.OperatorID = req.OperatorID.ToString()
	}

	validate, err := baseValidator.New()
	if err != nil {
		return err
	}

	err = validate.Struct(v)
	if err != nil {
		return baseValidator.CustomErrorMessage(err)
	}

	if !req.ReservationTimeFrom.IsZero() && !req.ReservationTimeTo.IsZero() {
		if req.ReservationTimeTo.Before(req.ReservationTimeFrom) || req.ReservationTimeTo.Equal(req.ReservationTimeFrom) {
			return fmt.Errorf("reservation_time_to must be after reservation_time_from: from=%v, to=%v", req.ReservationTimeFrom, req.ReservationTimeTo)
		}
	}

	return nil
}

func (portReservation) FetchRequest(ctx context.Context, req model.PortFetchRequest) error {
	type valid struct {
		PortID string `json:"drone_port_id" validate:"required,model-id"`
	}

	v := valid{
		PortID: req.PortID,
	}

	validate, err := baseValidator.New()
	if err != nil {
		return err
	}

	err = validate.Struct(v)
	if err != nil {
		return baseValidator.CustomErrorMessage(err)
	}

	if req.TimeFrom != nil && req.TimeTo != nil {
		if req.TimeTo.Before(*req.TimeFrom) || req.TimeTo.Equal(*req.TimeFrom) {
			return fmt.Errorf("time_to must be after time_from: from=%v, to=%v", *req.TimeFrom, *req.TimeTo)
		}
	}

	return nil
}
