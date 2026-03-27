package converter

import (
	"fmt"
	"time"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/pkg/logger"
	"uasl-reservation/internal/pkg/util"
	"uasl-reservation/internal/pkg/value"
)

type CompositeReservationRequestBuilder struct{}

func NewCompositeReservationRequestBuilder() *CompositeReservationRequestBuilder {
	return &CompositeReservationRequestBuilder{}
}

type CompositeReservationData struct {
	Vehicles         []model.VehicleReservationRequest
	PortReservations []model.PortReservationRequest
	UaslRequests     []model.UaslCheckRequest
}

type TryHoldVehicleInput struct {
	VehicleID    string
	StartAt      string
	EndAt        string
	AircraftInfo *model.VehicleDetailInfo
}

type TryHoldPortInput struct {
	PortID    string
	UsageType int32
	StartAt   string
	EndAt     string
}

type TryHoldCompositeUaslInput struct {
	ParentUaslReservation    *RegisterUaslReservationInput
	ChildUaslReservations    []*RegisterUaslReservationInput
	Vehicles                 []TryHoldVehicleInput
	Ports                    []TryHoldPortInput
	OperatingAircraft        *model.VehicleDetailInfo
	IgnoreFlightPlanConflict bool
	IsInterConnect           bool
}

func (b *CompositeReservationRequestBuilder) ToRequestData(
	req *TryHoldCompositeUaslInput,
	parentDomain *model.UaslReservation,
	childDomains []*model.UaslReservation,
) (*CompositeReservationData, error) {
	data := &CompositeReservationData{}

	data.Vehicles = make([]model.VehicleReservationRequest, 0, len(req.Vehicles))
	for _, vehicle := range req.Vehicles {
		var err error
		var startAt, endAt time.Time
		if s := vehicle.StartAt; s != "" {
			startAt, err = time.Parse(time.RFC3339, s)
			if err != nil {
				return nil, fmt.Errorf("invalid start_at format in vehicle: %w", err)
			}
		}
		if e := vehicle.EndAt; e != "" {
			endAt, err = time.Parse(time.RFC3339, e)
			if err != nil {
				return nil, fmt.Errorf("invalid end_at format in vehicle: %w", err)
			}
		}

		if startAt.IsZero() && parentDomain != nil {
			startAt = parentDomain.StartAt
		}
		if endAt.IsZero() && parentDomain != nil {
			endAt = parentDomain.EndAt
		}

		data.Vehicles = append(data.Vehicles, model.VehicleReservationRequest{
			VehicleID:          vehicle.VehicleID,
			OperatorID:         parentDomain.ExReservedBy,
			GroupReservationID: parentDomain.RequestID.ToString(),
			AircraftInfo:       vehicle.AircraftInfo,
			StartAt:            startAt,
			EndAt:              endAt,
		})
	}

	data.PortReservations = make([]model.PortReservationRequest, 0, len(req.Ports))
	for _, port := range req.Ports {
		pidStr := port.PortID

		var portStartAt, portEndAt time.Time
		if s := port.StartAt; s != "" {
			var parseErr error
			portStartAt, parseErr = time.Parse(time.RFC3339, s)
			if parseErr != nil {
				return nil, fmt.Errorf("invalid start_at format in port: %w", parseErr)
			}
		}
		if e := port.EndAt; e != "" {
			var parseErr error
			portEndAt, parseErr = time.Parse(time.RFC3339, e)
			if parseErr != nil {
				return nil, fmt.Errorf("invalid end_at format in port: %w", parseErr)
			}
		}

		if portStartAt.IsZero() || portEndAt.IsZero() {
			if parentDomain != nil {
				const portMargin = 15 * time.Minute
				totalDuration := parentDomain.EndAt.Sub(parentDomain.StartAt)
				usageType := port.UsageType
				if totalDuration < portMargin {

					if portStartAt.IsZero() {
						portStartAt = parentDomain.StartAt
					}
					if portEndAt.IsZero() {
						portEndAt = parentDomain.EndAt
					}
				} else {
					switch usageType {
					case 1:
						if portStartAt.IsZero() {
							portStartAt = parentDomain.StartAt
						}
						if portEndAt.IsZero() {
							portEndAt = parentDomain.StartAt.Add(portMargin)
						}
					case 2:
						if portStartAt.IsZero() {
							portStartAt = parentDomain.EndAt.Add(-portMargin)
						}
						if portEndAt.IsZero() {
							portEndAt = parentDomain.EndAt
						}
					default:
						if portStartAt.IsZero() {
							portStartAt = parentDomain.StartAt
						}
						if portEndAt.IsZero() {
							portEndAt = parentDomain.EndAt
						}
					}
				}
			}
		}

		var aircraftID *string
		if len(data.Vehicles) > 0 {
			firstID := data.Vehicles[0].VehicleID
			aircraftID = &firstID
		}

		data.PortReservations = append(data.PortReservations, model.PortReservationRequest{
			PortID:              pidStr,
			AircraftID:          aircraftID,
			GroupReservationID:  parentDomain.RequestID.ToString(),
			UsageType:           port.UsageType,
			ReservationTimeFrom: portStartAt,
			ReservationTimeTo:   portEndAt,
			OperatorID:          parentDomain.ExReservedBy,
		})
	}

	data.UaslRequests = make([]model.UaslCheckRequest, 0, len(childDomains))
	for _, childDomain := range childDomains {
		if childDomain.ExUaslSectionID != nil {
			data.UaslRequests = append(data.UaslRequests, model.UaslCheckRequest{
				ExUaslSectionID: *childDomain.ExUaslSectionID,
				TimeRange: model.TimeRange{
					Start: childDomain.StartAt,
					End:   childDomain.EndAt,
				},
			})
		}
	}

	return data, nil
}

