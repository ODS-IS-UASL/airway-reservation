package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"

	"uasl-reservation/internal/pkg/value"
)

type ExternalUaslDefinition struct {
	ID                value.ModelID     `gorm:"column:id;type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ExUaslSectionID   string            `gorm:"column:ex_uasl_section_id;type:varchar(255);not null;uniqueIndex" json:"exUaslSectionId"`
	ExUaslID          value.NullString  `gorm:"column:ex_uasl_id;type:varchar(255)" json:"exUaslId"`
	ExAdministratorID string            `gorm:"column:ex_administrator_id;type:varchar(255);not null" json:"exAdministratorId"`
	Geometry          PostGISGeometry   `gorm:"column:geometry;type:geometry;not null" json:"geometry"`
	PointIDs          pq.StringArray    `gorm:"column:point_ids;type:text[];not null" json:"pointIds"`
	FlightPurpose     value.NullString  `gorm:"column:flight_purpose;type:varchar(255)" json:"flightPurpose"`
	PriceInfo         PriceInfoList     `gorm:"column:price_info;type:jsonb" json:"priceInfo"`
	PriceTimezone     string            `gorm:"column:price_timezone;type:varchar(50);not null;default:'UTC'" json:"priceTimezone"`
	PriceVersion      int               `gorm:"column:price_version;type:integer;not null;default:1" json:"priceVersion"`
	Status            UaslSectionStatus `gorm:"column:status;type:uasl_reservation.uasl_section_status;not null;default:'AVAILABLE'" json:"status"`
	CreatedAt         time.Time         `gorm:"column:created_at;not null;default:now()" json:"createdAt"`
	UpdatedAt         time.Time         `gorm:"column:updated_at;not null;default:now()" json:"updatedAt"`
}

func (ExternalUaslDefinition) TableName() string {
	return "uasl_reservation.external_uasl_definitions"
}

type PriceInfo struct {
	PriceType          string  `json:"priceType"`
	Price              float64 `json:"price"`
	PricePerUnit       int     `json:"pricePerUnit"`
	EffectiveStartDate string  `json:"effectiveStartDate"`
	EffectiveEndDate   string  `json:"effectiveEndDate"`
	EffectiveStartTime string  `json:"effectiveStartTime"`
	EffectiveEndTime   string  `json:"effectiveEndTime"`
}

type PriceInfoList []PriceInfo

func (p *PriceInfoList) Scan(value interface{}) error {
	if value == nil {
		*p = PriceInfoList{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}

	return json.Unmarshal(bytes, p)
}

func (p PriceInfoList) Value() (driver.Value, error) {
	if len(p) == 0 {
		return nil, nil
	}
	return json.Marshal(p)
}

func (d *ExternalUaslDefinition) CalculatePrice(startAt time.Time, durationMinutes int, location *time.Location) (int, error) {
	if durationMinutes <= 0 {
		return 0, fmt.Errorf("duration must be positive")
	}

	if location == nil {
		loc, err := time.LoadLocation(d.PriceTimezone)
		if err != nil {
			return 0, fmt.Errorf("invalid timezone: %s", d.PriceTimezone)
		}
		location = loc
	}

	localStart := startAt.In(location)

	if len(d.PriceInfo) == 0 {
		return 0, nil
	}

	var applicableRule *PriceInfo
	for i := range d.PriceInfo {
		rule := &d.PriceInfo[i]

		if rule.EffectiveStartDate != "" || rule.EffectiveEndDate != "" {
			layoutDate := "2006-01-02"
			if rule.EffectiveStartDate != "" {
				sd, err := time.ParseInLocation(layoutDate, rule.EffectiveStartDate, location)
				if err != nil {

					continue
				}
				if localStart.Before(sd) {
					continue
				}
			}
			if rule.EffectiveEndDate != "" {
				ed, err := time.ParseInLocation(layoutDate, rule.EffectiveEndDate, location)
				if err != nil {
					continue
				}

				endOfDay := ed.Add(24 * time.Hour)
				if localStart.After(endOfDay) {
					continue
				}
			}
		}

		layoutTime := "15:04"
		sTimeStr := rule.EffectiveStartTime
		eTimeStr := rule.EffectiveEndTime
		if sTimeStr == "" {
			sTimeStr = "00:00"
		}
		if eTimeStr == "" {
			eTimeStr = "24:00"
		}

		parseHM := func(v string) (t time.Time, isMidnight24 bool, err error) {
			if v == "24:00" {

				tt, err := time.ParseInLocation(layoutTime, "00:00", location)
				return tt, true, err
			}
			tt, err := time.ParseInLocation(layoutTime, v, location)
			return tt, false, err
		}

		sT, s24, err1 := parseHM(sTimeStr)
		eT, e24, err2 := parseHM(eTimeStr)
		if err1 != nil || err2 != nil {
			continue
		}

		startOfDay := time.Date(localStart.Year(), localStart.Month(), localStart.Day(), 0, 0, 0, 0, location)
		sDateTime := startOfDay.Add(time.Duration(sT.Hour())*time.Hour + time.Duration(sT.Minute())*time.Minute)
		eDateTime := startOfDay.Add(time.Duration(eT.Hour())*time.Hour + time.Duration(eT.Minute())*time.Minute)
		if e24 {

			eDateTime = eDateTime.Add(24 * time.Hour)
		}
		if s24 {
			sDateTime = sDateTime.Add(24 * time.Hour)
		}

		crossesMidnight := eDateTime.Before(sDateTime) || eDateTime.Equal(sDateTime)

		if crossesMidnight {

			if !(localStart.Equal(sDateTime) || localStart.After(sDateTime) || localStart.Before(eDateTime)) {
				continue
			}
		} else {
			if localStart.Before(sDateTime) || localStart.Equal(eDateTime) || localStart.After(eDateTime) {
				continue
			}
		}

		applicableRule = rule
		break
	}

	if applicableRule == nil {
		return 0, nil
	}

	switch applicableRule.PriceType {
	case "TIME_MINUTE":
		totalPrice := float64(durationMinutes) * applicableRule.Price
		return int(totalPrice), nil
	default:

		return 0, nil
	}
}

type UaslSectionStatus string

const (
	UaslSectionStatusAvailable UaslSectionStatus = "AVAILABLE"
	UaslSectionStatusClosed    UaslSectionStatus = "CLOSED"
)

type PostGISGeometry struct {
	EWKT  []byte
	Valid bool
}

func NewPostGISGeometry(ewkt []byte) PostGISGeometry {
	if len(ewkt) == 0 {
		return PostGISGeometry{
			EWKT:  []byte{},
			Valid: false,
		}
	}
	return PostGISGeometry{
		EWKT:  ewkt,
		Valid: true,
	}
}

func NewEmptyPostGISGeometry() PostGISGeometry {
	return PostGISGeometry{
		EWKT:  []byte{},
		Valid: false,
	}
}

func (g PostGISGeometry) Value() (driver.Value, error) {
	if !g.Valid || len(g.EWKT) == 0 {
		return nil, nil
	}

	return string(g.EWKT), nil
}

func (g *PostGISGeometry) Scan(value interface{}) error {
	if value == nil {
		g.EWKT = []byte{}
		g.Valid = false
		return nil
	}

	switch v := value.(type) {
	case []byte:

		g.EWKT = v
		g.Valid = true
		return nil
	case string:

		g.EWKT = []byte(v)
		g.Valid = true
		return nil
	default:
		return fmt.Errorf("unsupported type for PostGISGeometry.Scan: %T", value)
	}
}

func (g PostGISGeometry) ToBytes() []byte {
	if !g.Valid {
		return []byte{}
	}
	return g.EWKT
}

func (g PostGISGeometry) ToString() string {
	if !g.Valid {
		return ""
	}
	return string(g.EWKT)
}

func (g PostGISGeometry) IsEmpty() bool {
	return !g.Valid || len(g.EWKT) == 0
}
