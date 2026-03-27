package value

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type NullString struct {
	String *string
	Valid  bool
}

func NewNullString(s *string, v bool) NullString {
	return NullString{
		String: s,
		Valid:  v,
	}
}

func (ns *NullString) ToString() string {
	if ns.Valid && ns.String != nil {
		return *ns.String
	}
	return ""
}

func NewEmptyNullString() NullString {
	return NullString{
		Valid: false,
	}
}

func (ns NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid || ns.String == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*ns.String)
}

func (ns *NullString) UnmarshalJSON(data []byte) error {

	if string(data) == "null" {
		*ns = NewNullString(nil, true)
		return nil
	}
	var s *string
	if err := json.Unmarshal(data, &s); err == nil {
		*ns = NewNullString(s, true)
		return nil
	}
	return fmt.Errorf("invalid value for NullString: %s", string(data))
}

func (n NullString) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	if n.String == nil {
		return nil, nil
	}
	return *n.String, nil
}

func (n *NullString) Scan(value interface{}) error {
	if value == nil {
		n.String = nil
		n.Valid = true
		return nil
	}

	stringValue, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan value: %v", value)
	}

	n.String = &stringValue
	n.Valid = true
	return nil
}
