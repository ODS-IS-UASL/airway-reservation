package model

type UaslDesignTopEntity struct {
	Uasl []UaslInfoEntity `json:"uasl"`
}

type UaslInfoEntity struct {
	UaslAdministratorID string       `json:"uaslAdministratorId"`
	BusinessNumber      string       `json:"businessNumber"`
	Uasl                []UaslEntity `json:"uasl"`
}

type UaslEntity struct {
	UaslID        string               `json:"uaslId"`
	UaslName      string               `json:"uaslName"`
	FlightPurpose string               `json:"flightPurpose"`
	UaslPoints    []UaslPointEntity    `json:"uaslPoints"`
	UaslSections  []UaslSectionsEntity `json:"uaslSections"`
	CreatedAt     string               `json:"createdAt"`
	UpdatedAt     string               `json:"updatedAt"`
}

type UaslPointEntity struct {
	UaslPointID string                 `json:"uaslPointId"`
	Geometry    map[string]interface{} `json:"geometry"`
}

type UaslSectionsEntity struct {
	UaslSectionID   string   `json:"uaslSectionId"`
	UaslSectionName string   `json:"uaslSectionName"`
	UaslPointIds    []string `json:"uaslPointIds"`
	DroneportIds    []string `json:"droneportIds"`
}

type UaslBulkData struct {
	Administrators []UaslAdministrator
	Definitions    []ExternalUaslDefinition
}
