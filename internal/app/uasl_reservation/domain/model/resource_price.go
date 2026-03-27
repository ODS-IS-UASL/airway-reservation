package model

import (
	"uasl-reservation/internal/pkg/value"
)

type ResourcePriceInfo struct {
	PriceID            string         `json:"price_id"`
	PriceType          int            `json:"price_type"`
	PricePerUnit       int            `json:"price_per_unit"`
	Price              int            `json:"price"`
	EffectiveStartTime value.NullTime `json:"effective_start_time"`
	EffectiveEndTime   value.NullTime `json:"effective_end_time"`
	Priority           int            `json:"priority"`
	OperatorID         value.ModelID  `json:"operator_id"`
}

type ResourcePriceListRequest struct {
	ResourceIDs        []string       `json:"resource_ids,omitempty"`
	ResourceType       *int           `json:"resource_type,omitempty"`
	PriceType          *int           `json:"price_type,omitempty"`
	EffectiveStartTime value.NullTime `json:"effective_start_time,omitempty"`
	EffectiveEndTime   value.NullTime `json:"effective_end_time,omitempty"`
}

type ResourcePrice struct {
	ResourceID   string              `json:"resource_id"`
	ResourceType int                 `json:"resource_type"`
	PriceInfos   []ResourcePriceInfo `json:"price_infos"`
}

type ResourcePriceList struct {
	Resources []ResourcePrice `json:"resources"`
}

type ResourcePriceResult struct {
	VehiclePrices map[string]int32 `json:"vehicle_prices"`
	PortPrices    map[string]int32 `json:"port_prices"`
	TotalAmount   int32            `json:"total_amount"`
}
