package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"uasl-reservation/internal/pkg/value"

	"gorm.io/datatypes"
)

type ExternalResourcePriceInfo struct {
	PriceType          int    `json:"priceType"`
	Price              int    `json:"price"`
	PricePerUnit       int    `json:"pricePerUnit"`
	Priority           int    `json:"priority"`
	EffectiveStartTime string `json:"effectiveStartTime"`
	EffectiveEndTime   string `json:"effectiveEndTime"`
}

type ExternalResourcePriceInfoList []ExternalResourcePriceInfo

func (p *ExternalResourcePriceInfoList) Scan(value interface{}) error {
	if value == nil {
		*p = ExternalResourcePriceInfoList{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}

	return json.Unmarshal(bytes, p)
}

func (p ExternalResourcePriceInfoList) Value() (driver.Value, error) {
	if len(p) == 0 {
		return nil, nil
	}
	return json.Marshal(p)
}

type ExternalUaslResource struct {
	ID                      value.ModelID                 `gorm:"column:id;type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name                    string                        `gorm:"column:name;type:varchar(255);not null" json:"name"`
	ResourceID              *value.ModelID                `gorm:"column:resource_id;type:uuid" json:"resourceId"`
	ExUaslSectionID         string                        `gorm:"column:ex_uasl_section_id;type:varchar(255);" json:"exUaslSectionId"`
	ResourceType            ExternalResourceType          `gorm:"column:resource_type;type:varchar(32);not null" json:"resourceType"`
	ExResourceID            string                        `gorm:"column:ex_resource_id;type:varchar(255);not null;uniqueIndex" json:"exResourceId"`
	OrganizationID          value.ModelID                 `gorm:"column:organization_id;type:uuid" json:"organizationId"`
	EstimatedPricePerMinute ExternalResourcePriceInfoList `gorm:"column:estimated_price_per_minute;type:jsonb" json:"estimatedPricePerMinute"`
	AircraftInfo            *datatypes.JSON               `gorm:"column:aircraft_info;type:jsonb" json:"aircraftInfo"`
	CreatedAt               time.Time                     `gorm:"column:created_at;not null;default:now()" json:"createdAt"`
	UpdatedAt               time.Time                     `gorm:"column:updated_at;not null;default:now()" json:"updatedAt"`
}

func (ExternalUaslResource) TableName() string {
	return "uasl_reservation.external_uasl_resources"
}
