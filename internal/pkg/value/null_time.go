package value

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type NullTime struct {
	Time  *time.Time
	Valid bool
}

func NewNullTime(t *time.Time, v bool) NullTime {
	return NullTime{
		Time:  t,
		Valid: v,
	}
}

func NewEmptyNullTime() NullTime {
	return NullTime{
		Valid: false,
	}
}

func NewNullTimeFromTime(t time.Time) NullTime {
	return NullTime{
		Time:  &t,
		Valid: true,
	}
}

func NewNullTimeFromString(s string) (NullTime, error) {
	if s == "" {
		return NewNullTime(nil, true), nil
	}

	parsedTime, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return NewEmptyNullTime(), fmt.Errorf("failed to parse RFC3339 time string: %w", err)
	}

	return NewNullTime(&parsedTime, true), nil
}

func NewNullTimeFromStringPointer(s *string) (NullTime, error) {
	if s == nil {
		return NewNullTime(nil, true), nil
	}
	return NewNullTimeFromString(*s)
}

func (nt NullTime) ToString() string {
	if nt.Valid && nt.Time != nil {
		return nt.Time.Format(time.RFC3339)
	}
	return ""
}

func (nt NullTime) ToStringPointer() *string {
	if nt.Valid && nt.Time != nil {
		str := nt.Time.Format(time.RFC3339)
		return &str
	}
	return nil
}

func (nt NullTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid || nt.Time == nil {
		return []byte("null"), nil
	}
	return json.Marshal(nt.Time.Format(time.RFC3339))
}

func (nt *NullTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*nt = NewNullTime(nil, true)
		return nil
	}
	var t *time.Time
	if err := json.Unmarshal(data, &t); err == nil {
		*nt = NewNullTime(t, true)
		return nil
	}
	return fmt.Errorf("invalid value for NullTime: %s", string(data))
}

func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	if nt.Time == nil {
		return nil, nil
	}
	return *nt.Time, nil
}

func (nt *NullTime) Scan(value interface{}) error {
	if value == nil {
		nt.Time = nil
		nt.Valid = true
		return nil
	}

	timeValue, ok := value.(time.Time)
	if !ok {
		return fmt.Errorf("failed to scan value: %v", value)
	}

	nt.Time = &timeValue
	nt.Valid = true
	return nil
}

func (nt NullTime) ToUtc() *NullTime {
	if nt.Time == nil || !nt.Valid {
		return &NullTime{Valid: true}
	}
	utc := nt.Time.UTC()
	return &NullTime{
		Time:  &utc,
		Valid: true,
	}
}
