package model

import "time"

type VehicleElement = VehicleReservationElement

type PortElement = PortReservationElement

type UaslReservationMessage struct {
	ID                          string
	ParentUaslReservationID     string
	ExUaslSectionID             string
	StartAt                     time.Time
	EndAt                       time.Time
	AcceptedAt                  *time.Time
	ExReservedBy                string
	AirspaceID                  string
	ProjectID                   string
	OperationID                 string
	OrganizationID              string
	Status                      string
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
	RequestID                   string
	ExUaslID                    string
	ExAdministratorID           string
	PricingRuleVersion          int32
	Amount                      int32
	EstimatedAt                 *time.Time
	FixedAt                     *time.Time
	Sequence                    int32
	FlightPurpose               string
	DestinationReservations     []*DestinationReservationInfo
	ConformityAssessmentResults []*ConformityAssessmentResultMsg
}

type ConformityAssessmentResultMsg struct {
	UaslSectionID     string
	AircraftInfo      *ReservationAircraftInfo
	EvaluationResults bool
	Type              string
	Reasons           string
}

type RegisterUaslReservationResponse struct {
	Data *UaslReservationMessage
}

type DeleteUaslReservationResponse struct {
	Data *UaslReservationMessage
}

type FindUaslReservationResponse struct {
	Parent                  *UaslReservationMessage
	ConflictedFlightPlanIDs []string
	Children                []*UaslReservationMessage
	Vehicles                []*VehicleElement
	Ports                   []*PortElement
}

type ReserveCompositeUaslResponse struct {
	ConflictedFlightPlanIDs []string
	ParentUaslReservation   *UaslReservationMessage
	ChildUaslReservations   []*UaslReservationMessage
	Vehicles                []*VehicleElement
	Ports                   []*PortElement
	ConflictType            string
	ConflictedResourceIds   []string
}

type UaslReservationListItemMsg struct {
	ParentUaslReservation *UaslReservationMessage
	ChildUaslReservations []*UaslReservationMessage
	Vehicles              []*VehicleElement
	Ports                 []*PortElement
	FlightPurpose         string
}

type PaginationInfo struct {
	CurrentPage int32
	LastPage    int32
	PerPage     int32
	Total       int32
}

type ListUaslReservationsResponse struct {
	Result   []*UaslReservationListItemMsg
	PageInfo *PaginationInfo
}

type SearchByConditionResponse struct {
	Result []*UaslReservationListItemMsg
}

type EstimateUaslReservationResponse struct {
	TotalAmount int32
	EstimatedAt string
}

type GetAvailabilityResponse struct {
	UaslSections []*AvailabilityItem
	Vehicles     []*VehicleAvailabilityItem
	Ports        []*PortAvailabilityItem
}

type NotifyReservationCompletedRequest struct {
	Notifications []*DestinationReservationNotification
}

type NotificationResult struct {
	ReservationID   string
	UaslID          string
	AdministratorID string
}

type NotifyReservationCompletedResponse struct {
	Sent   []*NotificationResult
	Failed []*NotificationResult
}
