package converter

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	dipsmodel "uasl-reservation/external/uasl/dips/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

const (
	DIPS_FPR_TIME_FORMAT             = "20060102 1504"
	DIPS_FLIGHT_SEARCH_TYPE_ALL_USER = "0"
)

type FlightPlanInfoConverter struct{}

func NewFlightPlanInfoConverter() *FlightPlanInfoConverter { return &FlightPlanInfoConverter{} }

func (c *FlightPlanInfoConverter) ToExternalRequest(req model.FlightPlanInfoRequest) dipsmodel.FlightPlanInfoRequest {
	allFlightPlan := DIPS_FLIGHT_SEARCH_TYPE_ALL_USER
	if req.AllFlightPlan != "" {
		allFlightPlan = req.AllFlightPlan
	}
	coordinates := req.Features.Coordinates
	if req.Features.Type == "Polygon" {
		coordinates = normalizePolygonCoordinates(req.Features.Coordinates)
	}
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	return dipsmodel.FlightPlanInfoRequest{
		Features:      dipsmodel.Geometry{Type: req.Features.Type, Center: req.Features.Center, Radius: req.Features.Radius, Coordinates: coordinates},
		AllFlightPlan: allFlightPlan,
		StartTime:     req.StartTime.In(jst).Format(DIPS_FPR_TIME_FORMAT),
		FinishTime:    req.FinishTime.In(jst).Format(DIPS_FPR_TIME_FORMAT),
	}
}

func normalizePolygonCoordinates(coords [][]float64) [][]float64 {
	if len(coords) == 0 {
		return coords
	}
	seen := map[string]struct{}{}
	unique := make([][]float64, 0, len(coords))
	for _, c := range coords {
		if len(c) < 2 {
			continue
		}
		key := fmt.Sprintf("%g,%g", c[0], c[1])
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, []float64{c[0], c[1]})
	}
	if len(unique) < 3 {
		return coords
	}
	if unique[0][0] != unique[len(unique)-1][0] || unique[0][1] != unique[len(unique)-1][1] {
		unique = append(unique, []float64{unique[0][0], unique[0][1]})
	}
	return unique
}

func (c *FlightPlanInfoConverter) ToDomainResponse(resp dipsmodel.FlightPlanInfoResponse) model.FlightPlanInfoResponse {
	items := make([]model.FlightPlanItem, 0, len(resp.FlightPlanInfo))
	for _, fp := range resp.FlightPlanInfo {
		items = append(items, model.FlightPlanItem{FlightPlanID: fp.FlightPlanID, StartTime: fp.StartTime, FinishTime: fp.FinishTime, FlyRoute: model.FlyRouteData{Type: fp.FlyRoute.Type, Center: fp.FlyRoute.Center, Radius: fp.FlyRoute.Radius, Coordinates: fp.FlyRoute.Coordinates}})
	}
	return model.FlightPlanInfoResponse{TotalCount: resp.TotalCount, FlightPlanList: items}
}

func (c *FlightPlanInfoConverter) ToFlightPlanRequestsFromDefinitions(definitions []*model.ExternalUaslDefinition) ([]model.GeometryData, error) {
	if len(definitions) == 0 {
		return nil, fmt.Errorf("no definitions provided")
	}

	geometries := make([]model.GeometryData, 0, len(definitions))
	for _, def := range definitions {
		if def == nil || def.Geometry.IsEmpty() {
			continue
		}
		geometry, err := ewktToGeometryData(def.Geometry.ToString())
		if err != nil {
			return nil, err
		}
		geometries = append(geometries, geometry)
	}
	return geometries, nil
}

func ewktToGeometryData(ewkt string) (model.GeometryData, error) {
	if idx := strings.Index(ewkt, ";"); idx >= 0 {
		ewkt = ewkt[idx+1:]
	}
	upper := strings.ToUpper(strings.TrimSpace(ewkt))
	switch {
	case strings.HasPrefix(upper, "POLYGON(("):
		body := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(ewkt), "POLYGON(("), "))")
		coords, err := parseWKTPoints(body)
		if err != nil {
			return model.GeometryData{}, err
		}
		return model.GeometryData{Type: "Polygon", Coordinates: coords}, nil
	case strings.HasPrefix(upper, "POINT("):
		body := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(ewkt), "POINT("), ")")
		coords, err := parseWKTPoint(body)
		if err != nil {
			return model.GeometryData{}, err
		}
		return model.GeometryData{Type: "Circle", Center: coords, Radius: 10000}, nil
	default:
		return model.GeometryData{}, fmt.Errorf("unsupported geometry: %s", ewkt)
	}
}

func parseWKTPoints(body string) ([][]float64, error) {
	parts := strings.Split(body, ",")
	coords := make([][]float64, 0, len(parts))
	for _, part := range parts {
		point, err := parseWKTPoint(part)
		if err != nil {
			return nil, err
		}
		coords = append(coords, point)
	}
	return coords, nil
}

func parseWKTPoint(body string) ([]float64, error) {
	fields := strings.Fields(strings.TrimSpace(body))
	if len(fields) < 2 {
		return nil, fmt.Errorf("invalid point: %s", body)
	}
	lng, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return nil, err
	}
	lat, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return nil, err
	}
	return []float64{lng, lat}, nil
}

func ExtractCentroidFromWKB(geometryBytes []byte, geometryString string) (lat, lng float64, ok bool) {
	raw := strings.TrimSpace(geometryString)
	if raw == "" {
		raw = strings.TrimSpace(string(geometryBytes))
	}
	if raw == "" {
		return 0, 0, false
	}
	geometry, err := ewktToGeometryData(raw)
	if err != nil {
		return 0, 0, false
	}
	if geometry.Type == "Circle" && len(geometry.Center) >= 2 {
		return geometry.Center[1], geometry.Center[0], true
	}
	if geometry.Type == "Polygon" && len(geometry.Coordinates) > 0 {
		var sumLng, sumLat float64
		var count float64
		for _, point := range geometry.Coordinates {
			if len(point) < 2 {
				continue
			}
			sumLng += point[0]
			sumLat += point[1]
			count++
		}
		if count > 0 && !math.IsNaN(sumLat) && !math.IsNaN(sumLng) {
			return sumLat / count, sumLng / count, true
		}
	}
	return 0, 0, false
}
