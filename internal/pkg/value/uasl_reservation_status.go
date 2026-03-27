package value

import (
	"encoding/json"
	"fmt"
)

type ReservationStatus string

const (
	RESERVATION_STATUS_PENDING   ReservationStatus = "PENDING"
	RESERVATION_STATUS_RESERVED  ReservationStatus = "RESERVED"
	RESERVATION_STATUS_CANCELED  ReservationStatus = "CANCELED"
	RESERVATION_STATUS_RESCINDED ReservationStatus = "RESCINDED"
	RESERVATION_STATUS_INHERITED ReservationStatus = "INHERITED"
)

func (status ReservationStatus) ToString() string {
	return string(status)
}

func (status *ReservationStatus) UnmarshalJSON(data []byte) error {
	var statusString string
	if err := json.Unmarshal(data, &statusString); err != nil {
		return err
	}

	switch statusString {
	case "PENDING", "RESERVED", "CANCELED", "RESCINDED", "INHERITED":
		*status = ReservationStatus(statusString)
	default:
		return fmt.Errorf("invalid reservation status: %s", statusString)
	}

	return nil
}
