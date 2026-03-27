package model

import (
	"encoding/json"
	"fmt"
)

type UaslReservationRequest struct {
	RequestID                string               `json:"requestId,omitempty"`
	OperatorID               string               `json:"operatorId"`
	OriginAdministratorID    string               `json:"originAdministratorId,omitempty"`
	OriginUaslID             string               `json:"originUaslId,omitempty"`
	OperatingAircraft        *VehicleDetail       `json:"operatingAircraft,omitempty"`
	IsInterConnect           bool                 `json:"isInterConnect"`
	IgnoreFlightPlanConflict bool                 `json:"ignoreFlightPlanConflict"`
	UaslSections             []UaslSectionElement `json:"uaslSections"`
	Vehicles                 []VehicleElement     `json:"vehicles,omitempty"`
	Ports                    []PortElement        `json:"ports,omitempty"`
}

type UaslSectionElement struct {
	UaslID        string `json:"uaslId"`
	UaslSectionID string `json:"uaslSectionId"`
	Sequence      int    `json:"sequence"`
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
}

type VehicleElement struct {
	VehicleID    string               `json:"vehicleId"`
	AircraftInfo *AircraftInfoElement `json:"aircraftInfo,omitempty"`
	StartAt      string               `json:"startAt,omitempty"`
	EndAt        string               `json:"endAt,omitempty"`
}

