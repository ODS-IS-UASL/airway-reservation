package model

import (
	"time"

	"uasl-reservation/internal/pkg/value"
)

type PortReservationRequest struct {
	PortID              string         `json:"dronePortId"`
	AircraftID          *string        `json:"aircraftId,omitempty"`
	GroupReservationID  string         `json:"groupReservationId"`
	UsageType           int32          `json:"usageType"`
	ReservationTimeFrom time.Time      `json:"reservationTimeFrom"`
	ReservationTimeTo   time.Time      `json:"reservationTimeTo"`
	OperatorID          *value.ModelID `json:"operatorId,omitempty"`
}

type PortFetchRequest struct {
	PortID   string     `json:"dronePortId"`
	TimeFrom *time.Time `json:"timeFrom,omitempty"`
	TimeTo   *time.Time `json:"timeTo,omitempty"`
}

type PortReservationResponse struct {
	DronePortReservationIDs []string `json:"dronePortReservationIds"`
}

type PortReservationInfo struct {
	ReservationID *string
	PortID        string
	PortName      string
	VehicleID     *string
	UsageType     int32
	StartAt       time.Time
	EndAt         time.Time
}

type PortReservationDetail struct {
	ReservationID         string
	GroupReservationID    string
	PortID                string
	PortName              string
	VehicleID             *string
	VehicleName           *string
	UsageType             int32
	StartAt               time.Time
	EndAt                 time.Time
	VisDronePortCompanyID *string
	ReservationActiveFlag bool
	OperatorID            string
	Amount                int32
	ExAdministratorID     string
}

func (p *PortReservationInfo) ToTimeRange() TimeRange {
	return TimeRange{Start: p.StartAt, End: p.EndAt}
}

func (p *PortReservationRequest) ToTimeRange() TimeRange {
	return TimeRange{Start: p.ReservationTimeFrom, End: p.ReservationTimeTo}
}

const (
	DronePortActiveStatusAvailable = 2
)

type DronePortInfo struct {
	DronePortID   string
	DronePortName string
	ActiveStatus  int
}