type RegisterUaslReservationInput struct {
	RequestID               string
	ExUaslSectionID         string
	ExUaslID                string
	StartAt                 string
	EndAt                   string
	AcceptedAt              string
	ExReservedBy            string
	AirspaceID              string
	ProjectID               string
	OperationID             string
	OrganizationID          string
	ExAdministratorID       string
	Status                  string
	Sequence                int32
	FlightPurpose           string
	ParentUaslReservationID string
}

type EstimateUaslSectionInput struct {
	ExUaslSectionID string
	ExUaslID        string
	StartAt         string
	EndAt           string
}

type EstimateVehicleInput struct {
	VehicleID string
	StartAt   string
	EndAt     string
}

type EstimatePortInput struct {
	PortID  string
	StartAt string
	EndAt   string
}

type EstimateUaslReservationInput struct {
	UaslSections   []EstimateUaslSectionInput
	Vehicles       []EstimateVehicleInput
	Ports          []EstimatePortInput
	IsInterConnect bool
}

type CancelUaslReservationInput struct {
	ID             string
	Status         string
	IsInterConnect bool
}

type DeleteUaslReservationInput struct {
	ID string
}

type FindUaslReservationInput struct {
	ID string
}

type ConfirmUaslReservationInput struct {
	ID             string
	Status         string
	IsInterConnect bool
}

type ListByOperatorInput struct {
	OperatorID string
	Page       int32
}

type ListAdminInput struct {
	Page int32
}

type GetAvailabilitySectionInput struct {
	UaslID        string
	UaslSectionID string
}

type GetAvailabilityInput struct {
	UaslSections   []GetAvailabilitySectionInput
	VehicleIDs     []string
	PortIDs        []string
	IsInterConnect bool
}

type SearchByConditionInput struct {
	RequestIDs []string
}

func (b *CompositeReservationRequestBuilder) ToDomainModels(
	parentInput *RegisterUaslReservationInput,
	childInputs []*RegisterUaslReservationInput,
	domainBuilder func(interface{}) (*model.UaslReservation, error),
) (*model.UaslReservation, []*model.UaslReservation, error) {
	var parentDomain *model.UaslReservation
	var childDomains []*model.UaslReservation

	if parentInput != nil {
		d, err := domainBuilder(parentInput)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build parent uasl reservation domain: %w", err)
		}
		parentDomain = d
	}

	for _, cr := range childInputs {
		d, err := domainBuilder(cr)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build child uasl reservation domain: %w", err)
		}
		childDomains = append(childDomains, d)
	}

	return parentDomain, childDomains, nil
}

func (b *CompositeReservationRequestBuilder) ToConfirmResourcesFromMappings(
	mappings []model.ExternalResourceReservation,
	current *model.UaslReservation,
) ([]model.VehicleReservationRequest, []model.PortReservationRequest, error) {

	vehicleReqs := make([]model.VehicleReservationRequest, 0)
	for _, m := range mappings {
		if m.ResourceType == model.ExternalResourceTypeVehicle {
			startAt := current.StartAt
			endAt := current.EndAt
			if m.StartAt != nil {
				startAt = *m.StartAt
			} else {
				logger.LogInfo("WARN: mapping start_at is nil, falling back to parent reservation start_at",
					"ex_resource_id", m.ExResourceID,
					"resource_type", m.ResourceType,
				)
			}
			if m.EndAt != nil {
				endAt = *m.EndAt
			} else {
				logger.LogInfo("WARN: mapping end_at is nil, falling back to parent reservation end_at",
					"ex_resource_id", m.ExResourceID,
					"resource_type", m.ResourceType,
				)
			}
			vehicleReqs = append(vehicleReqs, model.VehicleReservationRequest{
				VehicleID:          m.ExResourceID,
				StartAt:            startAt,
				EndAt:              endAt,
				OperatorID:         current.ExReservedBy,
				GroupReservationID: current.RequestID.ToString(),
			})
		}
	}

	var aircraftID *string
	if len(vehicleReqs) > 0 {
		firstID := vehicleReqs[0].VehicleID
		aircraftID = &firstID
	}

	var operatorID *value.ModelID
	if current.ExReservedBy != nil {
		operatorID = current.ExReservedBy
	} else {
		e := value.NewEmptyModelID()
		operatorID = &e
	}

	portReqs := make([]model.PortReservationRequest, 0)
	for _, m := range mappings {
		if m.ResourceType == model.ExternalResourceTypePort {
			if m.UsageType == nil {
				return nil, nil, fmt.Errorf("missing usage_type in external resource reservation for port_id=%s", m.ExResourceID)
			}
			usageTypeInt := *m.UsageType
			usageType := util.SafeIntToInt32(usageTypeInt)
			startAt := current.StartAt
			endAt := current.EndAt
			if m.StartAt != nil {
				startAt = *m.StartAt
			} else {
				logger.LogInfo("WARN: mapping start_at is nil, falling back to parent reservation start_at",
					"ex_resource_id", m.ExResourceID,
					"resource_type", m.ResourceType,
				)
			}
			if m.EndAt != nil {
				endAt = *m.EndAt
			} else {
				logger.LogInfo("WARN: mapping end_at is nil, falling back to parent reservation end_at",
					"ex_resource_id", m.ExResourceID,
					"resource_type", m.ResourceType,
				)
			}
			portReqs = append(portReqs, model.PortReservationRequest{
				PortID:              m.ExResourceID,
				AircraftID:          aircraftID,
				GroupReservationID:  current.RequestID.ToString(),
				UsageType:           usageType,
				ReservationTimeFrom: startAt,
				ReservationTimeTo:   endAt,
				OperatorID:          operatorID,
			})
		}
	}

	return vehicleReqs, portReqs, nil
}
