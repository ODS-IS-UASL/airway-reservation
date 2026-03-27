package model

import resourcepricemodel "uasl-reservation/external/uasl/resource_price/model"

type AircraftInfoRequestDto struct {
	AircraftID           string                                   `json:"aircraftId,omitempty"`
	AircraftName         string                                   `json:"aircraftName,omitempty"`
	Manufacturer         string                                   `json:"manufacturer,omitempty"`
	ModelNumber          string                                   `json:"modelNumber,omitempty"`
	ModelName            string                                   `json:"modelName,omitempty"`
	ManufacturingNumber  string                                   `json:"manufacturingNumber,omitempty"`
	AircraftType         *int                                     `json:"aircraftType,omitempty"`
	MaxTakeoffWeight     *float64                                 `json:"maxTakeoffWeight,omitempty"`
	BodyWeight           *float64                                 `json:"bodyWeight,omitempty"`
	MaxFlightSpeed       *float64                                 `json:"maxFlightSpeed,omitempty"`
	MaxFlightTime        *float64                                 `json:"maxFlightTime,omitempty"`
	Lat                  *float64                                 `json:"lat,omitempty"`
	Lon                  *float64                                 `json:"lon,omitempty"`
	Certification        *bool                                    `json:"certification,omitempty"`
	DipsRegistrationCode string                                   `json:"dipsRegistrationCode,omitempty"`
	OwnerType            *int                                     `json:"ownerType,omitempty"`
	OwnerID              string                                   `json:"ownerId,omitempty"`
	ImageData            string                                   `json:"imageData,omitempty"`
	PublicFlag           *bool                                    `json:"publicFlag,omitempty"`
	FileInfos            []AircraftInfoFileInfoListElementReq     `json:"fileInfos,omitempty"`
	PayloadInfos         []PayloadInfoListElementReq              `json:"payloadInfos,omitempty"`
	PriceInfos           []resourcepricemodel.PriceInfoRequestDto `json:"priceInfos,omitempty"`
}

type AircraftInfoFileInfoListElementReq struct {
	ProcessingType   int    `json:"processingType"`
	FileID           string `json:"fileId,omitempty"`
	FileLogicalName  string `json:"fileLogicalName,omitempty"`
	FilePhysicalName string `json:"filePhysicalName,omitempty"`
	FileData         string `json:"fileData,omitempty"`
}

type PayloadInfoListElementReq struct {
	ProcessingType    int    `json:"processingType"`
	PayloadID         string `json:"payloadId,omitempty"`
	PayloadName       string `json:"payloadName,omitempty"`
	PayloadDetailText string `json:"payloadDetailText,omitempty"`
	ImageData         string `json:"imageData,omitempty"`
	FilePhysicalName  string `json:"filePhysicalName,omitempty"`
	FileData          string `json:"fileData,omitempty"`
}

type AircraftInfoResponseDto struct {
	AircraftID string `json:"aircraftId"`
}

type AircraftInfoSearchListRequestDto struct {
	AircraftName          string   `json:"aircraftName,omitempty"`
	Manufacturer          string   `json:"manufacturer,omitempty"`
	ModelNumber           string   `json:"modelNumber,omitempty"`
	ModelName             string   `json:"modelName,omitempty"`
	ManufacturingNumber   string   `json:"manufacturingNumber,omitempty"`
	AircraftType          string   `json:"aircraftType,omitempty"`
	Certification         string   `json:"certification,omitempty"`
	DipsRegistrationCode  string   `json:"dipsRegistrationCode,omitempty"`
	OwnerType             string   `json:"ownerType,omitempty"`
	OwnerID               string   `json:"ownerId,omitempty"`
	MinLat                *float64 `json:"minLat,omitempty"`
	MinLon                *float64 `json:"minLon,omitempty"`
	MaxLat                *float64 `json:"maxLat,omitempty"`
	MaxLon                *float64 `json:"maxLon,omitempty"`
	PerPage               string   `json:"perPage,omitempty"`
	Page                  string   `json:"page,omitempty"`
	SortOrders            string   `json:"sortOrders,omitempty"`
	SortColumns           string   `json:"sortColumns,omitempty"`
	PublicFlag            string   `json:"publicFlag,omitempty"`
	ModelSearchSwitchFlag string   `json:"modelSearchSwitchFlag,omitempty"`
	IsRequiredPayloadInfo string   `json:"isRequiredPayloadInfo,omitempty"`
	IsRequiredPriceInfo   string   `json:"isRequiredPriceInfo,omitempty"`
}

type AircraftInfoSearchListResponseDto struct {
	Data        []AircraftInfoSearchListElement `json:"data"`
	PerPage     int                             `json:"perPage"`
	CurrentPage int                             `json:"currentPage"`
	LastPage    int                             `json:"lastPage"`
	Total       int                             `json:"total"`
}

