package value

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type NullInt32 struct {
	Int32 *int32
	Valid bool
}

func NewNullInt32(i *int32, v bool) NullInt32 {
	return NullInt32{
		Int32: i,
		Valid: v,
	}
}

func NewEmptyNullInt32() NullInt32 {
	return NullInt32{
		Valid: false,
	}
}

func NewNullInt32FromInt32(i int32) NullInt32 {
	return NullInt32{
		Int32: &i,
		Valid: true,
	}
}

func (ni *NullInt32) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*ni = NewNullInt32(nil, true)
		return nil
	}
	var i *int32
	if err := json.Unmarshal(data, &i); err == nil {
		*ni = NewNullInt32(i, true)
		return nil
	}
	return fmt.Errorf("invalid value for NullInt32: %s", string(data))
}

func (ni NullInt32) MarshalJSON() ([]byte, error) {
	if !ni.Valid || ni.Int32 == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*ni.Int32)
}

func (ni NullInt32) Value() (driver.Value, error) {
	if !ni.Valid || ni.Int32 == nil {
		return nil, nil
	}
	return int64(*ni.Int32), nil
}

func (ni *NullInt32) Scan(value interface{}) error {
	if value == nil {
		ni.Int32 = nil
		ni.Valid = true
		return nil
	}
	switch v := value.(type) {
	case int64:
		i32 := int32(v)
		ni.Int32 = &i32
		ni.Valid = true
	case int32:
		ni.Int32 = &v
		ni.Valid = true
	default:
		return fmt.Errorf("failed to scan NullInt32: unsupported type %T", value)
	}
	return nil
}

func (ni NullInt32) ToString() string {
	if !ni.Valid || ni.Int32 == nil {
		return ""
	}
	return fmt.Sprintf("%d", *ni.Int32)
}
