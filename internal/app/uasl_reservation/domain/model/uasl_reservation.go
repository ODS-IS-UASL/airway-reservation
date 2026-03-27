package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
	"uasl-reservation/internal/pkg/value"
)

type UaslReservation struct {
	ID                      value.ModelID              `json:"id" gorm:"column:id;primaryKey;type:uuid;default:uuid_generate_v4()"`
	ParentUaslReservationID *value.ModelID             `json:"parent_uasl_reservation_id" gorm:"column:parent_uasl_reservation_id;type:uuid"`
	RequestID               value.ModelID              `json:"request_id" gorm:"column:request_id;type:uuid;not null;index"`
	ExUaslSectionID         *string                    `json:"ex_uasl_section_id" gorm:"column:ex_uasl_section_id;type:varchar(255)"`
	ExUaslID                *string                    `json:"ex_uasl_id" gorm:"column:ex_uasl_id;type:varchar(255)"`
	ExAdministratorID       *string                    `json:"ex_administrator_id" gorm:"column:ex_administrator_id;type:varchar(255);index"`
	StartAt                 time.Time                  `json:"start_at" gorm:"column:start_at;not null"`
	EndAt                   time.Time                  `json:"end_at" gorm:"column:end_at;not null"`
	AcceptedAt              *time.Time                 `json:"accepted_at" gorm:"column:accepted_at"`
	AirspaceID              value.ModelID              `json:"airspace_id" gorm:"column:airspace_id;type:uuid;not null"`
	ExReservedBy            *value.ModelID             `json:"ex_reserved_by" gorm:"column:ex_reserved_by;type:uuid"`
	OrganizationID          *value.ModelID             `json:"organization_id" gorm:"column:organization_id;type:uuid"`
	ProjectID               *value.ModelID             `json:"project_id" gorm:"column:project_id;type:uuid;index"`
	OperationID             *value.ModelID             `json:"operation_id" gorm:"column:operation_id;type:uuid"`
	Status                  value.ReservationStatus    `json:"status" gorm:"column:status;type:reservation_status;not null"`
	PricingRuleVersion      *int                       `json:"pricing_rule_version" gorm:"column:pricing_rule_version;type:integer"`
	Amount                  *int                       `json:"amount" gorm:"column:amount;type:integer"`
	EstimatedAt             *time.Time                 `json:"estimated_at" gorm:"column:estimated_at"`
	FixedAt                 *time.Time                 `json:"fixed_at" gorm:"column:fixed_at"`
	Sequence                *int                       `json:"sequence" gorm:"column:sequence;type:integer"`
	ConformityAssessment    ConformityAssessmentList   `json:"conformity_assessment" gorm:"column:conformity_assessment;type:jsonb"`
	DestinationReservations DestinationReservationList `json:"destination_reservations" gorm:"column:destination_reservations;type:jsonb"`
	FlightPurpose           string                     `json:"flight_purpose" gorm:"->;column:flight_purpose"`
	CreatedAt               time.Time                  `json:"created_at" gorm:"column:created_at;not null;default:now()"`
	UpdatedAt               time.Time                  `json:"updated_at" gorm:"column:updated_at;not null;default:now()"`
}

func (a *UaslReservation) TableName() string {
	return "uasl_reservation.uasl_reservations"
}

type ConformityAssessmentItem struct {
	UaslSectionID     string            `json:"uaslSectionId"`
	StartAt           time.Time         `json:"startAt"`
	EndAt             time.Time         `json:"endAt"`
	AircraftInfo      VehicleDetailInfo `json:"aircraftInfo"`
	EvaluationResults bool              `json:"evaluationResults"`
	Type              string            `json:"type"`
	Reasons           string            `json:"reasons"`
}

type ConformityAssessmentList []ConformityAssessmentItem

func (c *ConformityAssessmentList) Scan(value interface{}) error {
	if value == nil {
		*c = ConformityAssessmentList{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}

	return json.Unmarshal(bytes, c)
}

func (c ConformityAssessmentList) Value() (driver.Value, error) {
	if len(c) == 0 {
		return nil, nil
	}
	return json.Marshal(c)
}

type DestinationReservationInfo struct {
	ReservationID     string `json:"reservationId"`
	ExUaslID          string `json:"exUaslId"`
	ExAdministratorID string `json:"exAdministratorId"`
}

type DestinationReservationList []DestinationReservationInfo

func (d *DestinationReservationList) Scan(value interface{}) error {
	if value == nil {
		*d = DestinationReservationList{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}
	return json.Unmarshal(bytes, d)
}

func (d DestinationReservationList) Value() (driver.Value, error) {
	if len(d) == 0 {
		return nil, nil
	}
	return json.Marshal(d)
}

type UaslReservationBatch struct {
	Parent   *UaslReservation   `json:"parent,omitempty"`
	Children []*UaslReservation `json:"children,omitempty"`
}

type UaslReservationListItem struct {
	Parent            *UaslReservation              `json:"parent,omitempty"`
	Children          []*UaslReservation            `json:"children,omitempty"`
	ExternalResources []ExternalResourceReservation `json:"external_resources,omitempty"`
	FlightPurpose     string                        `json:"flight_purpose,omitempty"`
}

type ExpiredUaslReservation struct {
	ParentID  value.ModelID `json:"parent_id"`
	RequestID value.ModelID `json:"request_id"`
}
