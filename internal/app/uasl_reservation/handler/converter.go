package handler

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	appconv "uasl-reservation/internal/app/uasl_reservation/application/converter"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/pkg/logger"
	internalmqtt "uasl-reservation/internal/pkg/mqtt"
	"uasl-reservation/internal/pkg/util"

	uuid "github.com/satori/go.uuid"
)

const defaultAirspaceID = "a73b0f60-574c-409c-a9ab-3bebeb60dcfa"

type restTryHoldRequest struct {
	RequestID                string            `json:"requestId"`
	OperatorID               string            `json:"operatorId"`
	AcceptedAt               string            `json:"acceptedAt"`
	AirspaceID               string            `json:"airspaceId"`
	ProjectID                string            `json:"projectId"`
	OperationID              string            `json:"operationId"`
	OrganizationID           string            `json:"organizationId"`
	AdministratorID          string            `json:"administratorId"`
	UaslID                   string            `json:"uaslId"`
	OriginAdministratorID    string            `json:"originAdministratorId"`
	OriginUaslID             string            `json:"originUaslId"`
	IgnoreFlightPlanConflict bool              `json:"ignoreFlightPlanConflict"`
	IsInterConnect           bool              `json:"isInterConnect"`
	OperatingAircraft        *restAircraftInfo `json:"operatingAircraft"`
	UaslSections             []restUaslSection `json:"uaslSections"`
	Vehicles                 []restVehicle     `json:"vehicles"`
	Ports                    []restPort        `json:"ports"`
}

type restUaslSection struct {
	UaslID        string `json:"uaslId"`
	UaslSectionID string `json:"uaslSectionId"`
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
	Sequence      int32  `json:"sequence"`
}

type restVehicle struct {
	VehicleID    string            `json:"vehicleId"`
	StartAt      string            `json:"startAt"`
	EndAt        string            `json:"endAt"`
	AircraftInfo *restAircraftInfo `json:"aircraftInfo"`
}

