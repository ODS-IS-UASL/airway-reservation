package model

type UaslReservationResponse struct {
	Data  *UaslReservationData `json:"data,omitempty"`
	Error *ResponseError       `json:"error,omitempty"`
}

type UaslReservationData struct {
	RequestID               string              `json:"request_id"`
	OperatorID              string              `json:"operator_id,omitempty"`
	Status                  string              `json:"status,omitempty"`
	TotalAmount             int                 `json:"total_amount,omitempty"`
	FlightPurpose           string              `json:"flight_purpose,omitempty"`
	ReservedAt              string              `json:"reserved_at,omitempty"`
	UpdatedAt               string              `json:"updated_at,omitempty"`
	EstimatedAt             string              `json:"estimated_at,omitempty"`
	PricingRuleVersion      int                 `json:"pricing_rule_version,omitempty"`
	OriginReservation       *ReservationEntity  `json:"origin_reservation,omitempty"`
	DestinationReservations []ReservationEntity `json:"destination_reservations,omitempty"`
}

type ReservationEntity struct {
	ReservationID               string                          `json:"reservation_id,omitempty"`
	UaslID                      string                          `json:"uasl_id,omitempty"`
	AdministratorID             string                          `json:"administrator_id,omitempty"`
	SubTotalAmount              int                             `json:"sub_total_amount,omitempty"`
	UaslSections                []UaslSectionReservationElement `json:"uasl_sections,omitempty"`
	Vehicles                    []VehicleReservationElement     `json:"vehicles,omitempty"`
	Ports                       []PortReservationElement        `json:"ports,omitempty"`
	ConflictedFlightPlanIds     []string                        `json:"conflicted_flight_plan_ids,omitempty"`
	ConformityAssessmentResults []ConformityAssessmentResult    `json:"conformity_assessment_results,omitempty"`
}

