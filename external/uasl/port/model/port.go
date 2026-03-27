package model

import (
	"time"

	resourcepricemodel "uasl-reservation/external/uasl/resource_price/model"
)

type DronePortInfoRegisterRequestDto struct {
	DronePortID             string                                   `json:"dronePortId,omitempty"`
	DronePortName           string                                   `json:"dronePortName,omitempty"`
	Address                 string                                   `json:"address,omitempty"`
	Manufacturer            string                                   `json:"manufacturer,omitempty"`
	SerialNumber            string                                   `json:"serialNumber,omitempty"`
	DronePortManufacturerID string                                   `json:"dronePortManufacturerId,omitempty"`
	PortType                *int                                     `json:"portType,omitempty"`
	VisDronePortCompanyID   string                                   `json:"visDronePortCompanyId,omitempty"`
	StoredAircraftID        string                                   `json:"storedAircraftId,omitempty"`
	Lat                     *float64                                 `json:"lat,omitempty"`
	Lon                     *float64                                 `json:"lon,omitempty"`
	Alt                     *float64                                 `json:"alt,omitempty"`
	SupportDroneType        string                                   `json:"supportDroneType,omitempty"`
	ActiveStatus            *int                                     `json:"activeStatus,omitempty"`
	InactiveTimeFrom        string                                   `json:"inactiveTimeFrom,omitempty"`
	InactiveTimeTo          string                                   `json:"inactiveTimeTo,omitempty"`
	ImageData               string                                   `json:"imageData,omitempty"`
	PublicFlag              *bool                                    `json:"publicFlag,omitempty"`
	PriceInfos              []resourcepricemodel.PriceInfoRequestDto `json:"priceInfos,omitempty"`
}

type DronePortInfoRegisterResponseDto struct {
	DronePortID string `json:"dronePortId"`
}

type DronePortInfoListRequestDto struct {
	DronePortName       string   `json:"dronePortName,omitempty"`
	Address             string   `json:"address,omitempty"`
	Manufacturer        string   `json:"manufacturer,omitempty"`
	SerialNumber        string   `json:"serialNumber,omitempty"`
	PortType            string   `json:"portType,omitempty"`
	MinLat              *float64 `json:"minLat,omitempty"`
	MinLon              *float64 `json:"minLon,omitempty"`
	MaxLat              *float64 `json:"maxLat,omitempty"`
	MaxLon              *float64 `json:"maxLon,omitempty"`
	SupportDroneType    string   `json:"supportDroneType,omitempty"`
	ActiveStatus        string   `json:"activeStatus,omitempty"`
	PerPage             string   `json:"perPage,omitempty"`
	Page                string   `json:"page,omitempty"`
	SortOrders          string   `json:"sortOrders,omitempty"`
	SortColumns         string   `json:"sortColumns,omitempty"`
	PublicFlag          string   `json:"publicFlag,omitempty"`
	IsRequiredPriceInfo string   `json:"isRequiredPriceInfo,omitempty"`
}

type DronePortInfoListResponseDto struct {
	Data        []DronePortInfoListResponseElement `json:"data"`
	PerPage     int                                `json:"perPage"`
	CurrentPage int                                `json:"currentPage"`
	LastPage    int                                `json:"lastPage"`
	Total       int                                `json:"total"`
}

type DronePortInfoListResponseElement struct {
	DronePortID             string                                                `json:"dronePortId"`
	DronePortName           string                                                `json:"dronePortName"`
	Address                 string                                                `json:"address"`
	Manufacturer            string                                                `json:"manufacturer"`
	SerialNumber            string                                                `json:"serialNumber"`
	DronePortManufacturerID string                                                `json:"dronePortManufacturerId"`
	PortType                int                                                   `json:"portType"`
	VisDronePortCompanyID   string                                                `json:"visDronePortCompanyId"`
	StoredAircraftID        string                                                `json:"storedAircraftId"`
	Lat                     float64                                               `json:"lat"`
	Lon                     float64                                               `json:"lon"`
	Alt                     float64                                               `json:"alt"`
	SupportDroneType        string                                                `json:"supportDroneType"`
	ActiveStatus            int                                                   `json:"activeStatus"`
	ScheduledStatus         int                                                   `json:"scheduledStatus"`
	InactiveTimeFrom        *time.Time                                            `json:"inactiveTimeFrom"`
	InactiveTimeTo          *time.Time                                            `json:"inactiveTimeTo"`
	OperatorID              string                                                `json:"operatorId"`
	PublicFlag              bool                                                  `json:"publicFlag"`
	UpdateTime              *time.Time                                            `json:"updateTime"`
	PriceInfos              []resourcepricemodel.PriceInfoSearchListDetailElement `json:"priceInfos"`
}

type DronePortInfoDetailResponseDto struct {
	DronePortID           string                                                `json:"dronePortId"`
	DronePortName         string                                                `json:"dronePortName"`
	Address               string                                                `json:"address"`
	Manufacturer          string                                                `json:"manufacturer"`
	SerialNumber          string                                                `json:"serialNumber"`
	PortType              int                                                   `json:"portType"`
	VisDronePortCompanyID string                                                `json:"visDronePortCompanyId"`
	StoredAircraftID      string                                                `json:"storedAircraftId"`
	Lat                   float64                                               `json:"lat"`
	Lon                   float64                                               `json:"lon"`
	Alt                   float64                                               `json:"alt"`
	SupportDroneType      string                                                `json:"supportDroneType"`
	ActiveStatus          int                                                   `json:"activeStatus"`
	ScheduledStatus       int                                                   `json:"scheduledStatus"`
	InactiveTimeFrom      string                                                `json:"inactiveTimeFrom"`
	InactiveTimeTo        string                                                `json:"inactiveTimeTo"`
	OperatorID            string                                                `json:"operatorId"`
	PublicFlag            bool                                                  `json:"publicFlag"`
	UpdateTime            string                                                `json:"updateTime"`
	ImageData             string                                                `json:"imageData"`
	PriceInfos            []resourcepricemodel.PriceInfoSearchListDetailElement `json:"priceInfos"`
}

