package model

import (
	"fmt"
	"time"

	"uasl-reservation/internal/pkg/value"

	"github.com/google/uuid"
)

type ExternalResourceType = ResourceType

const (
	ExternalResourceTypeVehicle = ResourceTypeVehicle
	ExternalResourceTypePort    = ResourceTypePort
	ExternalResourceTypePayload = ResourceTypePayload
)

type ExternalResourceReservation struct {
	ID                value.ModelID        `gorm:"column:id;primaryKey;type:uuid;default:uuid_generate_v4()"`
	RequestID         value.ModelID        `gorm:"column:request_id;type:uuid;not null"`
	ExAdministratorID string               `gorm:"column:ex_administrator_id;type:varchar(255)"`
	ExReservationID   string               `gorm:"column:ex_reservation_id;type:varchar(255)"`
	ExResourceID      string               `gorm:"column:ex_resource_id;type:varchar(255)"`
	StartAt           *time.Time           `gorm:"column:start_at;type:timestamp(0) with time zone"`
	EndAt             *time.Time           `gorm:"column:end_at;type:timestamp(0) with time zone"`
	UsageType         *int                 `gorm:"column:usage_type;type:integer"`
	Amount            *int                 `gorm:"column:amount;type:integer"`
	ResourceType      ExternalResourceType `gorm:"column:resource_type;type:varchar(32);not null"`
	ResourceName      string               `gorm:"->;column:resource_name"`
	CreatedAt         time.Time            `gorm:"column:created_at;type:timestamp(0) with time zone;not null;default:now()"`
	UpdatedAt         time.Time            `gorm:"column:updated_at;type:timestamp(0) with time zone;not null;default:now()"`
}

func (ExternalResourceReservation) TableName() string {
	return "uasl_reservation.external_resource_reservations"
}

func NewExternalResourceReservation(
	requestID value.ModelID,
	exReservationID string,
	resourceType ExternalResourceType,
) (*ExternalResourceReservation, error) {
	if requestID == "" {
		return nil, fmt.Errorf("request ID must not be empty")
	}
	if err := resourceType.Validate(); err != nil {
		return nil, err
	}

	now := time.Now()
	defaultAmount := 0
	return &ExternalResourceReservation{
		ID:              value.ModelID(uuid.New().String()),
		RequestID:       requestID,
		ExReservationID: exReservationID,
		ResourceType:    resourceType,
		Amount:          &defaultAmount,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func (err *ExternalResourceReservation) Validate() error {
	if err.ID == "" {
		return fmt.Errorf("reservation ID must not be empty")
	}
	if err.RequestID == "" {
		return fmt.Errorf("request ID must not be empty")
	}
	if validationErr := err.ResourceType.Validate(); validationErr != nil {
		return validationErr
	}
	return nil
}

type ExternalResourceData struct {
	Vehicles []*VehicleReservationDetail
	Ports    []*PortReservationDetail
}
