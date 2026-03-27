package validator

import (
	"context"
	"fmt"
	"time"

	"uasl-reservation/internal/app/uasl_reservation/application/converter"
	"uasl-reservation/internal/pkg/logger"
	"uasl-reservation/internal/pkg/myvalidator/baseValidator"
	"uasl-reservation/internal/pkg/value"
)

type uaslReservation struct{}

func NewUaslReservation() *uaslReservation {
	return &uaslReservation{}
}

func (uaslReservation) RegisterRequest(ctx context.Context, req *converter.RegisterUaslReservationInput) error {
	logger.LogInfo("Validate RegisterRequest - req: %+v", req)
	type valid struct {
		ParentUaslReservationID string                  `json:"parent_uasl_reservation_id" validate:"omitempty,model-id"`
		ExUaslSectionID         string                  `json:"ex_uasl_section_id" validate:"omitempty"`
		StartAt                 time.Time               `json:"start_at" validate:"required"`
		EndAt                   time.Time               `json:"end_at" validate:"required"`
		AcceptedAt              *time.Time              `json:"accepted_at" validate:"omitempty"`
		ExReservedBy            string                  `json:"ex_reserved_by" validate:"omitempty,model-id"`
		AirspaceID              string                  `json:"airspace_id" validate:"required,model-id"`
		ProjectID               string                  `json:"project_id" validate:"omitempty,model-id"`
		OperationID             string                  `json:"operation_id" validate:"omitempty,model-id"`
		OrganizationID          string                  `json:"organization_id" validate:"omitempty,model-id"`
		Status                  value.ReservationStatus `json:"status" validate:"required"`
	}
	StartAt, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		return fmt.Errorf("invalid StartAt time format: %v", err)
	}
	EndAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		return fmt.Errorf("invalid EndAt time format: %v", err)
	}

	var acceptedAtPtr *time.Time
	if req.AcceptedAt != "" {
		acceptedAt, err := time.Parse(time.RFC3339, req.AcceptedAt)
		if err != nil {
			return fmt.Errorf("invalid AcceptedAt time format: %v", err)
		}
		acceptedAtPtr = &acceptedAt
	}

	v := valid{
		ParentUaslReservationID: req.ParentUaslReservationID,
		ExUaslSectionID:         req.ExUaslSectionID,
		StartAt:                 StartAt,
		EndAt:                   EndAt,
		AcceptedAt:              acceptedAtPtr,
		ExReservedBy:            req.ExReservedBy,
		AirspaceID:              req.AirspaceID,
		ProjectID:               req.ProjectID,
		OperationID:             req.OperationID,
		OrganizationID:          req.OrganizationID,
		Status:                  value.ReservationStatus(req.Status),
	}

	if !EndAt.After(StartAt) {
		return fmt.Errorf("EndAt must be after StartAt")
	}

	if v.Status != value.RESERVATION_STATUS_PENDING && v.AcceptedAt == nil {
		return fmt.Errorf("AcceptedAt must be set when status is not PENDING")
	}
	validate, err := baseValidator.New()
	if err != nil {
		return err
	}
	err = validate.Struct(v)
	if err != nil {
		return baseValidator.CustomErrorMessage(err)
	}
	return nil
}

func (uaslReservation) CancelRequest(ctx context.Context, req *converter.CancelUaslReservationInput) error {
	type valid struct {
		ID     string                  `json:"id" validate:"required,model-id"`
		Status value.ReservationStatus `json:"status" validate:"omitempty"`
	}
	v := valid{
		ID:     req.ID,
		Status: value.ReservationStatus(req.Status),
	}
	validate, err := baseValidator.New()
	if err != nil {
		return err
	}
	err = validate.Struct(v)
	if err != nil {
		return baseValidator.CustomErrorMessage(err)
	}
	return nil
}

func (uaslReservation) ConfirmRequest(ctx context.Context, req *converter.ConfirmUaslReservationInput) error {
	type valid struct {
		ID     string                  `json:"id" validate:"required,model-id"`
		Status value.ReservationStatus `json:"status" validate:"omitempty"`
	}

	v := valid{
		ID:     req.ID,
		Status: value.ReservationStatus(req.Status),
	}

	validate, err := baseValidator.New()
	if err != nil {
		return err
	}
	err = validate.Struct(v)
	if err != nil {
		return baseValidator.CustomErrorMessage(err)
	}
	return nil
}

func (uaslReservation) AvailabilityRequest(ctx context.Context, req *converter.GetAvailabilityInput) error {
	logger.LogInfo("Validate AvailabilityRequest - sections count: %d, isInterConnect: %v",
		len(req.UaslSections), req.IsInterConnect)

	if len(req.UaslSections) == 0 {
		return fmt.Errorf("uaslSections must not be empty")
	}

	type sectionValid struct {
		UaslID        string `json:"uasl_id" validate:"required"`
		UaslSectionID string `json:"uasl_section_id" validate:"required"`
	}

	validate, err := baseValidator.New()
	if err != nil {
		return err
	}

	for i, s := range req.UaslSections {
		v := sectionValid{
			UaslID:        s.UaslID,
			UaslSectionID: s.UaslSectionID,
		}
		if err := validate.Struct(v); err != nil {
			return fmt.Errorf("uaslSections[%d]: %w", i, baseValidator.CustomErrorMessage(err))
		}
	}

	return nil
}