type DronePortEnvironmentInfoResponseDto struct {
	DronePortID      string  `json:"dronePortId"`
	WindSpeed        float64 `json:"windSpeed"`
	WindDirection    float64 `json:"windDirection"`
	Rainfall         float64 `json:"rainfall"`
	Temp             float64 `json:"temp"`
	Pressure         float64 `json:"pressure"`
	ObstacleDetected bool    `json:"obstacleDetected"`
	ObservationTime  string  `json:"observationTime"`
}

type DronePortReserveInfoRegisterListRequestDto struct {
	Data              []DronePortReserveElement `json:"data"`
	ReserveProviderID string                    `json:"reserveProviderId,omitempty"`
}

type DronePortReserveElement struct {
	GroupReservationID  string `json:"groupReservationId"`
	DronePortID         string `json:"dronePortId"`
	AircraftID          string `json:"aircraftId,omitempty"`
	RouteReservationID  string `json:"routeReservationId,omitempty"`
	UsageType           int    `json:"usageType"`
	ReservationTimeFrom string `json:"reservationTimeFrom"`
	ReservationTimeTo   string `json:"reservationTimeTo"`
}

type DronePortReserveInfoRegisterListResponseDto struct {
	DronePortReservationIDs []string `json:"dronePortReservationIds"`
}

type DronePortReserveInfoUpdateRequestDto struct {
	GroupReservationID     string `json:"groupReservationId,omitempty"`
	DronePortReservationID string `json:"dronePortReservationId"`
	DronePortID            string `json:"dronePortId,omitempty"`
	AircraftID             string `json:"aircraftId,omitempty"`
	RouteReservationID     string `json:"routeReservationId,omitempty"`
	UsageType              *int   `json:"usageType,omitempty"`
	ReservationTimeFrom    string `json:"reservationTimeFrom"`
	ReservationTimeTo      string `json:"reservationTimeTo"`
}

type DronePortReserveInfoUpdateResponseDto struct {
	DronePortReservationID string `json:"dronePortReservationId"`
}

type DronePortReserveInfoListRequestDto struct {
	GroupReservationID string `json:"groupReservationId,omitempty"`
	DronePortID        string `json:"dronePortId,omitempty"`
	DronePortName      string `json:"dronePortName,omitempty"`
	AircraftID         string `json:"aircraftId,omitempty"`
	RouteReservationID string `json:"routeReservationId,omitempty"`
	TimeFrom           string `json:"timeFrom,omitempty"`
	TimeTo             string `json:"timeTo,omitempty"`
	ReserveProviderID  string `json:"reserveProviderId,omitempty"`
	PerPage            string `json:"perPage,omitempty"`
	Page               string `json:"page,omitempty"`
	SortOrders         string `json:"sortOrders,omitempty"`
	SortColumns        string `json:"sortColumns,omitempty"`
}

type DronePortReserveInfoListResponseDto struct {
	Data        []DronePortReserveInfoListElement `json:"data"`
	PerPage     int                               `json:"perPage"`
	CurrentPage int                               `json:"currentPage"`
	LastPage    int                               `json:"lastPage"`
	Total       int                               `json:"total"`
}

type DronePortReserveInfoListElement struct {
	DronePortReservationID string `json:"dronePortReservationId"`
	GroupReservationID     string `json:"groupReservationId"`
	DronePortID            string `json:"dronePortId"`
	AircraftID             string `json:"aircraftId"`
	RouteReservationID     string `json:"routeReservationId"`
	UsageType              int    `json:"usageType"`
	ReservationTimeFrom    string `json:"reservationTimeFrom"`
	ReservationTimeTo      string `json:"reservationTimeTo"`
	DronePortName          string `json:"dronePortName"`
	AircraftName           string `json:"aircraftName"`
	VisDronePortCompanyID  string `json:"visDronePortCompanyId"`
	ReservationActiveFlag  bool   `json:"reservationActiveFlag"`
	InactiveTimeFrom       string `json:"inactiveTimeFrom"`
	InactiveTimeTo         string `json:"inactiveTimeTo"`
	ReserveProviderID      string `json:"reserveProviderId"`
	OperatorID             string `json:"operatorId"`
}

type DronePortReserveInfoDetailResponseDto struct {
	DronePortReservationID string `json:"dronePortReservationId"`
	GroupReservationID     string `json:"groupReservationId"`
	DronePortID            string `json:"dronePortId"`
	AircraftID             string `json:"aircraftId"`
	RouteReservationID     string `json:"routeReservationId"`
	UsageType              int    `json:"usageType"`
	ReservationTimeFrom    string `json:"reservationTimeFrom"`
	ReservationTimeTo      string `json:"reservationTimeTo"`
	DronePortName          string `json:"dronePortName"`
	AircraftName           string `json:"aircraftName"`
	VisDronePortCompanyID  string `json:"visDronePortCompanyId"`
	ReservationActiveFlag  bool   `json:"reservationActiveFlag"`
	ReserveProviderID      string `json:"reserveProviderId"`
	OperatorID             string `json:"operatorId"`
}
