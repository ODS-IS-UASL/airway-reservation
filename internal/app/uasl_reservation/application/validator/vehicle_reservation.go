package validator

import (
	"context"
	"fmt"
	"time"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/pkg/myvalidator/baseValidator"
)

type vehicleReservation struct{}

func NewVehicleReservation() *vehicleReservation {
	return &vehicleReservation{}
}

func (vehicleReservation) ReservationRequest(ctx context.Context, req model.VehicleReservationRequest) error {
	type valid struct {
		VehicleID  string    `json:"vehicle_id" validate:"required"`
		OperatorID string    `json:"operator_id" validate:"required"`
		StartAt    time.Time `json:"start_at" validate:"required"`
		EndAt      time.Time `json:"end_at" validate:"required"`
	}

	v := valid{
		VehicleID:  req.VehicleID,
		OperatorID: req.OperatorID.ToString(),
		StartAt:    req.StartAt,
		EndAt:      req.EndAt,
	}

	validate, err := baseValidator.New()
	if err != nil {
		return err
	}

	err = validate.Struct(v)
	if err != nil {
		return baseValidator.CustomErrorMessage(err)
	}

	if req.EndAt.Before(req.StartAt) || req.EndAt.Equal(req.StartAt) {
		return fmt.Errorf("end_at must be after start_at: start_at=%v, end_at=%v", req.StartAt, req.EndAt)
	}
	if req.AircraftInfo == nil {
		return fmt.Errorf("aircraft_info is required")
	}
	if req.AircraftInfo.Maker == "" {
		return fmt.Errorf("aircraft_info.maker is required")
	}
	if req.AircraftInfo.ModelNumber == "" {
		return fmt.Errorf("aircraft_info.model_number is required")
	}
	if req.AircraftInfo.Name == "" {
		return fmt.Errorf("aircraft_info.name is required")
	}
	if req.AircraftInfo.Type == "" {
		return fmt.Errorf("aircraft_info.type is required")
	}
	if req.AircraftInfo.Length == "" {
		return fmt.Errorf("aircraft_info.length is required")
	}

	return nil
}

func (vehicleReservation) FetchRequest(ctx context.Context, req model.VehicleFetchRequest) error {
	type valid struct {
		VehicleID string `json:"vehicle_id" validate:"required"`
	}

	v := valid{
		VehicleID: req.VehicleID,
	}

	validate, err := baseValidator.New()
	if err != nil {
		return err
	}

	err = validate.Struct(v)
	if err != nil {
		return baseValidator.CustomErrorMessage(err)
	}

	if req.StartAt != nil && req.EndAt != nil {
		if req.EndAt.Before(*req.StartAt) || req.EndAt.Equal(*req.StartAt) {
			return fmt.Errorf("end_at must be after start_at: start_at=%v, end_at=%v", *req.StartAt, *req.EndAt)
		}
	}

	return nil
}