type AircraftInfoSearchListElement struct {
	AircraftID           string                                                `json:"aircraftId"`
	AircraftName         string                                                `json:"aircraftName"`
	Manufacturer         string                                                `json:"manufacturer"`
	ModelNumber          string                                                `json:"modelNumber"`
	ModelName            string                                                `json:"modelName"`
	ManufacturingNumber  string                                                `json:"manufacturingNumber"`
	AircraftType         int                                                   `json:"aircraftType"`
	MaxTakeoffWeight     float64                                               `json:"maxTakeoffWeight"`
	BodyWeight           float64                                               `json:"bodyWeight"`
	MaxFlightSpeed       float64                                               `json:"maxFlightSpeed"`
	MaxFlightTime        float64                                               `json:"maxFlightTime"`
	Certification        bool                                                  `json:"certification"`
	Lat                  float64                                               `json:"lat"`
	Lon                  float64                                               `json:"lon"`
	DipsRegistrationCode string                                                `json:"dipsRegistrationCode"`
	OwnerType            int                                                   `json:"ownerType"`
	OwnerID              string                                                `json:"ownerId"`
	PublicFlag           bool                                                  `json:"publicFlag"`
	OperatorID           string                                                `json:"operatorId"`
	PayloadInfos         []PayloadInfoSearchListElement                        `json:"payloadInfos"`
	PriceInfos           []resourcepricemodel.PriceInfoSearchListDetailElement `json:"priceInfos"`
}

type PayloadInfoSearchListElement struct {
	PayloadID         string `json:"payloadId"`
	PayloadName       string `json:"payloadName"`
	PayloadDetailText string `json:"payloadDetailText"`
	FilePhysicalName  string `json:"filePhysicalName"`
	OperatorID        string `json:"operatorId"`
}

type AircraftInfoDetailResponseDto struct {
	AircraftID           string                                                `json:"aircraftId"`
	AircraftName         string                                                `json:"aircraftName"`
	Manufacturer         string                                                `json:"manufacturer"`
	ModelNumber          string                                                `json:"modelNumber"`
	ModelName            string                                                `json:"modelName"`
	ManufacturingNumber  string                                                `json:"manufacturingNumber"`
	AircraftType         int                                                   `json:"aircraftType"`
	MaxTakeoffWeight     float64                                               `json:"maxTakeoffWeight"`
	BodyWeight           float64                                               `json:"bodyWeight"`
	MaxFlightSpeed       float64                                               `json:"maxFlightSpeed"`
	MaxFlightTime        float64                                               `json:"maxFlightTime"`
	Certification        bool                                                  `json:"certification"`
	Lat                  float64                                               `json:"lat"`
	Lon                  float64                                               `json:"lon"`
	DipsRegistrationCode string                                                `json:"dipsRegistrationCode"`
	OwnerType            int                                                   `json:"ownerType"`
	OwnerID              string                                                `json:"ownerId"`
	ImageData            string                                                `json:"imageData"`
	PublicFlag           bool                                                  `json:"publicFlag"`
	OperatorID           string                                                `json:"operatorId"`
	FileInfos            []AircraftInfoFileInfoListElementRes                  `json:"fileInfos"`
	PayloadInfos         []PayloadInfoDetailElement                            `json:"payloadInfos"`
	PriceInfos           []resourcepricemodel.PriceInfoSearchListDetailElement `json:"priceInfos"`
}

type AircraftInfoFileInfoListElementRes struct {
	FileLogicalName  string `json:"fileLogicalName"`
	FilePhysicalName string `json:"filePhysicalName"`
	FileID           string `json:"fileId"`
}

type PayloadInfoDetailElement struct {
	PayloadID         string `json:"payloadId"`
	PayloadName       string `json:"payloadName"`
	PayloadDetailText string `json:"payloadDetailText"`
	ImageData         string `json:"imageData"`
	FilePhysicalName  string `json:"filePhysicalName"`
	OperatorID        string `json:"operatorId"`
}

type AircraftReserveInfoRequestDto struct {
	AircraftReservationID string `json:"aircraftReservationId,omitempty"`
	GroupReservationID    string `json:"groupReservationId,omitempty"`
	AircraftID            string `json:"aircraftId,omitempty"`
	ReservationTimeFrom   string `json:"reservationTimeFrom,omitempty"`
	ReservationTimeTo     string `json:"reservationTimeTo,omitempty"`
}

type AircraftReserveInfoResponseDto struct {
	AircraftReservationID string `json:"aircraftReservationId"`
}

type AircraftReserveInfoListRequestDto struct {
	GroupReservationID string `json:"groupReservationId,omitempty"`
	AircraftID         string `json:"aircraftId,omitempty"`
	AircraftName       string `json:"aircraftName,omitempty"`
	TimeFrom           string `json:"timeFrom,omitempty"`
	TimeTo             string `json:"timeTo,omitempty"`
	ReserveProviderID  string `json:"reserveProviderId,omitempty"`
	PerPage            string `json:"perPage,omitempty"`
	Page               string `json:"page,omitempty"`
	SortOrders         string `json:"sortOrders,omitempty"`
	SortColumns        string `json:"sortColumns,omitempty"`
}

type AircraftReserveInfoListResponseDto struct {
	Data        []AircraftReserveInfoDetailResponseDto `json:"data"`
	PerPage     int                                    `json:"perPage"`
	CurrentPage int                                    `json:"currentPage"`
	LastPage    int                                    `json:"lastPage"`
	Total       int                                    `json:"total"`
}

type AircraftReserveInfoDetailResponseDto struct {
	AircraftReservationID string `json:"aircraftReservationId"`
	GroupReservationID    string `json:"groupReservationId"`
	AircraftID            string `json:"aircraftId"`
	ReservationTimeFrom   string `json:"reservationTimeFrom"`
	ReservationTimeTo     string `json:"reservationTimeTo"`
	AircraftName          string `json:"aircraftName"`
	ReserveProviderID     string `json:"reserveProviderId"`
	OperatorID            string `json:"operatorId"`
}