type VehicleDetail struct {
	AircraftInfoID int32   `json:"aircraftInfoId"`
	RegistrationID string  `json:"registrationId"`
	Maker          string  `json:"maker"`
	ModelNumber    string  `json:"modelNumber"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	Length         float64 `json:"length"`
}

type PortElement struct {
	PortID    string `json:"portId"`
	UsageType int    `json:"usageType"`
	StartAt   string `json:"startAt"`
	EndAt     string `json:"endAt"`
}

type UaslReservationResponse struct {
	RequestID               string              `json:"requestId"`
	OperatorID              string              `json:"operatorId,omitempty"`
	Status                  string              `json:"status,omitempty"`
	TotalAmount             int                 `json:"totalAmount,omitempty"`
	FlightPurpose           string              `json:"flightPurpose,omitempty"`
	ReservedAt              string              `json:"reservedAt,omitempty"`
	UpdatedAt               string              `json:"updatedAt,omitempty"`
	EstimatedAt             string              `json:"estimatedAt,omitempty"`
	PricingRuleVersion      int                 `json:"pricingRuleVersion,omitempty"`
	OriginReservation       *ReservationEntity  `json:"originReservation,omitempty"`
	DestinationReservations []ReservationEntity `json:"destinationReservations,omitempty"`
}

type DestinationReservationNotification struct {
	RequestId                   string                          `json:"requestId"`
	ReservationId               string                          `json:"reservationId"`
	OperatorId                  string                          `json:"operatorId"`
	UaslId                      string                          `json:"uaslId"`
	AdministratorId             string                          `json:"administratorId"`
	FlightPurpose               string                          `json:"flightPurpose,omitempty"`
	Status                      string                          `json:"status"`
	SubTotalAmount              int32                           `json:"subTotalAmount,omitempty"`
	ReservedAt                  string                          `json:"reservedAt,omitempty"`
	EstimatedAt                 string                          `json:"estimatedAt,omitempty"`
	UpdatedAt                   string                          `json:"updatedAt,omitempty"`
	UaslSections                []UaslSectionReservationElement `json:"uaslSections,omitempty"`
	Ports                       []PortReservationElement        `json:"ports,omitempty"`
	OperatingAircrafts          []AircraftInfoElement           `json:"operatingAircrafts,omitempty"`
	ConflictedFlightPlanIds     []string                        `json:"conflictedFlightPlanIds,omitempty"`
	ConformityAssessmentResults []ConformityAssessmentResult    `json:"conformityAssessmentResults,omitempty"`
}

type ReservationEntity struct {
	ReservationID               string                          `json:"reservationId,omitempty"`
	UaslID                      string                          `json:"uaslId,omitempty"`
	AdministratorID             string                          `json:"administratorId,omitempty"`
	SubTotalAmount              int                             `json:"subTotalAmount,omitempty"`
	UaslSections                []UaslSectionReservationElement `json:"uaslSections,omitempty"`
	Vehicles                    []VehicleReservationElement     `json:"vehicles,omitempty"`
	Ports                       []PortReservationElement        `json:"ports,omitempty"`
	ConflictedFlightPlanIds     []string                        `json:"conflictedFlightPlanIds,omitempty"`
	ConformityAssessmentResults []ConformityAssessmentResult    `json:"conformityAssessmentResults,omitempty"`
}

type UaslSectionReservationElement struct {
	UaslSectionID string `json:"uaslSectionId"`
	Sequence      int    `json:"sequence,omitempty"`
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
	Amount        int    `json:"amount,omitempty"`
}

type VehicleReservationElement struct {
	VehicleID     string               `json:"vehicleId"`
	ReservationID string               `json:"reservationId,omitempty"`
	Name          string               `json:"name,omitempty"`
	StartAt       string               `json:"startAt,omitempty"`
	EndAt         string               `json:"endAt,omitempty"`
	Amount        int                  `json:"amount,omitempty"`
	AircraftInfo  *AircraftInfoElement `json:"aircraftInfo,omitempty"`
}

type PortReservationElement struct {
	PortID        string `json:"portId"`
	ReservationID string `json:"reservationId,omitempty"`
	Name          string `json:"name,omitempty"`
	UsageType     int    `json:"usageType"`
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
	Amount        int    `json:"amount,omitempty"`
}

type BoolString string

func (b *BoolString) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*b = ""
		return nil
	}

	var boolVal bool
	if err := json.Unmarshal(data, &boolVal); err == nil {
		if boolVal {
			*b = "true"
		} else {
			*b = "false"
		}
		return nil
	}

	var strVal string
	if err := json.Unmarshal(data, &strVal); err == nil {
		*b = BoolString(strVal)
		return nil
	}

	return fmt.Errorf("invalid BoolString value: %s", string(data))
}

type ConformityAssessmentResult struct {
	UaslSectionID     string               `json:"uaslSectionId"`
	EvaluationResults BoolString           `json:"evaluationResults"`
	Type              string               `json:"type,omitempty"`
	Reasons           string               `json:"reasons,omitempty"`
	AircraftInfo      *AircraftInfoElement `json:"aircraftInfo,omitempty"`
}

type AircraftInfoElement struct {
	AircraftInfoID int32   `json:"aircraftInfoId,omitempty"`
	RegistrationID string  `json:"registrationId,omitempty"`
	Maker          string  `json:"maker,omitempty"`
	ModelNumber    string  `json:"modelNumber,omitempty"`
	Name           string  `json:"name,omitempty"`
	Type           string  `json:"type,omitempty"`
	Length         float64 `json:"length,omitempty"`
}

type UaslReservationConfirmRequest struct {
	IsInterConnect bool `json:"isInterConnect"`
}

type AvailabilityRequest struct {
	UaslSections   []AvailabilityUaslSectionElement `json:"uaslSections"`
	Vehicles       []AvailabilityVehicleInput       `json:"vehicles,omitempty"`
	Ports          []AvailabilityPortInput          `json:"ports,omitempty"`
	IsInterConnect bool                             `json:"isInterConnect"`
}

type AvailabilityUaslSectionElement struct {
	UaslID        string `json:"uaslId"`
	UaslSectionID string `json:"uaslSectionId"`
}

type AvailabilityVehicleInput struct {
	VehicleID string `json:"vehicleId"`
}

type AvailabilityPortInput struct {
	PortID string `json:"portId"`
}

type AvailabilityResponse struct {
	Result AvailabilityItemList `json:"result"`
}

type AvailabilityItemList struct {
	UaslSections []AvailabilityItem        `json:"uaslSections"`
	Vehicles     []VehicleAvailabilityItem `json:"vehicles,omitempty"`
	Ports        []PortAvailabilityItem    `json:"ports,omitempty"`
}

type VehicleAvailabilityItem struct {
	RequestID     string `json:"requestId"`
	ReservationID string `json:"reservationId,omitempty"`
	Name          string `json:"name,omitempty"`
	VehicleID     string `json:"vehicleId"`
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
}

type PortAvailabilityItem struct {
	RequestID     string `json:"requestId"`
	ReservationID string `json:"reservationId,omitempty"`
	Name          string `json:"name,omitempty"`
	PortID        string `json:"portId"`
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
}

type UaslReservationListItem struct {
	RequestID               string              `json:"requestId"`
	OperatorID              string              `json:"operatorId"`
	Status                  string              `json:"status"`
	TotalAmount             int                 `json:"totalAmount,omitempty"`
	FlightPurpose           string              `json:"flightPurpose,omitempty"`
	EstimatedAt             string              `json:"estimatedAt,omitempty"`
	ReservedAt              string              `json:"reservedAt,omitempty"`
	UpdatedAt               string              `json:"updatedAt,omitempty"`
	OriginReservation       *ReservationEntity  `json:"originReservation,omitempty"`
	DestinationReservations []ReservationEntity `json:"destinationReservations,omitempty"`
}

type SearchByRequestIDsRequest struct {
	RequestIDs []string `json:"requestIds"`
}

type SearchByRequestIDsResponse struct {
	Result []UaslReservationListItem `json:"result"`
}

type AvailabilityItem struct {
	RequestID     string `json:"requestId"`
	OperatorID    string `json:"operatorId"`
	FlightPurpose string `json:"flightPurpose,omitempty"`
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
}

type EstimateRequest struct {
	UaslSections   []EstimateSectionElement `json:"uaslSections"`
	Vehicles       []VehicleElement         `json:"vehicles,omitempty"`
	Ports          []EstimatePortElement    `json:"ports,omitempty"`
	IsInterConnect bool                     `json:"isInterConnect"`
}

type EstimateSectionElement struct {
	UaslID        string `json:"uaslId,omitempty"`
	UaslSectionID string `json:"uaslSectionId"`
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
}

type EstimatePortElement struct {
	PortID  string `json:"portId"`
	StartAt string `json:"startAt"`
	EndAt   string `json:"endAt"`
}

type EstimateResponse struct {
	TotalAmount int32  `json:"totalAmount"`
	EstimatedAt string `json:"estimatedAt"`
}
