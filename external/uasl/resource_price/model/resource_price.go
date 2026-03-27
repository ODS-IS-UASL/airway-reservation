package model

type ResourcePriceListRequest struct {
	ResourceID         string `json:"resourceId"`
	ResourceType       *int   `json:"resourceType,omitempty"`
	PriceType          *int   `json:"priceType,omitempty"`
	EffectiveStartTime string `json:"effectiveStartTime,omitempty"`
	EffectiveEndTime   string `json:"effectiveEndTime,omitempty"`
}

type PriceInfoDetail struct {
	PriceID                string `json:"priceId"`
	PrimaryRouteOperatorID string `json:"primaryRouteOperatorId"`
	PriceType              int    `json:"priceType"`
	PricePerUnit           int    `json:"pricePerUnit"`
	Price                  int    `json:"price"`
	EffectiveStartTime     string `json:"effectiveStartTime"`
	EffectiveEndTime       string `json:"effectiveEndTime"`
	Priority               int    `json:"priority"`
	OperatorID             string `json:"operatorId"`
}

type ResourcePriceElement struct {
	ResourceID   string            `json:"resourceId"`
	ResourceType int               `json:"resourceType"`
	PriceInfos   []PriceInfoDetail `json:"priceInfos"`
}

type ResourcePriceListResponse struct {
	Resources []ResourcePriceElement `json:"resources"`
}

type PriceInfoSearchListDetailElement struct {
	PriceID                string `json:"priceId"`
	PrimaryRouteOperatorID string `json:"primaryRouteOperatorId"`
	PriceType              int    `json:"priceType"`
	PricePerUnit           int    `json:"pricePerUnit"`
	Price                  int    `json:"price"`
	EffectiveStartTime     string `json:"effectiveStartTime"`
	EffectiveEndTime       string `json:"effectiveEndTime"`
	Priority               int    `json:"priority"`
	OperatorID             string `json:"operatorId"`
}

type PriceInfoRequestDto struct {
	ProcessingType     int    `json:"processingType"`
	PriceID            string `json:"priceId,omitempty"`
	PriceType          *int   `json:"priceType,omitempty"`
	PricePerUnit       *int   `json:"pricePerUnit,omitempty"`
	Price              *int   `json:"price,omitempty"`
	EffectiveStartTime string `json:"effectiveStartTime,omitempty"`
	EffectiveEndTime   string `json:"effectiveEndTime,omitempty"`
	Priority           *int   `json:"priority,omitempty"`
}
