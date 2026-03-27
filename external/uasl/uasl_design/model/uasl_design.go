package model

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type UaslDesignTopEntity struct {
	Uasl []UaslInfoEntity `json:"uasl"`
}

type UaslInfoEntity struct {
	UaslAdministratorID string       `json:"uaslAdministratorId"`
	BusinessNumber      string       `json:"businessNumber"`
	Uasl                UaslEntities `json:"uasl"`
}

type UaslEntities []UaslEntity

func (u *UaslEntities) UnmarshalJSON(data []byte) error {

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*u = []UaslEntity{}
		return nil
	}

	var arr []UaslEntity
	if err := json.Unmarshal(data, &arr); err == nil {
		*u = arr
		return nil
	}

	var single UaslEntity
	if err := json.Unmarshal(data, &single); err == nil {
		*u = []UaslEntity{single}
		return nil
	}

	return fmt.Errorf("UaslEntities: unable to unmarshal data into either array or object")
}

func (u UaslEntities) MarshalJSON() ([]byte, error) {
	return json.Marshal([]UaslEntity(u))
}

type UaslEntity struct {
	UaslID        string               `json:"uaslId"`
	UaslName      string               `json:"uaslName"`
	FlightPurpose string               `json:"flightPurpose"`
	DroneList     []int                `json:"droneList,omitempty"`
	UaslPoints    []UaslPointEntity    `json:"uaslPoints"`
	UaslSections  []UaslSectionsEntity `json:"uaslSections"`
	CreatedAt     string               `json:"createdAt"`
	UpdatedAt     string               `json:"updatedAt"`
}

type UaslPointEntity struct {
	UaslPointID        string                 `json:"uaslPointId"`
	Geometry           map[string]interface{} `json:"geometry"`
	DeviationGeometry  map[string]interface{} `json:"deviationGeometry,omitempty"`
	ExternalGuarantee  bool                   `json:"externalGuarantee,omitempty"`
	ExternalSystemInfo *ExternalSystemInfo    `json:"externalSystemInfo,omitempty"`
	UaslPointName      string                 `json:"uaslPointName,omitempty"`
}

type ExternalSystemInfo struct {
	SystemID    *string `json:"systemId"`
	UaslID      *string `json:"uaslId"`
	UaslPointID *string `json:"uaslPointId"`
}

type UaslSectionsEntity struct {
	UaslSectionID   string   `json:"uaslSectionId"`
	UaslSectionName string   `json:"uaslSectionName"`
	UaslPointIds    []string `json:"uaslPointIds"`
	DroneportIds    []string `json:"droneportIds"`
}