type restAircraftInfo struct {
	AircraftInfoID int32   `json:"aircraftInfoId"`
	RegistrationID string  `json:"registrationId"`
	Maker          string  `json:"maker"`
	ModelNumber    string  `json:"modelNumber"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	Length         float64 `json:"length"`
}

type restPort struct {
	PortID    string `json:"portId"`
	UsageType int32  `json:"usageType"`
	StartAt   string `json:"startAt"`
	EndAt     string `json:"endAt"`
}

type restAvailabilityRequest struct {
	UaslSections   []restAvailabilitySection `json:"uaslSections"`
	RequestIDs     []string                  `json:"requestIds"`
	Vehicles       []restVehicle             `json:"vehicles"`
	Ports          []restPort                `json:"ports"`
	IsInterConnect bool                      `json:"isInterConnect"`
}

type restAvailabilitySection struct {
	UaslID        string `json:"uaslId"`
	UaslSectionID string `json:"uaslSectionId"`
}

type restSearchRequest struct {
	RequestIDs []string `json:"requestIds"`
}

type restNotifyReservationCompletedRequest struct {
	EventID                     string                           `json:"eventId"`
	RequestID                   string                           `json:"requestId"`
	ReservationID               string                           `json:"reservationId"`
	OperatorID                  string                           `json:"operatorId"`
	UaslID                      string                           `json:"uaslId"`
	AdministratorID             string                           `json:"administratorId"`
	FlightPurpose               string                           `json:"flightPurpose"`
	Status                      string                           `json:"status"`
	SubTotalAmount              int32                            `json:"subTotalAmount"`
	ReservedAt                  string                           `json:"reservedAt"`
	EstimatedAt                 string                           `json:"estimatedAt"`
	UpdatedAt                   string                           `json:"updatedAt"`
	UaslSections                []restNotificationUaslSection    `json:"uaslSections"`
	Ports                       []restPortElement                `json:"ports"`
	OperatingAircrafts          []restAircraftInfo               `json:"operatingAircrafts"`
	ConflictedFlightPlanIDs     []string                         `json:"conflictedFlightPlanIds"`
	ConformityAssessmentResults []restConformityAssessmentResult `json:"conformityAssessmentResults"`
}

type restNotificationUaslSection struct {
	UaslSectionID string `json:"uaslSectionId"`
	Sequence      int32  `json:"sequence"`
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
	Amount        int32  `json:"amount"`
}

type restPortElement struct {
	PortID        string `json:"portId"`
	ReservationID string `json:"reservationId"`
	RequestID     string `json:"requestId"`
	UsageType     int32  `json:"usageType"`
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
	Name          string `json:"name"`
}

type restConformityAssessmentResult struct {
	UaslSectionID     string            `json:"uaslSectionId"`
	AircraftInfo      *restAircraftInfo `json:"aircraftInfo"`
	EvaluationResults bool              `json:"evaluationResults"`
	Type              string            `json:"type"`
	Reasons           string            `json:"reasons"`
}

type mqttMessage struct {
	EventID                 string                         `json:"eventId"`
	RequestID               string                         `json:"requestId"`
	OperatorID              string                         `json:"operatorId"`
	FlightPurpose           string                         `json:"flightPurpose,omitempty"`
	Status                  string                         `json:"status"`
	TotalAmount             int32                          `json:"totalAmount,omitempty"`
	ReservedAt              string                         `json:"reservedAt"`
	EstimatedAt             string                         `json:"estimatedAt,omitempty"`
	UpdatedAt               string                         `json:"updatedAt"`
	OriginReservation       *reservationEntity             `json:"originReservation,omitempty"`
	DestinationReservations []destinationReservationEntity `json:"destinationReservations,omitempty"`
}

type reservationEntity struct {
	ReservationID               string                       `json:"reservationId"`
	UaslID                      string                       `json:"uaslId"`
	AdministratorID             string                       `json:"administratorId"`
	SubTotalAmount              int32                        `json:"subTotalAmount,omitempty"`
	UaslSections                []uaslSectionEntity          `json:"uaslSections,omitempty"`
	Ports                       []portElement                `json:"ports,omitempty"`
	Vehicles                    []vehicleElement             `json:"vehicles,omitempty"`
	ConflictedFlightPlanIDs     []string                     `json:"conflictedFlightPlanIds,omitempty"`
	ConformityAssessmentResults []conformityAssessmentResult `json:"conformityAssessmentResults,omitempty"`
}

type destinationReservationEntity struct {
	ReservationID               string                                  `json:"reservationId"`
	UaslID                      string                                  `json:"uaslId"`
	AdministratorID             string                                  `json:"administratorId"`
	SubTotalAmount              int32                                   `json:"subTotalAmount,omitempty"`
	UaslSections                []uaslSectionEntity                     `json:"uaslSections,omitempty"`
	Ports                       []portElement                           `json:"ports,omitempty"`
	ConflictedFlightPlanIDs     []string                                `json:"conflictedFlightPlanIds,omitempty"`
	ConformityAssessmentResults []destinationConformityAssessmentResult `json:"conformityAssessmentResults,omitempty"`
}

type uaslSectionEntity struct {
	UaslSectionID string `json:"uaslSectionId"`
	Sequence      int32  `json:"sequence"`
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
	Amount        int32  `json:"amount,omitempty"`
}

type vehicleElement struct {
	ReservationID string        `json:"reservationId,omitempty"`
	VehicleID     string        `json:"vehicleId,omitempty"`
	AircraftInfo  *aircraftInfo `json:"aircraftInfo,omitempty"`
	StartAt       string        `json:"startAt,omitempty"`
	EndAt         string        `json:"endAt,omitempty"`
	Amount        int32         `json:"amount,omitempty"`
}

type aircraftInfo struct {
	AircraftInfoID int32   `json:"aircraftInfoId,omitempty"`
	RegistrationID string  `json:"registrationId,omitempty"`
	Maker          string  `json:"maker,omitempty"`
	ModelNumber    string  `json:"modelNumber,omitempty"`
	Name           string  `json:"name,omitempty"`
	Type           string  `json:"type,omitempty"`
	Length         float64 `json:"length,omitempty"`
}

type portElement struct {
	PortID        string `json:"portId"`
	ReservationID string `json:"reservationId,omitempty"`
	UsageType     int32  `json:"usageType"`
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
	Name          string `json:"name,omitempty"`
	Amount        int32  `json:"amount,omitempty"`
}

type conformityAssessmentResult struct {
	UaslSectionID     string                  `json:"uaslSectionId"`
	AircraftInfo      *conformityAircraftInfo `json:"aircraftInfo,omitempty"`
	EvaluationResults string                  `json:"evaluationResults"`
	Type              string                  `json:"type"`
	Reasons           string                  `json:"reasons"`
}

type destinationConformityAssessmentResult struct {
	UaslSectionID     string                  `json:"uaslSectionId"`
	AircraftInfo      *conformityAircraftInfo `json:"aircraftInfo,omitempty"`
	EvaluationResults string                  `json:"evaluationResults"`
	Type              string                  `json:"type"`
	Reasons           string                  `json:"reasons"`
}

type conformityAircraftInfo struct {
	Maker       string  `json:"maker,omitempty"`
	ModelNumber string  `json:"modelNumber,omitempty"`
	Name        string  `json:"name,omitempty"`
	Type        string  `json:"type,omitempty"`
	Length      float64 `json:"length,omitempty"`
}

func (r *restTryHoldRequest) toInput() (*appconv.TryHoldCompositeUaslInput, error) {
	if r.OperatorID == "" {
		return nil, fmt.Errorf("operatorId is required")
	}
	if _, err := uuid.FromString(r.OperatorID); err != nil {
		return nil, fmt.Errorf("operatorId must be a valid UUID")
	}
	if len(r.UaslSections) == 0 {
		return nil, fmt.Errorf("uaslSections must be a non-empty array")
	}

	requestID := r.RequestID
	if requestID == "" {
		requestID = uuid.NewV4().String()
	}
	airspaceID := r.AirspaceID
	if airspaceID == "" {
		airspaceID = defaultAirspaceID
	}

	var minStartAt time.Time
	var maxEndAt time.Time
	childReservations := make([]*appconv.RegisterUaslReservationInput, 0, len(r.UaslSections))
	for i, section := range r.UaslSections {
		startAt, err := time.Parse(time.RFC3339, section.StartAt)
		if err != nil {
			return nil, fmt.Errorf("uaslSections[%d].startAt is invalid: %w", i, err)
		}
		endAt, err := time.Parse(time.RFC3339, section.EndAt)
		if err != nil {
			return nil, fmt.Errorf("uaslSections[%d].endAt is invalid: %w", i, err)
		}
		if !endAt.After(startAt) {
			return nil, fmt.Errorf("uaslSections[%d].endAt must be after startAt", i)
		}
		if section.UaslSectionID == "" {
			return nil, fmt.Errorf("uaslSections[%d].uaslSectionId is required", i)
		}
		if _, err := uuid.FromString(section.UaslSectionID); err != nil {
			return nil, fmt.Errorf("uaslSections[%d].uaslSectionId must be a valid UUID", i)
		}
		if i == 0 || startAt.Before(minStartAt) {
			minStartAt = startAt
		}
		if i == 0 || endAt.After(maxEndAt) {
			maxEndAt = endAt
		}

		sequence := section.Sequence
		if sequence == 0 {
			sequence = int32(i + 1)
		}
		acceptedAt := r.AcceptedAt
		if acceptedAt == "" {
			acceptedAt = minStartAt.Format(time.RFC3339)
		}

		childReservations = append(childReservations, &appconv.RegisterUaslReservationInput{
			RequestID:       requestID,
			ExUaslSectionID: section.UaslSectionID,
			ExUaslID:        section.UaslID,
			StartAt:         section.StartAt,
			EndAt:           section.EndAt,
			AcceptedAt:      acceptedAt,
			ExReservedBy:    r.OperatorID,
			AirspaceID:      airspaceID,
			ProjectID:       r.ProjectID,
			OperationID:     r.OperationID,
			OrganizationID:  r.OrganizationID,
			Status:          "INHERITED",
			Sequence:        sequence,
		})
	}

	acceptedAt := r.AcceptedAt
	if acceptedAt == "" {
		acceptedAt = minStartAt.Format(time.RFC3339)
	}

	parentUaslID := r.UaslID
	if r.OriginUaslID != "" {
		parentUaslID = r.OriginUaslID
	}
	parentAdminID := r.AdministratorID
	if r.OriginAdministratorID != "" {
		parentAdminID = r.OriginAdministratorID
	}

	parentReservation := &appconv.RegisterUaslReservationInput{
		RequestID:         requestID,
		ExUaslID:          parentUaslID,
		ExAdministratorID: parentAdminID,
		StartAt:           minStartAt.Format(time.RFC3339),
		EndAt:             maxEndAt.Format(time.RFC3339),
		AcceptedAt:        acceptedAt,
		ExReservedBy:      r.OperatorID,
		AirspaceID:        airspaceID,
		ProjectID:         r.ProjectID,
		OperationID:       r.OperationID,
		OrganizationID:    r.OrganizationID,
		Status:            "PENDING",
	}

	vehicles := make([]appconv.TryHoldVehicleInput, 0, len(r.Vehicles))
	var operatingAircraft *model.VehicleDetailInfo
	for _, vehicle := range r.Vehicles {
		if operatingAircraft == nil && vehicle.AircraftInfo != nil {
			operatingAircraft = vehicle.AircraftInfo.toModel()
		}
		if vehicle.VehicleID == "" {
			continue
		}
		startAt := vehicle.StartAt
		if startAt == "" {
			startAt = parentReservation.StartAt
		}
		endAt := vehicle.EndAt
		if endAt == "" {
			endAt = parentReservation.EndAt
		}
		vehicles = append(vehicles, appconv.TryHoldVehicleInput{
			VehicleID:    vehicle.VehicleID,
			StartAt:      startAt,
			EndAt:        endAt,
			AircraftInfo: vehicle.AircraftInfo.toModel(),
		})
	}
	if r.OperatingAircraft != nil {
		operatingAircraft = r.OperatingAircraft.toModel()
	}

	ports := make([]appconv.TryHoldPortInput, 0, len(r.Ports))
	for _, port := range r.Ports {
		ports = append(ports, appconv.TryHoldPortInput{
			PortID:    port.PortID,
			UsageType: port.UsageType,
			StartAt:   port.StartAt,
			EndAt:     port.EndAt,
		})
	}

	return &appconv.TryHoldCompositeUaslInput{
		Vehicles:                 vehicles,
		Ports:                    ports,
		ParentUaslReservation:    parentReservation,
		ChildUaslReservations:    childReservations,
		IgnoreFlightPlanConflict: r.IgnoreFlightPlanConflict,
		IsInterConnect:           r.IsInterConnect,
		OperatingAircraft:        operatingAircraft,
	}, nil
}

func (a *restAircraftInfo) toModel() *model.VehicleDetailInfo {
	if a == nil {
		return nil
	}
	return &model.VehicleDetailInfo{
		AircraftInfoID: a.AircraftInfoID,
		RegistrationID: a.RegistrationID,
		Maker:          a.Maker,
		ModelNumber:    a.ModelNumber,
		Name:           a.Name,
		Type:           a.Type,
		Length:         strconv.FormatFloat(a.Length, 'f', -1, 64),
	}
}

func (r *restAvailabilityRequest) toGetAvailabilityInput() (*appconv.GetAvailabilityInput, error) {
	if len(r.UaslSections) == 0 && len(r.RequestIDs) == 0 {
		return nil, fmt.Errorf("either uaslSections or requestIds must be provided")
	}
	if len(r.UaslSections) > 0 && len(r.RequestIDs) > 0 {
		return nil, fmt.Errorf("only one of uaslSections or requestIds can be provided")
	}
	sections := make([]appconv.GetAvailabilitySectionInput, 0, len(r.UaslSections))
	for i, section := range r.UaslSections {
		if section.UaslSectionID == "" {
			return nil, fmt.Errorf("uaslSections[%d].uaslSectionId is required", i)
		}
		sections = append(sections, appconv.GetAvailabilitySectionInput{
			UaslID:        section.UaslID,
			UaslSectionID: section.UaslSectionID,
		})
	}
	vehicleIDs := make([]string, 0, len(r.Vehicles))
	for _, vehicle := range r.Vehicles {
		if vehicle.VehicleID != "" {
			vehicleIDs = append(vehicleIDs, vehicle.VehicleID)
		}
	}
	portIDs := make([]string, 0, len(r.Ports))
	for _, port := range r.Ports {
		if port.PortID != "" {
			portIDs = append(portIDs, port.PortID)
		}
	}
	return &appconv.GetAvailabilityInput{
		UaslSections:   sections,
		VehicleIDs:     vehicleIDs,
		PortIDs:        portIDs,
		IsInterConnect: r.IsInterConnect,
	}, nil
}

func (r *restAvailabilityRequest) toSearchInput() (*appconv.SearchByConditionInput, error) {
	if len(r.RequestIDs) == 0 {
		return nil, fmt.Errorf("requestIds must not be empty")
	}
	if len(r.UaslSections) > 0 {
		return nil, fmt.Errorf("only one of uaslSections or requestIds can be provided")
	}
	return &appconv.SearchByConditionInput{
		RequestIDs: append([]string{}, r.RequestIDs...),
	}, nil
}

func (r *restSearchRequest) toInput() (*appconv.SearchByConditionInput, error) {
	if len(r.RequestIDs) == 0 {
		return nil, fmt.Errorf("requestIds must not be empty")
	}
	return &appconv.SearchByConditionInput{RequestIDs: r.RequestIDs}, nil
}

type restEstimateRequest struct {
	UaslSections   []restEstimateSection `json:"uaslSections"`
	Vehicles       []restVehicle         `json:"vehicles"`
	Ports          []restPort            `json:"ports"`
	IsInterConnect bool                  `json:"isInterConnect"`
}

type restEstimateSection struct {
	UaslID        string `json:"uaslId"`
	UaslSectionID string `json:"uaslSectionId"`
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
}

func (r *restEstimateRequest) toInput() (*appconv.EstimateUaslReservationInput, error) {
	if len(r.UaslSections) == 0 {
		return nil, fmt.Errorf("uaslSections must be a non-empty array")
	}
	sections := make([]appconv.EstimateUaslSectionInput, 0, len(r.UaslSections))
	for i, section := range r.UaslSections {
		if section.UaslSectionID == "" || section.StartAt == "" || section.EndAt == "" {
			return nil, fmt.Errorf("uaslSections[%d].uaslSectionId, startAt and endAt are required", i)
		}
		sections = append(sections, appconv.EstimateUaslSectionInput{
			ExUaslID:        section.UaslID,
			ExUaslSectionID: section.UaslSectionID,
			StartAt:         section.StartAt,
			EndAt:           section.EndAt,
		})
	}
	vehicles := make([]appconv.EstimateVehicleInput, 0, len(r.Vehicles))
	for _, vehicle := range r.Vehicles {
		vehicles = append(vehicles, appconv.EstimateVehicleInput{
			VehicleID: vehicle.VehicleID,
			StartAt:   vehicle.StartAt,
			EndAt:     vehicle.EndAt,
		})
	}
	ports := make([]appconv.EstimatePortInput, 0, len(r.Ports))
	for _, port := range r.Ports {
		ports = append(ports, appconv.EstimatePortInput{
			PortID:  port.PortID,
			StartAt: port.StartAt,
			EndAt:   port.EndAt,
		})
	}
	return &appconv.EstimateUaslReservationInput{
		UaslSections:   sections,
		Vehicles:       vehicles,
		Ports:          ports,
		IsInterConnect: r.IsInterConnect,
	}, nil
}

func (r *restNotifyReservationCompletedRequest) toModelReq() *model.NotifyReservationCompletedRequest {
	sections := make([]model.UaslSectionReservationElement, 0, len(r.UaslSections))
	for _, s := range r.UaslSections {
		sections = append(sections, model.UaslSectionReservationElement{
			UaslSectionID: s.UaslSectionID,
			Sequence:      int(s.Sequence),
			StartAt:       s.StartAt,
			EndAt:         s.EndAt,
			Amount:        int(s.Amount),
		})
	}
	ports := make([]model.PortReservationElement, 0, len(r.Ports))
	for _, p := range r.Ports {
		ports = append(ports, model.PortReservationElement{
			PortID:        p.PortID,
			ReservationID: p.ReservationID,
			StartAt:       p.StartAt,
			EndAt:         p.EndAt,
			UsageType:     int(p.UsageType),
			Name:          p.Name,
		})
	}
	aircrafts := make([]model.ReservationAircraftInfo, 0, len(r.OperatingAircrafts))
	for _, a := range r.OperatingAircrafts {
		aircrafts = append(aircrafts, model.ReservationAircraftInfo{
			AircraftInfoID: a.AircraftInfoID,
			RegistrationID: a.RegistrationID,
			Maker:          a.Maker,
			ModelNumber:    a.ModelNumber,
			Name:           a.Name,
			Type:           a.Type,
			Length:         a.Length,
		})
	}
	conformities := make([]model.ConformityAssessmentResult, 0, len(r.ConformityAssessmentResults))
	for _, c := range r.ConformityAssessmentResults {
		var ai *model.ReservationAircraftInfo
		if c.AircraftInfo != nil {
			ai = &model.ReservationAircraftInfo{
				AircraftInfoID: c.AircraftInfo.AircraftInfoID,
				RegistrationID: c.AircraftInfo.RegistrationID,
				Maker:          c.AircraftInfo.Maker,
				ModelNumber:    c.AircraftInfo.ModelNumber,
				Name:           c.AircraftInfo.Name,
				Type:           c.AircraftInfo.Type,
				Length:         c.AircraftInfo.Length,
			}
		}
		evalResult := "false"
		if c.EvaluationResults {
			evalResult = "true"
		}
		conformities = append(conformities, model.ConformityAssessmentResult{
			UaslSectionID:     c.UaslSectionID,
			AircraftInfo:      ai,
			EvaluationResults: evalResult,
			Type:              c.Type,
			Reasons:           c.Reasons,
		})
	}
	notification := &model.DestinationReservationNotification{
		RequestId:                   r.RequestID,
		ReservationId:               r.ReservationID,
		OperatorId:                  r.OperatorID,
		UaslId:                      r.UaslID,
		AdministratorId:             r.AdministratorID,
		FlightPurpose:               r.FlightPurpose,
		Status:                      r.Status,
		SubTotalAmount:              r.SubTotalAmount,
		ReservedAt:                  r.ReservedAt,
		EstimatedAt:                 r.EstimatedAt,
		UpdatedAt:                   r.UpdatedAt,
		ConflictedFlightPlanIds:     r.ConflictedFlightPlanIDs,
		UaslSections:                sections,
		Ports:                       ports,
		OperatingAircrafts:          aircrafts,
		ConformityAssessmentResults: conformities,
	}
	return &model.NotifyReservationCompletedRequest{
		Notifications: []*model.DestinationReservationNotification{notification},
	}
}

func buildAvailabilityResponse(res *model.GetAvailabilityResponse) map[string]any {
	result := map[string]any{}
	if len(res.UaslSections) > 0 {
		sections := make([]map[string]any, 0, len(res.UaslSections))
		for _, section := range res.UaslSections {
			sections = append(sections, map[string]any{
				"requestId":     section.RequestID,
				"operatorId":    section.OperatorID,
				"flightPurpose": section.FlightPurpose,
				"startAt":       section.StartAt,
				"endAt":         section.EndAt,
			})
		}
		result["uaslSections"] = sections
	}
	if len(res.Vehicles) > 0 {
		vehicles := make([]map[string]any, 0, len(res.Vehicles))
		for _, vehicle := range res.Vehicles {
			vehicles = append(vehicles, map[string]any{
				"requestId":     vehicle.RequestID,
				"reservationId": vehicle.ReservationID,
				"vehicleId":     vehicle.VehicleID,
				"name":          vehicle.Name,
				"startAt":       vehicle.StartAt,
				"endAt":         vehicle.EndAt,
			})
		}
		result["vehicles"] = vehicles
	}
	if len(res.Ports) > 0 {
		ports := make([]map[string]any, 0, len(res.Ports))
		for _, port := range res.Ports {
			ports = append(ports, map[string]any{
				"requestId":     port.RequestID,
				"reservationId": port.ReservationID,
				"portId":        port.PortID,
				"name":          port.Name,
				"startAt":       port.StartAt,
				"endAt":         port.EndAt,
			})
		}
		result["ports"] = ports
	}
	return map[string]any{"result": result}
}

func buildListItemResponse(item *model.UaslReservationListItemMsg) map[string]any {
	if item == nil || item.ParentUaslReservation == nil {
		return nil
	}
	return buildReserveCompositeResponse(
		item.ParentUaslReservation,
		item.ChildUaslReservations,
		item.Vehicles,
		item.Ports,
		nil,
		false,
		false,
	)
}

func buildListResponse(res *model.ListUaslReservationsResponse) map[string]any {
	items := make([]map[string]any, 0, len(res.Result))
	for _, item := range res.Result {
		if mapped := buildListItemResponse(item); mapped != nil {
			items = append(items, mapped)
		}
	}
	pageInfo := map[string]any{
		"currentPage": int32(1),
		"lastPage":    int32(0),
		"perPage":     int32(20),
		"total":       int32(0),
	}
	if info := res.PageInfo; info != nil {
		pageInfo["currentPage"] = info.CurrentPage
		pageInfo["lastPage"] = info.LastPage
		pageInfo["perPage"] = info.PerPage
		pageInfo["total"] = info.Total
	}
	return map[string]any{
		"result":      items,
		"currentPage": pageInfo["currentPage"],
		"lastPage":    pageInfo["lastPage"],
		"perPage":     pageInfo["perPage"],
		"total":       pageInfo["total"],
	}
}

func buildSearchResponse(res *model.SearchByConditionResponse) map[string]any {
	items := make([]map[string]any, 0, len(res.Result))
	for _, item := range res.Result {
		if mapped := buildListItemResponse(item); mapped != nil {
			items = append(items, mapped)
		}
	}
	return map[string]any{"result": items}
}

func buildDeleteResponse(res *model.DeleteUaslReservationResponse, requestID string) map[string]any {
	responseID := requestID
	if res != nil && res.Data != nil && res.Data.ID != "" {
		responseID = res.Data.ID
	}
	return map[string]any{"requestId": responseID}
}

func buildNotifyReservationCompletedResponse(req *restNotifyReservationCompletedRequest) map[string]any {
	return map[string]any{
		"message":   "accepted",
		"requestId": req.RequestID,
	}
}

func buildReserveCompositeResponse(
	parent *model.UaslReservationMessage,
	children []*model.UaslReservationMessage,
	vehicles []*model.VehicleElement,
	ports []*model.PortElement,
	conflictedFlightPlanIDs []string,
	isInterConnect bool,
	isTryHold bool,
) map[string]any {
	if parent == nil {
		return map[string]any{
			"conflictedFlightPlanIds": conflictedFlightPlanIDs,
			"originReservation":       nil,
			"destinationReservations": []map[string]any{},
			"totalAmount":             0,
		}
	}

	fixedAt := ""
	if parent.FixedAt != nil {
		fixedAt = parent.FixedAt.Format(time.RFC3339)
	}
	acceptedAt := ""
	if parent.AcceptedAt != nil {
		acceptedAt = parent.AcceptedAt.Format(time.RFC3339)
	}
	estimatedAt := ""
	if parent.EstimatedAt != nil {
		estimatedAt = parent.EstimatedAt.Format(time.RFC3339)
	}
	response := map[string]any{
		"requestId":   parent.RequestID,
		"operatorId":  parent.ExReservedBy,
		"status":      parent.Status,
		"updatedAt":   parent.UpdatedAt.Format(time.RFC3339),
		"estimatedAt": estimatedAt,
		"reservedAt":  fixedAt,
	}
	if response["reservedAt"] == "" && acceptedAt != "" {
		response["reservedAt"] = acceptedAt
	}
	if parent.FlightPurpose != "" {
		response["flightPurpose"] = parent.FlightPurpose
	}
	if isInterConnect && parent.PricingRuleVersion > 0 {
		response["pricingRuleVersion"] = parent.PricingRuleVersion
	}

	destinationReservationLookup := buildDestinationReservationLookup(parent, children)

	type group struct {
		UaslID          string
		AdministratorID string
		Sections        []map[string]any
		Conformities    []map[string]any
	}

	orderedChildren := make([]*model.UaslReservationMessage, 0, len(children))
	orderedChildren = append(orderedChildren, children...)
	sort.SliceStable(orderedChildren, func(i, j int) bool {
		if orderedChildren[i].Sequence != orderedChildren[j].Sequence {
			return orderedChildren[i].Sequence < orderedChildren[j].Sequence
		}
		return orderedChildren[i].StartAt.Before(orderedChildren[j].StartAt)
	})

	originUaslID := parent.ExUaslID
	originAdministratorID := parent.ExAdministratorID

	var groups []group
	for _, child := range orderedChildren {
		section := map[string]any{
			"uaslSectionId": child.ExUaslSectionID,
			"sequence":      child.Sequence,
			"startAt":       child.StartAt.Format(time.RFC3339),
			"endAt":         child.EndAt.Format(time.RFC3339),
		}
		if child.Amount > 0 {
			section["amount"] = child.Amount
		}
		if len(groups) == 0 || groups[len(groups)-1].UaslID != child.ExUaslID {
			groups = append(groups, group{
				UaslID:          child.ExUaslID,
				AdministratorID: child.ExAdministratorID,
			})
		}
		last := &groups[len(groups)-1]
		last.Sections = append(last.Sections, section)
		last.Conformities = append(last.Conformities, buildConformityAssessmentArray(child.ConformityAssessmentResults)...)
	}

	if originUaslID == "" && len(groups) > 0 {
		originUaslID = groups[0].UaslID
	}
	if originAdministratorID == "" && len(groups) > 0 {
		originAdministratorID = groups[0].AdministratorID
	}

	portsByGroup := make(map[int][]map[string]any, len(groups))
	lastGroup := len(groups) - 1
	firstByAdmin := map[string]int{}
	lastByAdmin := map[string]int{}
	for i, g := range groups {
		if _, ok := firstByAdmin[g.AdministratorID]; !ok {
			firstByAdmin[g.AdministratorID] = i
		}
		lastByAdmin[g.AdministratorID] = i
	}
	for _, port := range ports {
		mapped := buildPortMap(port, isTryHold)
		target := 0
		if len(groups) > 0 {
			adminID := port.ExAdministratorID
			if adminID != "" {
				if port.UsageType == 2 {
					target = lastByAdmin[adminID]
				} else {
					target = firstByAdmin[adminID]
				}
			} else if port.UsageType == 2 {
				target = lastGroup
			}
		}
		portsByGroup[target] = append(portsByGroup[target], mapped)
	}

	originVehicles := make([]map[string]any, 0, len(vehicles))
	for _, vehicle := range vehicles {
		originVehicles = append(originVehicles, buildVehicleMap(vehicle, isTryHold))
	}

	destinations := make([]map[string]any, 0)
	var origin map[string]any
	for i, g := range groups {
		groupPorts := portsByGroup[i]
		if groupPorts == nil {
			groupPorts = []map[string]any{}
		}
		groupConformities := g.Conformities
		if groupConformities == nil {
			groupConformities = []map[string]any{}
		}
		resolvedAdminID := g.AdministratorID
		if i == 0 && resolvedAdminID == "" {
			resolvedAdminID = originAdministratorID
		}
		entity := map[string]any{
			"uaslId":                      g.UaslID,
			"administratorId":             resolvedAdminID,
			"uaslSections":                g.Sections,
			"ports":                       groupPorts,
			"conformityAssessmentResults": groupConformities,
			"conflictedFlightPlanIds":     []string{},
		}
		if i == 0 && !isInterConnect {
			entity["reservationId"] = parent.ID
			entity["vehicles"] = originVehicles
			entity["conflictedFlightPlanIds"] = conflictedFlightPlanIDs
			entity["subTotalAmount"] = sumAmounts(g.Sections) + sumAmounts(originVehicles) + sumPortAmountsUnique(groupPorts)
			origin = entity
			continue
		}
		if reservationID := destinationReservationLookup[g.UaslID]; reservationID != "" {
			entity["reservationId"] = reservationID
		} else if parent.ID != "" {
			entity["reservationId"] = parent.ID
		}
		entity["subTotalAmount"] = sumAmounts(g.Sections) + sumPortAmountsUnique(groupPorts)
		destinations = append(destinations, entity)
	}

	if origin == nil && !isInterConnect {
		allPorts := make([]map[string]any, 0)
		for _, ps := range portsByGroup {
			allPorts = append(allPorts, ps...)
		}
		origin = map[string]any{
			"reservationId":               parent.ID,
			"uaslId":                      originUaslID,
			"administratorId":             originAdministratorID,
			"uaslSections":                []map[string]any{},
			"vehicles":                    originVehicles,
			"ports":                       allPorts,
			"conformityAssessmentResults": []map[string]any{},
			"conflictedFlightPlanIds":     conflictedFlightPlanIDs,
			"subTotalAmount":              sumAmounts(originVehicles),
		}
	}

	if origin != nil && !isInterConnect {
		response["originReservation"] = origin
	}
	response["destinationReservations"] = destinations
	response["totalAmount"] = calculateUnionTotalAmount(origin, destinations)
	return response
}

func buildVehicleMap(v *model.VehicleElement, isTryHold bool) map[string]any {
	result := map[string]any{}
	hasReservedVehicle := v.VehicleID != ""
	if hasReservedVehicle {
		result["vehicleId"] = v.VehicleID
	}
	if hasReservedVehicle {
		if !isTryHold && v.ReservationID != "" {
			result["reservationId"] = v.ReservationID
		}
		if v.StartAt != "" {
			result["startAt"] = v.StartAt
		}
		if v.EndAt != "" {
			result["endAt"] = v.EndAt
		}
		if v.Amount > 0 {
			result["amount"] = v.Amount
		}
	}
	if ai := v.AircraftInfo; ai != nil {
		result["aircraftInfo"] = map[string]any{
			"aircraftInfoId": ai.AircraftInfoID,
			"registrationId": ai.RegistrationID,
			"maker":          ai.Maker,
			"modelNumber":    ai.ModelNumber,
			"name":           ai.Name,
			"type":           ai.Type,
			"length":         ai.Length,
		}
	}
	return result
}

func buildPortMap(p *model.PortElement, isTryHold bool) map[string]any {
	result := map[string]any{
		"portId":    p.PortID,
		"usageType": p.UsageType,
		"startAt":   p.StartAt,
		"endAt":     p.EndAt,
	}
	if !isTryHold && p.ReservationID != "" {
		result["reservationId"] = p.ReservationID
	}
	if p.Name != "" {
		result["name"] = p.Name
	}
	if p.Amount > 0 {
		result["amount"] = p.Amount
	}
	return result
}

func buildConformityAssessmentArray(items []*model.ConformityAssessmentResultMsg) []map[string]any {
	results := make([]map[string]any, 0, len(items))
	for _, item := range items {
		assessmentType := item.Type
		if assessmentType == "" {
			assessmentType = "null"
		}
		entry := map[string]any{
			"uaslSectionId":     item.UaslSectionID,
			"evaluationResults": item.EvaluationResults,
			"type":              assessmentType,
			"reasons":           item.Reasons,
		}
		if ai := item.AircraftInfo; ai != nil {
			entry["aircraftInfo"] = map[string]any{
				"aircraftInfoId": ai.AircraftInfoID,
				"registrationId": ai.RegistrationID,
				"maker":          ai.Maker,
				"modelNumber":    ai.ModelNumber,
				"name":           ai.Name,
				"type":           ai.Type,
				"length":         ai.Length,
			}
		}
		results = append(results, entry)
	}
	return results
}

func buildConflictResponse(
	requestID string,
	conflictType string,
	conflictedResourceIDs []string,
	conflictedFlightPlanIDs []string,
	administratorID string,
	reservationID string,
	overrideMessage string,
) map[string]any {
	errorCode, message, normalizedType := mapConflictMeta(conflictType)
	if overrideMessage != "" {
		message = overrideMessage
	}
	if conflictedResourceIDs == nil {
		conflictedResourceIDs = []string{}
	}
	if conflictedFlightPlanIDs == nil {
		conflictedFlightPlanIDs = []string{}
	}

	return map[string]any{
		"requestId": requestID,
		"errorCode": errorCode,
		"message":   message,
		"conflicts": map[string]any{
			"type":  normalizedType,
			"items": buildConflictItems(requestID, normalizedType, conflictedResourceIDs, conflictedFlightPlanIDs, administratorID, reservationID),
		},
	}
}

func mapConflictMeta(conflictType string) (string, string, string) {
	switch strings.ToUpper(conflictType) {
	case "FLIGHT_PLAN", "FLIGHTPLAN":
		return "AIRSPACE_CONFLICT", "Reservation not executed due to airspace conflict", "FLIGHTPLAN"
	case "UASL":
		return "UASL_CONFLICT", "uasl reservation conflict detected", "UASL"
	case "VEHICLE":
		return "VEHICLE_CONFLICT", "vehicle reservation conflict detected", "VEHICLE"
	case "PORT":
		return "PORT_CONFLICT", "port reservation conflict detected", "PORT"
	default:
		return "CONFLICT", "Reservation not executed due to conflict", strings.ToUpper(conflictType)
	}
}

func buildConflictItems(
	requestID string,
	conflictType string,
	conflictedResourceIDs []string,
	conflictedFlightPlanIDs []string,
	administratorID string,
	reservationID string,
) []map[string]any {
	if conflictType == "FLIGHTPLAN" {
		return []map[string]any{{
			"requestId":       requestID,
			"reservationId":   reservationID,
			"administratorId": administratorID,
			"airspaceId":      "",
			"resourceId":      "",
			"flightPlanIds":   conflictedFlightPlanIDs,
		}}
	}

	items := make([]map[string]any, 0, len(conflictedResourceIDs))
	for _, id := range conflictedResourceIDs {
		item := map[string]any{
			"requestId":       requestID,
			"reservationId":   reservationID,
			"administratorId": administratorID,
			"resourceId":      id,
			"flightPlanIds":   []string{},
		}
		switch conflictType {
		case "UASL":
			item["uaslId"] = id
		case "VEHICLE":
			item["vehicleId"] = id
		case "PORT":
			item["portId"] = id
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		items = append(items, map[string]any{
			"requestId":       requestID,
			"reservationId":   reservationID,
			"administratorId": administratorID,
			"resourceId":      "",
			"flightPlanIds":   []string{},
		})
	}
	return items
}

func sumAmounts(items []map[string]any) int {
	total := 0
	for _, item := range items {
		switch v := item["amount"].(type) {
		case int:
			total += v
		case int32:
			total += int(v)
		case int64:
			total += int(v)
		case float64:
			total += int(v)
		}
	}
	return total
}

func sumPortAmountsUnique(ports []map[string]any) int {
	type portKey struct {
		portID  string
		startAt string
		endAt   string
	}
	seen := make(map[portKey]bool)
	total := 0
	for _, p := range ports {
		key := portKey{
			portID:  stringValue(p["portId"]),
			startAt: stringValue(p["startAt"]),
			endAt:   stringValue(p["endAt"]),
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		total += extractIntAmount(p["amount"])
	}
	return total
}

type resourceInterval struct {
	startAt time.Time
	endAt   time.Time
	amount  int
}

func calculateUnionTotalAmount(origin map[string]any, destinations []map[string]any) int {
	allGroups := map[string][]resourceInterval{}
	mergeInto := func(groups map[string][]resourceInterval) {
		for id, intervals := range groups {
			allGroups[id] = append(allGroups[id], intervals...)
		}
	}
	mergeInto(collectEntityIntervals(origin))
	for _, destination := range destinations {
		mergeInto(collectEntityIntervals(destination))
	}

	total := 0
	for _, intervals := range allGroups {
		total += calcUnionAmountForGroup(intervals)
	}
	return total
}

func collectEntityIntervals(entity map[string]any) map[string][]resourceInterval {
	groups := map[string][]resourceInterval{}
	if entity == nil {
		return groups
	}
	addInterval := func(prefix, id, startAt, endAt string, amount int) {
		if id == "" || amount <= 0 {
			return
		}
		start, ok1 := parseResourceTime(startAt)
		end, ok2 := parseResourceTime(endAt)
		if !ok1 || !ok2 || !end.After(start) {
			return
		}
		groups[prefix+id] = append(groups[prefix+id], resourceInterval{startAt: start, endAt: end, amount: amount})
	}

	for _, section := range getMapSlice(entity["uaslSections"]) {
		addInterval("section:", stringValue(section["uaslSectionId"]), stringValue(section["startAt"]), stringValue(section["endAt"]), extractIntAmount(section["amount"]))
	}
	for _, vehicle := range getMapSlice(entity["vehicles"]) {
		addInterval("vehicle:", stringValue(vehicle["vehicleId"]), stringValue(vehicle["startAt"]), stringValue(vehicle["endAt"]), extractIntAmount(vehicle["amount"]))
	}
	for _, port := range getMapSlice(entity["ports"]) {
		addInterval("port:", stringValue(port["portId"]), stringValue(port["startAt"]), stringValue(port["endAt"]), extractIntAmount(port["amount"]))
	}
	return groups
}

func calcUnionAmountForGroup(intervals []resourceInterval) int {
	if len(intervals) == 0 {
		return 0
	}
	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i].startAt.Before(intervals[j].startAt)
	})

	total := 0
	var coveredUntil time.Time
	for _, interval := range intervals {
		durationMins := interval.endAt.Sub(interval.startAt).Minutes()
		if durationMins <= 0 {
			continue
		}
		rate := float64(interval.amount) / durationMins
		if coveredUntil.IsZero() || !interval.startAt.Before(coveredUntil) {
			total += interval.amount
			coveredUntil = interval.endAt
			continue
		}
		if interval.endAt.After(coveredUntil) {
			total += int(rate * interval.endAt.Sub(coveredUntil).Minutes())
			coveredUntil = interval.endAt
		}
	}
	return total
}

func parseResourceTime(s string) (time.Time, bool) {
	for _, format := range []string{time.RFC3339, "2006-01-02T15:04:05.000Z", "2006-01-02T15:04:05Z07:00"} {
		if t, err := time.Parse(format, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func getMapSlice(raw any) []map[string]any {
	items, ok := raw.([]map[string]any)
	if ok {
		return items
	}
	interfaces, ok := raw.([]any)
	if !ok {
		return nil
	}
	result := make([]map[string]any, 0, len(interfaces))
	for _, item := range interfaces {
		if mapped, ok := item.(map[string]any); ok {
			result = append(result, mapped)
		}
	}
	return result
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func extractIntAmount(v any) int {
	switch value := v.(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func publishMQTTMessage(response map[string]any) error {
	requestID := stringValue(response["requestId"])
	operatorID := stringValue(response["operatorId"])
	if requestID == "" || operatorID == "" {
		return fmt.Errorf("missing requestId or operatorId in response")
	}

	payload := mqttMessage{
		EventID:       uuid.NewV4().String(),
		RequestID:     requestID,
		OperatorID:    operatorID,
		FlightPurpose: stringValue(response["flightPurpose"]),
		Status:        stringValue(response["status"]),
		ReservedAt:    stringValue(response["reservedAt"]),
		EstimatedAt:   stringValue(response["estimatedAt"]),
		UpdatedAt:     stringValue(response["updatedAt"]),
		TotalAmount:   int32(extractIntAmount(response["totalAmount"])),
	}
	if payload.UpdatedAt == "" {
		payload.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	if origin, ok := response["originReservation"].(map[string]any); ok {
		payload.OriginReservation = buildReservationEntity(origin)
	}
	for _, item := range getMapSlice(response["destinationReservations"]) {
		if entity := buildDestinationReservationEntity(item); entity != nil {
			payload.DestinationReservations = append(payload.DestinationReservations, *entity)
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal mqtt payload: %w", err)
	}
	client, err := internalmqtt.NewMQTTClient()
	if err != nil {
		return err
	}
	topic := fmt.Sprintf("uasl/operator/%s/uaslReservation/%s", operatorID, requestID)
	return client.Publish(topic, 1, false, body)
}

func buildNotifyReservationCompletedRequest(res *model.ReserveCompositeUaslResponse) *model.NotifyReservationCompletedRequest {
	if res == nil || res.ParentUaslReservation == nil {
		return nil
	}
	parent := res.ParentUaslReservation
	destinations := parent.DestinationReservations
	if len(destinations) == 0 {
		destinations = buildDestinationsFromChildren(res.ChildUaslReservations, parent.ExAdministratorID)
		logger.LogInfo("NotifyReservationCompleted: destinations rebuilt from children, count=%d", len(destinations))
	}
	if len(destinations) == 0 {
		return nil
	}

	requestID := parent.RequestID
	operatorID := parent.ExReservedBy
	status := parent.Status
	flightPurpose := parent.FlightPurpose
	estimatedAt := ""
	if parent.EstimatedAt != nil {
		estimatedAt = parent.EstimatedAt.Format(time.RFC3339)
	}
	updatedAt := parent.UpdatedAt.Format(time.RFC3339)
	reservedAt := ""
	if parent.FixedAt != nil {
		reservedAt = parent.FixedAt.Format(time.RFC3339)
	} else if parent.AcceptedAt != nil {
		reservedAt = parent.AcceptedAt.Format(time.RFC3339)
	}

	children := res.ChildUaslReservations
	ports := res.Ports

	conformities := make([]*model.ConformityAssessmentResultMsg, 0)
	for _, child := range children {
		if child == nil {
			continue
		}
		conformities = append(conformities, child.ConformityAssessmentResults...)
	}

	lastSeqUaslID := getLastSequenceUaslID(children)
	operatingAircrafts := buildOperatingAircraftsFromConformity(conformities)
	sectionsByUasl, sectionIDsByUasl, amountByUasl := groupSectionsByUasl(children)

	notifications := make([]*model.DestinationReservationNotification, 0, len(destinations))
	for _, dest := range destinations {
		if dest.ReservationID == "" {
			continue
		}
		if parent.ExAdministratorID != "" && dest.ExAdministratorID == parent.ExAdministratorID {
			continue
		}

		destUaslID := dest.ExUaslID
		destSections := sectionsByUasl[destUaslID]
		destSectionIDs := sectionIDsByUasl[destUaslID]
		destPorts, portAmount := collectDestinationPorts(ports, destUaslID, lastSeqUaslID)
		destConformity := filterConformityBySections(conformities, destSectionIDs, buildSectionRangesFromNotifications(destSections))

		uaslSections := make([]model.UaslSectionReservationElement, 0, len(destSections))
		for _, s := range destSections {
			if s != nil {
				uaslSections = append(uaslSections, *s)
			}
		}

		portElems := make([]model.PortReservationElement, 0, len(destPorts))
		for _, p := range destPorts {
			if p != nil {
				portElems = append(portElems, model.PortReservationElement{
					PortID:            p.PortID,
					ReservationID:     p.ReservationID,
					Name:              p.Name,
					UsageType:         p.UsageType,
					StartAt:           p.StartAt,
					EndAt:             p.EndAt,
					Amount:            p.Amount,
					ExAdministratorID: p.ExAdministratorID,
				})
			}
		}

		conformityResults := make([]model.ConformityAssessmentResult, 0, len(destConformity))
		for _, c := range destConformity {
			if c == nil {
				continue
			}
			cr := model.ConformityAssessmentResult{
				UaslSectionID:     c.UaslSectionID,
				EvaluationResults: fmt.Sprint(c.EvaluationResults),
				Type:              c.Type,
				Reasons:           c.Reasons,
			}
			if c.AircraftInfo != nil {
				cr.AircraftInfo = &model.ReservationAircraftInfo{
					AircraftInfoID: c.AircraftInfo.AircraftInfoID,
					RegistrationID: c.AircraftInfo.RegistrationID,
					Maker:          c.AircraftInfo.Maker,
					ModelNumber:    c.AircraftInfo.ModelNumber,
					Name:           c.AircraftInfo.Name,
					Type:           c.AircraftInfo.Type,
					Length:         c.AircraftInfo.Length,
				}
			}
			conformityResults = append(conformityResults, cr)
		}

		opAircrafts := make([]model.ReservationAircraftInfo, 0, len(operatingAircrafts))
		for _, ai := range operatingAircrafts {
			if ai != nil {
				opAircrafts = append(opAircrafts, *ai)
			}
		}

		notifications = append(notifications, &model.DestinationReservationNotification{
			RequestId:                   requestID,
			ReservationId:               dest.ReservationID,
			OperatorId:                  operatorID,
			UaslId:                      destUaslID,
			AdministratorId:             dest.ExAdministratorID,
			FlightPurpose:               flightPurpose,
			Status:                      status,
			SubTotalAmount:              amountByUasl[destUaslID] + portAmount,
			ReservedAt:                  reservedAt,
			EstimatedAt:                 estimatedAt,
			UpdatedAt:                   updatedAt,
			UaslSections:                uaslSections,
			Ports:                       portElems,
			OperatingAircrafts:          opAircrafts,
			ConflictedFlightPlanIds:     res.ConflictedFlightPlanIDs,
			ConformityAssessmentResults: conformityResults,
		})
	}
	if len(notifications) == 0 {
		return nil
	}
	return &model.NotifyReservationCompletedRequest{Notifications: notifications}
}

func buildDestinationReservationLookup(
	parent *model.UaslReservationMessage,
	children []*model.UaslReservationMessage,
) map[string]string {
	lookup := make(map[string]string)
	for _, dr := range parent.DestinationReservations {
		if dr.ExUaslID == "" || dr.ReservationID == "" {
			continue
		}
		lookup[dr.ExUaslID] = dr.ReservationID
	}
	if len(lookup) > 0 {
		return lookup
	}

	for _, child := range children {
		if child == nil {
			continue
		}
		exUaslID := child.ExUaslID
		parentResID := child.ParentUaslReservationID
		adminID := child.ExAdministratorID
		if parent.ExAdministratorID != "" && adminID == parent.ExAdministratorID {
			continue
		}
		if exUaslID == "" || parentResID == "" {
			continue
		}
		if _, exists := lookup[exUaslID]; !exists {
			lookup[exUaslID] = parentResID
		}
	}
	return lookup
}

func buildDestinationsFromChildren(
	children []*model.UaslReservationMessage,
	parentAdminID string,
) []*model.DestinationReservationInfo {
	seen := make(map[string]*model.DestinationReservationInfo)
	for _, child := range children {
		if child == nil {
			continue
		}
		adminID := child.ExAdministratorID
		uaslID := child.ExUaslID
		reservationID := child.ID
		if parentAdminID != "" && adminID == parentAdminID {
			continue
		}
		if uaslID == "" || reservationID == "" {
			continue
		}
		if _, exists := seen[uaslID]; !exists {
			seen[uaslID] = &model.DestinationReservationInfo{
				ReservationID:     reservationID,
				ExUaslID:          uaslID,
				ExAdministratorID: adminID,
			}
		}
	}

	result := make([]*model.DestinationReservationInfo, 0, len(seen))
	for _, info := range seen {
		result = append(result, info)
	}
	return result
}

func getLastSequenceUaslID(children []*model.UaslReservationMessage) string {
	var lastSeq int32 = -1
	lastUaslID := ""
	for _, child := range children {
		if child == nil {
			continue
		}
		if child.Sequence >= lastSeq {
			lastSeq = child.Sequence
			lastUaslID = child.ExUaslID
		}
	}
	return lastUaslID
}

func groupSectionsByUasl(children []*model.UaslReservationMessage) (map[string][]*model.UaslSectionReservationElement, map[string]map[string]struct{}, map[string]int32) {
	sectionsByUasl := make(map[string][]*model.UaslSectionReservationElement)
	sectionIDsByUasl := make(map[string]map[string]struct{})
	amountByUasl := make(map[string]int32)
	for _, child := range children {
		if child == nil {
			continue
		}
		uaslID := child.ExUaslID
		if uaslID == "" {
			continue
		}
		sectionsByUasl[uaslID] = append(sectionsByUasl[uaslID], &model.UaslSectionReservationElement{
			UaslSectionID: child.ExUaslSectionID,
			Sequence:      int(child.Sequence),
			StartAt:       child.StartAt.Format(time.RFC3339),
			EndAt:         child.EndAt.Format(time.RFC3339),
			Amount:        int(child.Amount),
		})
		if _, ok := sectionIDsByUasl[uaslID]; !ok {
			sectionIDsByUasl[uaslID] = make(map[string]struct{})
		}
		if child.ExUaslSectionID != "" {
			sectionIDsByUasl[uaslID][child.ExUaslSectionID] = struct{}{}
		}
		amountByUasl[uaslID] += child.Amount
	}
	return sectionsByUasl, sectionIDsByUasl, amountByUasl
}

type sectionRange struct {
	start string
	end   string
}

func collectDestinationPorts(ports []*model.PortElement, destUaslID, lastSeqUaslID string) ([]*model.PortElement, int32) {
	if destUaslID == "" || destUaslID != lastSeqUaslID {
		return nil, 0
	}
	result := make([]*model.PortElement, 0)
	var total int32
	for _, port := range ports {
		if port == nil || port.UsageType != 2 {
			continue
		}
		result = append(result, port)
		total += int32(port.Amount)
	}
	return result, total
}

func buildSectionRangesFromNotifications(sections []*model.UaslSectionReservationElement) map[string][]sectionRange {
	ranges := make(map[string][]sectionRange)
	for _, section := range sections {
		if section == nil || section.UaslSectionID == "" {
			continue
		}
		ranges[section.UaslSectionID] = append(ranges[section.UaslSectionID], sectionRange{
			start: section.StartAt,
			end:   section.EndAt,
		})
	}
	return ranges
}

func filterConformityBySections(
	results []*model.ConformityAssessmentResultMsg,
	sectionIDs map[string]struct{},
	sectionRanges map[string][]sectionRange,
) []*model.ConformityAssessmentResultMsg {
	if len(results) == 0 {
		return nil
	}
	out := make([]*model.ConformityAssessmentResultMsg, 0, len(results))
	seen := make(map[string]struct{})
	for _, result := range results {
		if result == nil || result.UaslSectionID == "" {
			continue
		}
		if len(sectionRanges) > 0 {
			if _, ok := sectionRanges[result.UaslSectionID]; !ok {
				continue
			}
		} else if len(sectionIDs) > 0 {
			if _, ok := sectionIDs[result.UaslSectionID]; !ok {
				continue
			}
		}
		key := fmt.Sprintf("%s|%s|%s", result.UaslSectionID, result.Type, result.Reasons)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, result)
	}
	return out
}

func buildOperatingAircraftsFromConformity(results []*model.ConformityAssessmentResultMsg) []*model.ReservationAircraftInfo {
	if len(results) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]*model.ReservationAircraftInfo, 0)
	for _, result := range results {
		if result == nil || result.AircraftInfo == nil {
			continue
		}
		aircraft := result.AircraftInfo
		key := fmt.Sprintf("%d:%s:%s:%s", aircraft.AircraftInfoID, aircraft.RegistrationID, aircraft.Maker, aircraft.ModelNumber)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, &model.ReservationAircraftInfo{
			AircraftInfoID: aircraft.AircraftInfoID,
			RegistrationID: aircraft.RegistrationID,
			Maker:          aircraft.Maker,
			ModelNumber:    aircraft.ModelNumber,
			Name:           aircraft.Name,
			Type:           aircraft.Type,
			Length:         aircraft.Length,
		})
	}
	return out
}
func buildReservationEntity(item map[string]any) *reservationEntity {
	entity := &reservationEntity{
		ReservationID:           stringValue(item["reservationId"]),
		UaslID:                  stringValue(item["uaslId"]),
		AdministratorID:         stringValue(item["administratorId"]),
		SubTotalAmount:          int32(extractIntAmount(item["subTotalAmount"])),
		ConflictedFlightPlanIDs: getStringSlice(item["conflictedFlightPlanIds"]),
	}
	for _, section := range getMapSlice(item["uaslSections"]) {
		entity.UaslSections = append(entity.UaslSections, uaslSectionEntity{
			UaslSectionID: stringValue(section["uaslSectionId"]),
			Sequence:      int32(extractIntAmount(section["sequence"])),
			StartAt:       stringValue(section["startAt"]),
			EndAt:         stringValue(section["endAt"]),
			Amount:        int32(extractIntAmount(section["amount"])),
		})
	}
	for _, vehicle := range getMapSlice(item["vehicles"]) {
		v := vehicleElement{
			ReservationID: stringValue(vehicle["reservationId"]),
			VehicleID:     stringValue(vehicle["vehicleId"]),
			StartAt:       stringValue(vehicle["startAt"]),
			EndAt:         stringValue(vehicle["endAt"]),
			Amount:        int32(extractIntAmount(vehicle["amount"])),
		}
		if aircraft, ok := vehicle["aircraftInfo"].(map[string]any); ok {
			v.AircraftInfo = &aircraftInfo{
				AircraftInfoID: int32(extractIntAmount(aircraft["aircraftInfoId"])),
				RegistrationID: stringValue(aircraft["registrationId"]),
				Maker:          stringValue(aircraft["maker"]),
				ModelNumber:    stringValue(aircraft["modelNumber"]),
				Name:           stringValue(aircraft["name"]),
				Type:           stringValue(aircraft["type"]),
				Length:         parseAnyFloat(aircraft["length"]),
			}
		}
		entity.Vehicles = append(entity.Vehicles, v)
	}
	for _, port := range getMapSlice(item["ports"]) {
		entity.Ports = append(entity.Ports, portElement{
			PortID:        stringValue(port["portId"]),
			ReservationID: stringValue(port["reservationId"]),
			UsageType:     int32(extractIntAmount(port["usageType"])),
			StartAt:       stringValue(port["startAt"]),
			EndAt:         stringValue(port["endAt"]),
			Name:          stringValue(port["name"]),
			Amount:        int32(extractIntAmount(port["amount"])),
		})
	}
	for _, result := range getMapSlice(item["conformityAssessmentResults"]) {
		entity.ConformityAssessmentResults = append(entity.ConformityAssessmentResults, conformityAssessmentResult{
			UaslSectionID:     stringValue(result["uaslSectionId"]),
			AircraftInfo:      buildConformityAircraftInfo(result["aircraftInfo"]),
			EvaluationResults: fmt.Sprint(result["evaluationResults"]),
			Type:              stringValue(result["type"]),
			Reasons:           stringValue(result["reasons"]),
		})
	}
	return entity
}

func buildDestinationReservationEntity(item map[string]any) *destinationReservationEntity {
	base := buildReservationEntity(item)
	if base == nil {
		return nil
	}
	entity := &destinationReservationEntity{
		ReservationID:           base.ReservationID,
		UaslID:                  base.UaslID,
		AdministratorID:         base.AdministratorID,
		SubTotalAmount:          base.SubTotalAmount,
		UaslSections:            base.UaslSections,
		Ports:                   base.Ports,
		ConflictedFlightPlanIDs: base.ConflictedFlightPlanIDs,
	}
	for _, result := range base.ConformityAssessmentResults {
		entity.ConformityAssessmentResults = append(entity.ConformityAssessmentResults, destinationConformityAssessmentResult(result))
	}
	return entity
}

func buildConformityAircraftInfo(raw any) *conformityAircraftInfo {
	item, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	return &conformityAircraftInfo{
		Maker:       stringValue(item["maker"]),
		ModelNumber: stringValue(item["modelNumber"]),
		Name:        stringValue(item["name"]),
		Type:        stringValue(item["type"]),
		Length:      parseAnyFloat(item["length"]),
	}
}

func getStringSlice(raw any) []string {
	if items, ok := raw.([]string); ok {
		return items
	}
	values, ok := raw.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if s, ok := value.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func parseAnyFloat(v any) float64 {
	if parsed, ok := util.ParseFloat64FromInterface(v); ok {
		return parsed
	}
	return 0
}

func runPostReservationSideEffects(
	notify func(*model.NotifyReservationCompletedRequest) (*model.NotifyReservationCompletedResponse, error),
	response map[string]any,
	modelResponse *model.ReserveCompositeUaslResponse,
) {
	if err := publishMQTTMessage(response); err != nil {
		logger.LogError("Failed to publish MQTT message: %v", err)
	}
	req := buildNotifyReservationCompletedRequest(modelResponse)
	if req == nil || len(req.Notifications) == 0 {
		return
	}
	if _, err := notify(req); err != nil {
		logger.LogError("Failed to notify reservation completed: %v", err)
	}
}
