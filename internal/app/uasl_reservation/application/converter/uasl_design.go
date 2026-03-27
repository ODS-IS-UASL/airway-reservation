package converter

import (
	"encoding/json"
	"fmt"
	"strings"

	conformityAssessmentModel "uasl-reservation/external/uasl/uasl_design/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/pkg/value"
)

func ToUaslBulkDataFromDTO(topEntity conformityAssessmentModel.UaslDesignTopEntity, isInternal bool, baseURL string) *model.UaslBulkData {
	bulk := &model.UaslBulkData{}
	for _, uaslInfo := range topEntity.Uasl {
		var uaslIDs []string
		for _, uasl := range uaslInfo.Uasl {
			if uasl.UaslID != "" {
				uaslIDs = append(uaslIDs, uasl.UaslID)
			}
		}
		var externalServices value.NullJSON
		if len(uaslIDs) > 0 && baseURL != "" {
			services := make(model.ExternalServicesList, 0, len(uaslIDs))
			for _, id := range uaslIDs {
				services = append(services, model.ExternalService{ExUaslID: id, Services: model.ExternalServiceEndpoints{BaseURL: baseURL}})
			}
			if raw, err := json.Marshal(services); err == nil {
				msg := json.RawMessage(raw)
				externalServices = value.NewNullJSON(&msg, true)
			}
		}
		bulk.Administrators = append(bulk.Administrators, model.UaslAdministrator{ExAdministratorID: uaslInfo.UaslAdministratorID, BusinessNumber: value.NewNullString(&uaslInfo.BusinessNumber, uaslInfo.BusinessNumber != ""), IsInternal: isInternal, ExternalServices: externalServices})
		for _, uasl := range uaslInfo.Uasl {
			pointMap := map[string]map[string]interface{}{}
			for _, point := range uasl.UaslPoints {
				pointMap[point.UaslPointID] = point.Geometry
			}
			for _, section := range uasl.UaslSections {
				bulk.Definitions = append(bulk.Definitions, model.ExternalUaslDefinition{ExUaslSectionID: section.UaslSectionID, ExUaslID: value.NewNullString(&uasl.UaslID, uasl.UaslID != ""), ExAdministratorID: uaslInfo.UaslAdministratorID, Geometry: model.NewPostGISGeometry(toSectionEWKT(section.UaslPointIds, pointMap)), PointIDs: section.UaslPointIds, FlightPurpose: value.NewNullString(&uasl.FlightPurpose, uasl.FlightPurpose != ""), PriceInfo: model.PriceInfoList{}, PriceTimezone: "UTC", PriceVersion: 1, Status: model.UaslSectionStatusAvailable})
			}
		}
	}
	return bulk
}

func toSectionEWKT(pointIDs []string, pointGeometryMap map[string]map[string]interface{}) []byte {
	for _, pointID := range pointIDs {
		if ewkt := geoJSONToEWKT(pointGeometryMap[pointID]); ewkt != "" {
			return []byte(ewkt)
		}
	}
	return nil
}
func geoJSONToEWKT(geometry map[string]interface{}) string {
	if len(geometry) == 0 {
		return ""
	}
	typeName, _ := geometry["type"].(string)
	switch strings.ToUpper(typeName) {
	case "POLYGON":
		if rings, ok := geometry["coordinates"].([]interface{}); ok && len(rings) > 0 {
			return "SRID=4326;" + polygonRingToWKT(rings[0])
		}
	case "MULTIPOLYGON":
		if polys, ok := geometry["coordinates"].([]interface{}); ok && len(polys) > 0 {
			if rings, ok := polys[0].([]interface{}); ok && len(rings) > 0 {
				return "SRID=4326;" + polygonRingToWKT(rings[0])
			}
		}
	}
	return ""
}
func polygonRingToWKT(raw interface{}) string {
	coords, ok := raw.([]interface{})
	if !ok || len(coords) == 0 {
		return ""
	}
	points := []string{}
	for _, coord := range coords {
		pair, ok := coord.([]interface{})
		if !ok || len(pair) < 2 {
			continue
		}
		lng, ok1 := pair[0].(float64)
		lat, ok2 := pair[1].(float64)
		if !ok1 || !ok2 {
			continue
		}
		points = append(points, fmt.Sprintf("%f %f", lng, lat))
	}
	if len(points) < 3 {
		return ""
	}
	if points[0] != points[len(points)-1] {
		points = append(points, points[0])
	}
	return "POLYGON((" + strings.Join(points, ",") + "))"
}
