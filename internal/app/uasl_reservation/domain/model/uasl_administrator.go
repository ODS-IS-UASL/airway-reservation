package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"uasl-reservation/internal/pkg/value"
)

type UaslAdministrator struct {
	ID                value.ModelID    `gorm:"column:id;type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ExAdministratorID string           `gorm:"column:ex_administrator_id;type:varchar(255);not null;uniqueIndex" json:"exAdministratorId"`
	BusinessNumber    value.NullString `gorm:"column:business_number;type:varchar(255)" json:"businessNumber"`
	Name              string           `gorm:"column:name;type:varchar(255);not null" json:"name"`
	IsInternal        bool             `gorm:"column:is_internal;type:boolean;not null;default:false" json:"isInternal"`
	ExternalServices  value.NullJSON   `gorm:"column:external_services;type:jsonb" json:"externalServices"`
	CreatedAt         time.Time        `gorm:"column:created_at;not null;default:now()" json:"createdAt"`
	UpdatedAt         time.Time        `gorm:"column:updated_at;not null;default:now()" json:"updatedAt"`
}

func (UaslAdministrator) TableName() string {
	return "uasl_reservation.uasl_administrators"
}

func (a *UaslAdministrator) IsExternal() bool {
	return !a.IsInternal
}

type ExternalService struct {
	ExUaslID string                   `json:"exUaslId"`
	Services ExternalServiceEndpoints `json:"services"`
}

type ExternalServiceEndpoints struct {
	BaseURL string `json:"baseUrl,omitempty"`
}

type ExternalServicesList []ExternalService

func (e *ExternalServicesList) Scan(value interface{}) error {
	if value == nil {
		*e = ExternalServicesList{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}

	return json.Unmarshal(bytes, e)
}

func (e ExternalServicesList) Value() (driver.Value, error) {
	if len(e) == 0 {
		return nil, nil
	}
	return json.Marshal(e)
}

func (e ExternalServicesList) GetServiceByExUaslID(exUaslID string) *ExternalService {
	for i := range e {
		if e[i].ExUaslID == exUaslID {
			return &e[i]
		}
	}
	return nil
}

func (a *UaslAdministrator) GetExternalServicesList() (ExternalServicesList, error) {
	if !a.ExternalServices.Valid || a.ExternalServices.JSON == nil {
		return nil, nil
	}
	var servicesList ExternalServicesList
	if err := json.Unmarshal(*a.ExternalServices.JSON, &servicesList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ExternalServices for administrator %s: %w", a.ExAdministratorID, err)
	}
	return servicesList, nil
}
