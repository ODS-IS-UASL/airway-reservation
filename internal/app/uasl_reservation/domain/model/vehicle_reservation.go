package model

import (
	"time"

	"uasl-reservation/internal/pkg/value"
)

type VehicleReservationRequest struct {
	VehicleID          string         `json:"vehicleId"`
	OperatorID         *value.ModelID `json:"operatorId"`
	GroupReservationID string         `json:"groupReservationId"`
	AircraftInfo       *VehicleDetailInfo
	StartAt            time.Time `json:"startAt"`
	EndAt              time.Time `json:"endAt"`
}

type VehicleFetchRequest struct {
	VehicleID string     `json:"vehicleId"`
	StartAt   *time.Time `json:"startAt,omitempty"`
	EndAt     *time.Time `json:"endAt,omitempty"`
}

type VehicleReservationResponse struct {
	ID        value.ModelID `json:"id"`
	VehicleID string        `json:"vehicleId"`
	StartAt   time.Time     `json:"startAt"`
	EndAt     time.Time     `json:"endAt"`
}

type VehicleReservationInfo struct {
	ReservationID *string
	VehicleID     string
	VehicleName   string
	StartAt       time.Time
	EndAt         time.Time
}

type VehicleReservationDetail struct {
	ReservationID      string
	GroupReservationID string
	VehicleID          string
	StartAt            time.Time
	EndAt              time.Time
	VehicleName        string
	OperatorID         string
	Amount             int32
	AircraftInfo       *VehicleDetailInfo
}

type VehicleDetailInfo struct {
	AircraftInfoID int32  `json:"aircraft_info_id"`
	RegistrationID string `json:"registration_id"`
	Maker          string `json:"maker"`
	ModelNumber    string `json:"model_number"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Length         string `json:"length"`
}

type VehicleForAssessment struct {
	VehicleID    string
	AircraftInfo *VehicleDetailInfo
}

type AircraftFileInfo struct {
	FileLogicalName  string `json:"fileLogicalName"`
	FilePhysicalName string `json:"filePhysicalName"`
	FileID           string `json:"fileId"`
}

type AircraftPayloadInfo struct {
	PayloadID         string `json:"payloadId"`
	PayloadName       string `json:"payloadName"`
	PayloadDetailText string `json:"payloadDetailText"`
	ImageData         string `json:"imageData"`
	FilePhysicalName  string `json:"filePhysicalName"`
	OperatorID        string `json:"operatorId"`
}

type AircraftInfoDetail struct {
	AircraftID           string
	AircraftName         string
	Manufacturer         string
	ModelNumber          string
	ModelName            string
	ManufacturingNumber  string
	AircraftType         int
	MaxTakeoffWeight     float64
	BodyWeight           float64
	MaxFlightSpeed       float64
	MaxFlightTime        float64
	Certification        bool
	Lat                  float64
	Lon                  float64
	DipsRegistrationCode string
	OwnerType            int
	OwnerID              string
	OperatorID           string
	ImageData            string
	FileInfos            []AircraftFileInfo
	PayloadInfos         []AircraftPayloadInfo
	PriceInfos           []ResourcePriceInfo
}

type AircraftPriceInfo struct {
	PriceID            string
	PriceType          int
	PricePerUnit       int
	Price              int
	EffectiveStartTime string
	EffectiveEndTime   string
	Priority           int
	OperatorID         string
}