type ResponseError struct {
	Code    string                 `json:"code,omitempty"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type UaslSectionReservationElement struct {
	UaslSectionID string `json:"uasl_section_id"`
	Sequence      int    `json:"sequence,omitempty"`
	StartAt       string `json:"start_at"`
	EndAt         string `json:"end_at"`
	Amount        int    `json:"amount,omitempty"`
}

type VehicleReservationElement struct {
	VehicleID     string                   `json:"vehicle_id"`
	ReservationID string                   `json:"reservation_id,omitempty"`
	Name          string                   `json:"name,omitempty"`
	StartAt       string                   `json:"start_at,omitempty"`
	EndAt         string                   `json:"end_at,omitempty"`
	Amount        int                      `json:"amount,omitempty"`
	AircraftInfo  *ReservationAircraftInfo `json:"aircraft_info,omitempty"`
}

type PortReservationElement struct {
	PortID            string `json:"port_id"`
	ReservationID     string `json:"reservation_id,omitempty"`
	Name              string `json:"name,omitempty"`
	UsageType         int    `json:"usage_type"`
	StartAt           string `json:"start_at"`
	EndAt             string `json:"end_at"`
	Amount            int    `json:"amount,omitempty"`
	ExAdministratorID string `json:"ex_administrator_id,omitempty"`
}

type ConformityAssessmentResult struct {
	UaslSectionID     string                   `json:"uasl_section_id"`
	EvaluationResults string                   `json:"evaluation_results"`
	Type              string                   `json:"type,omitempty"`
	Reasons           string                   `json:"reasons,omitempty"`
	AircraftInfo      *ReservationAircraftInfo `json:"aircraft_info,omitempty"`
}

type ReservationAircraftInfo struct {
	AircraftInfoID int32   `json:"aircraft_info_id,omitempty"`
	RegistrationID string  `json:"registration_id,omitempty"`
	Maker          string  `json:"maker,omitempty"`
	ModelNumber    string  `json:"model_number,omitempty"`
	Name           string  `json:"name,omitempty"`
	Type           string  `json:"type,omitempty"`
	Length         float64 `json:"length,omitempty"`
}

type ExternalReservationSearchRequest struct {
	RequestIDs []string `json:"requestIds"`
}

type ExternalReservationListResponse struct {
	Result []ExternalReservationListItem `json:"result"`
}

type ExternalReservationListItem struct {
	RequestID               string              `json:"requestId"`
	OperatorID              string              `json:"operatorId"`
	Status                  string              `json:"status"`
	TotalAmount             int                 `json:"totalAmount"`
	EstimatedAt             string              `json:"estimatedAt,omitempty"`
	ReservedAt              string              `json:"reservedAt,omitempty"`
	UpdatedAt               string              `json:"updatedAt,omitempty"`
	FlightPurpose           string              `json:"flightPurpose,omitempty"`
	OriginReservation       *ReservationEntity  `json:"originReservation,omitempty"`
	DestinationReservations []ReservationEntity `json:"destinationReservations,omitempty"`
}

type ExternalAvailabilityGroupedResponse struct {
	UaslSections []ExternalAvailabilitySectionItem `json:"uaslSections,omitempty"`
	Vehicles     []ExternalAvailabilityVehicleItem `json:"vehicles,omitempty"`
	Ports        []ExternalAvailabilityPortItem    `json:"ports,omitempty"`
}

type ExternalAvailabilitySectionItem struct {
	RequestID     string `json:"requestId"`
	OperatorID    string `json:"operatorId,omitempty"`
	FlightPurpose string `json:"flightPurpose,omitempty"`
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
}

type ExternalAvailabilityVehicleItem struct {
	VehicleID     string                   `json:"vehicleId"`
	ReservationID string                   `json:"reservationId,omitempty"`
	RequestID     string                   `json:"requestId,omitempty"`
	AircraftInfo  *ReservationAircraftInfo `json:"aircraftInfo,omitempty"`
	StartAt       string                   `json:"startAt,omitempty"`
	EndAt         string                   `json:"endAt,omitempty"`
	Name          string                   `json:"name,omitempty"`
}

type ExternalAvailabilityPortItem struct {
	PortID        string `json:"portId"`
	ReservationID string `json:"reservationId,omitempty"`
	RequestID     string `json:"requestId,omitempty"`
	UsageType     int    `json:"usageType,omitempty"`
	StartAt       string `json:"startAt,omitempty"`
	EndAt         string `json:"endAt,omitempty"`
	Name          string `json:"name,omitempty"`
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
	OperatingAircrafts          []ReservationAircraftInfo       `json:"operatingAircrafts,omitempty"`
	ConflictedFlightPlanIds     []string                        `json:"conflictedFlightPlanIds,omitempty"`
	ConformityAssessmentResults []ConformityAssessmentResult    `json:"conformityAssessmentResults,omitempty"`
}

type ExternalEstimateRequest struct {
	UaslSections   []ExternalEstimateSectionRequest `json:"uaslSections"`
	Vehicles       []ExternalEstimateVehicleRequest `json:"vehicles,omitempty"`
	Ports          []ExternalEstimatePortRequest    `json:"ports,omitempty"`
	IsInterConnect bool                             `json:"isInterConnect"`
}

type ExternalEstimateSectionRequest struct {
	UaslID        string `json:"uaslId,omitempty"`
	UaslSectionID string `json:"uaslSectionId"`
	StartAt       string `json:"startAt"`
	EndAt         string `json:"endAt"`
}

type ExternalEstimateVehicleRequest struct {
	VehicleID string `json:"vehicleId"`
	StartAt   string `json:"startAt,omitempty"`
	EndAt     string `json:"endAt,omitempty"`
}

type ExternalEstimatePortRequest struct {
	PortID  string `json:"portId"`
	StartAt string `json:"startAt"`
	EndAt   string `json:"endAt"`
}

type ExternalEstimateResponse struct {
	TotalAmount int32  `json:"totalAmount"`
	EstimatedAt string `json:"estimatedAt"`
}
