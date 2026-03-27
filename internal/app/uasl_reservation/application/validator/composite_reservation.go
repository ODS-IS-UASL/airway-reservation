package validator

import (
	"context"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/pkg/logger"
)

type CompositeReservationValidator struct {
	vehicleValidator *vehicleReservation
	portValidator    *portReservation
}

func NewCompositeReservationValidator() *CompositeReservationValidator {
	return &CompositeReservationValidator{
		vehicleValidator: NewVehicleReservation(),
		portValidator:    NewPortReservation(),
	}
}

func (v *CompositeReservationValidator) ValidateVehicleReservations(
	ctx context.Context,
	vehicleReqs []model.VehicleReservationRequest,
) error {
	if len(vehicleReqs) == 0 {
		return nil
	}

	for _, vehicleReq := range vehicleReqs {
		if err := v.vehicleValidator.ReservationRequest(ctx, vehicleReq); err != nil {
			logger.LogError("vehicle reservation validation failed", "vehicle_id", vehicleReq.VehicleID, "error", err)
			return err
		}
	}

	return nil
}

func (v *CompositeReservationValidator) ValidatePortReservations(
	ctx context.Context,
	portReservations []model.PortReservationRequest,
) error {
	if len(portReservations) == 0 {
		return nil
	}

	for _, portReq := range portReservations {
		if err := v.portValidator.ReservationRequest(ctx, portReq); err != nil {
			logger.LogError("port reservation validation failed", "port_id", portReq.PortID, "error", err)
			return err
		}
	}

	return nil
}

func (v *CompositeReservationValidator) ValidateCompositeReservations(
	ctx context.Context,
	vehicleReqs []model.VehicleReservationRequest,
	portReservations []model.PortReservationRequest,
) error {

	if err := v.ValidateVehicleReservations(ctx, vehicleReqs); err != nil {
		return err
	}

	return v.ValidatePortReservations(ctx, portReservations)
}