func (uaslReservation) SearchByConditionRequest(ctx context.Context, req *converter.SearchByConditionInput) error {
	logger.LogInfo("Validate SearchByConditionRequest - requestIds count: %d", len(req.RequestIDs))

	if len(req.RequestIDs) == 0 {
		return fmt.Errorf("requestIds must not be empty")
	}

	type requestIdValid struct {
		RequestID string `json:"request_id" validate:"required,model-id"`
	}

	validate, err := baseValidator.New()
	if err != nil {
		return err
	}

	for i, rid := range req.RequestIDs {
		v := requestIdValid{RequestID: rid}
		if err := validate.Struct(v); err != nil {
			return fmt.Errorf("requestIds[%d]: %w", i, baseValidator.CustomErrorMessage(err))
		}
	}

	return nil
}

func (uaslReservation) DeleteRequest(ctx context.Context, req *converter.DeleteUaslReservationInput) error {
	type valid struct {
		ID string `json:"id" validate:"required,model-id"`
	}
	v := valid{
		ID: req.ID,
	}
	validate, err := baseValidator.New()
	if err != nil {
		return err
	}
	err = validate.Struct(v)
	if err != nil {
		return baseValidator.CustomErrorMessage(err)
	}
	return nil
}

func (uaslReservation) EstimateRequest(ctx context.Context, req *converter.EstimateUaslReservationInput) ([]string, []string, []string, error) {

	if req == nil {
		return nil, nil, nil, fmt.Errorf("request is required")
	}

	if len(req.UaslSections) == 0 {
		return nil, nil, nil, fmt.Errorf("uaslSections is required")
	}

	type validSection struct {
		ExUaslSectionID string `json:"ex_uasl_section_id" validate:"required"`
		StartAt         string `json:"start_at" validate:"required,datetime"`
		EndAt           string `json:"end_at" validate:"required,datetime"`
	}

	validate, err := baseValidator.New()
	if err != nil {
		return nil, nil, nil, err
	}

	sectionIDs := make([]string, 0, len(req.UaslSections))
	for i, s := range req.UaslSections {
		v := validSection{
			ExUaslSectionID: s.ExUaslSectionID,
			StartAt:         s.StartAt,
			EndAt:           s.EndAt,
		}
		if err := validate.Struct(v); err != nil {
			return nil, nil, nil, fmt.Errorf("validation failed: %w", baseValidator.CustomErrorMessage(err))
		}
		start, _ := time.Parse(time.RFC3339, s.StartAt)
		end, _ := time.Parse(time.RFC3339, s.EndAt)
		if !end.After(start) {
			return nil, nil, nil, fmt.Errorf("uaslSections[%d]: end_at は start_at より後である必要があります", i)
		}
		sectionIDs = append(sectionIDs, s.ExUaslSectionID)
	}

	vehicleIDs := make([]string, 0)
	portIDs := make([]string, 0)
	if len(req.Vehicles) > 0 {
		for i, v := range req.Vehicles {
			if v.VehicleID == "" {
				return nil, nil, nil, fmt.Errorf("vehicles[%d].vehicleId is required", i)
			}
			if s := v.StartAt; s != "" {
				if _, err := time.Parse(time.RFC3339, s); err != nil {
					return nil, nil, nil, fmt.Errorf("vehicles[%d].startAt invalid RFC3339: %v", i, err)
				}
			}
			if e := v.EndAt; e != "" {
				if _, err := time.Parse(time.RFC3339, e); err != nil {
					return nil, nil, nil, fmt.Errorf("vehicles[%d].endAt invalid RFC3339: %v", i, err)
				}
			}
			vehicleIDs = append(vehicleIDs, v.VehicleID)
		}
	}
	if len(req.Ports) > 0 {
		for i, p := range req.Ports {
			if p.PortID == "" {
				return nil, nil, nil, fmt.Errorf("ports[%d].portId is required", i)
			}
			if s := p.StartAt; s != "" {
				if _, err := time.Parse(time.RFC3339, s); err != nil {
					return nil, nil, nil, fmt.Errorf("ports[%d].startAt invalid RFC3339: %v", i, err)
				}
			}
			if e := p.EndAt; e != "" {
				if _, err := time.Parse(time.RFC3339, e); err != nil {
					return nil, nil, nil, fmt.Errorf("ports[%d].endAt invalid RFC3339: %v", i, err)
				}
			}
			portIDs = append(portIDs, p.PortID)
		}
	}

	return sectionIDs, vehicleIDs, portIDs, nil
}
