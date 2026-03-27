package value

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type NullJSON struct {
	JSON  *json.RawMessage
	Valid bool
}

func NewNullJSON(j *json.RawMessage, v bool) NullJSON {
	return NullJSON{
		JSON:  j,
		Valid: v,
	}
}

func NewEmptyNullJSON() NullJSON {
	return NullJSON{
		JSON:  nil,
		Valid: false,
	}
}

func (nj *NullJSON) UnmarshalJSON(data []byte) error {

	if string(data) == "null" {
		*nj = NewNullJSON(nil, true)
		return nil
	}

	if len(data) > 0 {
		rawMsg := json.RawMessage(data)
		*nj = NewNullJSON(&rawMsg, true)
		return nil
	}
	return fmt.Errorf("invalid value for NullJSON: %s", string(data))
}

func (nj *NullJSON) MarshalJSON() ([]byte, error) {
	if !nj.Valid {
		return []byte("null"), nil
	}
	if nj.JSON == nil {
		return []byte("null"), nil
	}
	return *nj.JSON, nil
}

func (nj *NullJSON) ToString() string {
	if nj.Valid && nj.JSON != nil {
		return string(*nj.JSON)
	}
	return ""
}

func (nj *NullJSON) ToStringPointer() *string {
	if nj.Valid && nj.JSON != nil {
		str := string(*nj.JSON)
		return &str
	}
	return nil
}

func NewNullJSONFromStringPointer(s *string) NullJSON {
	if s == nil {
		return NewNullJSON(nil, true)
	}
	rawMsg := json.RawMessage(*s)
	return NewNullJSON(&rawMsg, true)
}

func NewNullJSONFromString(s string) NullJSON {
	if s == "" {
		return NewNullJSON(nil, true)
	}
	rawMsg := json.RawMessage(s)
	return NewNullJSON(&rawMsg, true)
}

func (nj NullJSON) Value() (driver.Value, error) {
	if !nj.Valid {
		return nil, nil
	}
	if nj.JSON == nil {
		return nil, nil
	}
	return string(*nj.JSON), nil
}

func (nj *NullJSON) Scan(value interface{}) error {
	if value == nil {
		nj.JSON = nil
		nj.Valid = true
		return nil
	}

	switch v := value.(type) {
	case string:
		rawMsg := json.RawMessage(v)
		nj.JSON = &rawMsg
		nj.Valid = true
	case []byte:
		rawMsg := json.RawMessage(v)
		nj.JSON = &rawMsg
		nj.Valid = true
	default:
		return fmt.Errorf("failed to scan value: %v", value)
	}

	return nil
}

func (nj NullJSON) IsValidJSON() bool {
	if !nj.Valid || nj.JSON == nil {
		return false
	}
	var js interface{}
	return json.Unmarshal(*nj.JSON, &js) == nil
}
