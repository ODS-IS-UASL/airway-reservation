package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"uasl-reservation/internal/app/uasl_reservation/application/converter"
	gatewayIF "uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/repositoryIF"
	"uasl-reservation/internal/pkg/logger"
)

const flightPlanConflictCheckTimeout = 5 * time.Second

type AirspaceConflictService struct {
	flightPlanGW           gatewayIF.FlightPlanInfoOpenAPIGatewayIF
	externalUaslDefRepo    repositoryIF.ExternalUaslDefinitionRepositoryIF
	uaslReservationRepo    repositoryIF.UaslReservationRepositoryIF
	converter              *converter.FlightPlanInfoConverter
	conformityAssessmentGW gatewayIF.ConformityAssessmentGatewayIF
}

func NewAirspaceConflictService(
	flightPlanGW gatewayIF.FlightPlanInfoOpenAPIGatewayIF,
	externalUaslDefRepo repositoryIF.ExternalUaslDefinitionRepositoryIF,
	uaslReservationRepo repositoryIF.UaslReservationRepositoryIF,
	conformityAssessmentGW gatewayIF.ConformityAssessmentGatewayIF,
) *AirspaceConflictService {
	return &AirspaceConflictService{
		flightPlanGW:           flightPlanGW,
		externalUaslDefRepo:    externalUaslDefRepo,
		uaslReservationRepo:    uaslReservationRepo,
		conformityAssessmentGW: conformityAssessmentGW,
		converter:              converter.NewFlightPlanInfoConverter(),
	}
}

func (s *AirspaceConflictService) CheckAirspaceConflict(
	ctx context.Context,
	uaslRequests []model.UaslCheckRequest,
) (*model.AirspaceConflictResult, error) {

	timeoutCtx, cancel := context.WithTimeout(ctx, flightPlanConflictCheckTimeout)
	defer cancel()

	if s.flightPlanGW == nil {
		return nil, nil
	}

	if len(uaslRequests) == 0 {
		return nil, nil
	}

	var overallStartTime, overallEndTime = uaslRequests[0].TimeRange.Start, uaslRequests[0].TimeRange.End
	for _, ar := range uaslRequests {
		if ar.TimeRange.Start.Before(overallStartTime) {
			overallStartTime = ar.TimeRange.Start
		}
		if ar.TimeRange.End.After(overallEndTime) {
			overallEndTime = ar.TimeRange.End
		}
	}

	overallTimeRange := model.TimeRange{Start: overallStartTime, End: overallEndTime}
	if err := overallTimeRange.Validate(); err != nil {
		return nil, fmt.Errorf("invalid overall uasl time range: %w", err)
	}

	sectionIDs := make([]string, 0, len(uaslRequests))
	for _, req := range uaslRequests {
		sectionIDs = append(sectionIDs, string(req.ExUaslSectionID))
	}

	definitions, err := s.externalUaslDefRepo.FindByExSectionIDs(ctx, sectionIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to find uasl definitions: %w", err)
	}

	if len(definitions) == 0 {
		return nil, fmt.Errorf("no uasl definitions found for sections: %v", sectionIDs)
	}

	allGeometries, err := s.converter.ToFlightPlanRequestsFromDefinitions(definitions)
	if err != nil {
		return nil, fmt.Errorf("failed to convert definitions to geometries: %w", err)
	}

	mergedGeometry, err := s.mergeGeometries(allGeometries)
	if err != nil {
		return nil, fmt.Errorf("failed to merge geometries: %w", err)
	}

	flightPlanReq := model.FlightPlanInfoRequest{
		Features:   mergedGeometry,
		StartTime:  overallStartTime,
		FinishTime: overallEndTime,
	}

	allUsersReq := flightPlanReq
	allUsersReq.AllFlightPlan = "0"

	allUsersResp, err := s.flightPlanGW.Fetch(timeoutCtx, allUsersReq)
	if err != nil {
		return nil, fmt.Errorf("DIPS API call failed (all users): %w", err)
	}

	logger.LogInfo("AirspaceConflict: All users flight plans - totalCount=%d", allUsersResp.TotalCount)

	if allUsersResp.TotalCount == 0 {
		logger.LogInfo("AirspaceConflict: No conflicts found (all users: 0)")
		return model.NewAirspaceConflictResult(false, []string{}), nil
	}

	ownAccountReq := flightPlanReq
	ownAccountReq.AllFlightPlan = "1"

	ownAccountResp, err := s.flightPlanGW.Fetch(timeoutCtx, ownAccountReq)
	if err != nil {
		return nil, fmt.Errorf("DIPS API call failed (own account): %w", err)
	}

	logger.LogInfo("AirspaceConflict: Own account flight plans - totalCount=%d", ownAccountResp.TotalCount)

	ownIDs := make(map[string]struct{}, len(ownAccountResp.FlightPlanList))
	for _, fp := range ownAccountResp.FlightPlanList {
		ownIDs[fp.FlightPlanID] = struct{}{}
	}

	conflictedFlightPlanIds := make([]string, 0)
	if allUsersResp.TotalCount > 0 {
		for _, fp := range allUsersResp.FlightPlanList {
			if _, ok := ownIDs[fp.FlightPlanID]; ok {

				continue
			}
			conflictedFlightPlanIds = append(conflictedFlightPlanIds, fp.FlightPlanID)
		}
	}

	otherOrgConflicts := len(conflictedFlightPlanIds)

	logger.LogInfo("AirspaceConflict: Conflict calculation - all=%d, own=%d, other=%d",
		allUsersResp.TotalCount,
		ownAccountResp.TotalCount,
		otherOrgConflicts,
	)

	hasConflict := otherOrgConflicts > 0

	conflictResult := model.NewAirspaceConflictResult(hasConflict, conflictedFlightPlanIds)

	if hasConflict {
		logger.LogInfo("AirspaceConflict: Conflict detected with other organizations' flight plans - other_orgs_count=%d, all=%d, own=%d",
			otherOrgConflicts,
			allUsersResp.TotalCount,
			ownAccountResp.TotalCount,
		)

	} else if ownAccountResp.TotalCount > 0 {
		logger.LogInfo("AirspaceConflict: Own organization's flight plans found, but no conflicts with other organizations - own_count=%d",
			ownAccountResp.TotalCount,
		)
	} else {
		logger.LogInfo("AirspaceConflict: No conflicts found - checked_uasl_sections=%d", len(uaslRequests))
	}

	return conflictResult, nil
}

func (s *AirspaceConflictService) CheckAirspaceConflictWithPolicy(
	ctx context.Context,
	uaslRequests []model.UaslCheckRequest,
	ignoreFlightPlanConflict bool,
) (*model.AirspaceConflictResult, bool, error) {

	conflictResult, err := s.CheckAirspaceConflict(ctx, uaslRequests)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			logger.LogInfo("airspace conflict check skipped due to timeout/cancel; reservation will proceed",
				"timeout_seconds", int(flightPlanConflictCheckTimeout.Seconds()),
				"error", err)
		} else {
			logger.LogError("airspace conflict check failed; skipping flight plan check and proceeding reservation",
				"error", err)
		}
		return nil, true, nil
	}

	if conflictResult == nil || !conflictResult.HasConflict {
		return conflictResult, true, nil
	}

	conflictCount := len(conflictResult.ConflictedFlightPlanIDs)

	if !ignoreFlightPlanConflict {

		logger.LogInfo("Airspace conflict detected: reservation not executed (ignoreFlightPlanConflict=false)",
			"conflict_count", conflictCount,
			"conflicted_flight_plans", conflictResult.ConflictedFlightPlanIDs)
		return conflictResult, false, nil
	}

	logger.LogInfo("Airspace conflict detected: proceeding with reservation (ignoreFlightPlanConflict=true). User can cancel based on conflict information",
		"conflict_count", conflictCount,
		"conflicted_flight_plans", conflictResult.ConflictedFlightPlanIDs)
	return conflictResult, true, nil
}

func (s *AirspaceConflictService) mergeGeometries(geometries []model.GeometryData) (model.GeometryData, error) {
	if len(geometries) == 0 {
		return model.GeometryData{}, fmt.Errorf("no geometries to merge")
	}
	if len(geometries) == 1 {
		return geometries[0], nil
	}

	hasCircle := false
	hasPolygon := false
	for _, g := range geometries {
		if g.Type == "Circle" {
			hasCircle = true
		} else if g.Type == "Polygon" {
			hasPolygon = true
		}
	}

	if hasCircle && !hasPolygon {
		return s.mergeCircles(geometries), nil
	}

	if hasPolygon && !hasCircle {
		return s.mergePolygons(geometries), nil
	}

	logger.LogInfo("Geometry merge: Using bounding box for mixed Circle/Polygon types. "+
		"This may include unbooked airspace areas (conservative approach for safety). "+
		"geometry_count=%d", len(geometries))
	return s.createBoundingBox(geometries), nil
}

func (s *AirspaceConflictService) mergeCircles(geometries []model.GeometryData) model.GeometryData {

	var sumLon, sumLat float64
	for _, g := range geometries {
		sumLon += g.Center[0]
		sumLat += g.Center[1]
	}
	centerLon := sumLon / float64(len(geometries))
	centerLat := sumLat / float64(len(geometries))

	maxDistance := 0.0
	for _, g := range geometries {

		dLon := g.Center[0] - centerLon
		dLat := g.Center[1] - centerLat
		centerDistance := math.Sqrt(dLon*dLon+dLat*dLat) * 111000
		edgeDistance := centerDistance + g.Radius
		if edgeDistance > maxDistance {
			maxDistance = edgeDistance
		}
	}

	return model.GeometryData{
		Type:   "Circle",
		Center: []float64{centerLon, centerLat},
		Radius: maxDistance,
	}
}

func (s *AirspaceConflictService) mergePolygons(geometries []model.GeometryData) model.GeometryData {
	var allCoordinates [][]float64
	for _, g := range geometries {
		allCoordinates = append(allCoordinates, g.Coordinates...)
	}

	hull := computeConvexHull(allCoordinates)

	if len(hull) == 0 {
		return model.GeometryData{
			Type:        "Polygon",
			Coordinates: allCoordinates,
		}
	}

	if !(hull[0][0] == hull[len(hull)-1][0] && hull[0][1] == hull[len(hull)-1][1]) {
		hull = append(hull, hull[0])
	}

	return model.GeometryData{
		Type:        "Polygon",
		Coordinates: hull,
	}
}

func computeConvexHull(points [][]float64) [][]float64 {
	if len(points) == 0 {
		return nil
	}

	uniqMap := make(map[string]struct{}, len(points))
	uniq := make([][2]float64, 0, len(points))
	for _, p := range points {
		if len(p) < 2 {
			continue
		}
		key := fmt.Sprintf("%f,%f", p[0], p[1])
		if _, ok := uniqMap[key]; ok {
			continue
		}
		uniqMap[key] = struct{}{}
		uniq = append(uniq, [2]float64{p[0], p[1]})
	}

	if len(uniq) == 0 {
		return nil
	}
	if len(uniq) == 1 {
		return [][]float64{{uniq[0][0], uniq[0][1]}}
	}

	sort.Slice(uniq, func(i, j int) bool {
		if uniq[i][0] == uniq[j][0] {
			return uniq[i][1] < uniq[j][1]
		}
		return uniq[i][0] < uniq[j][0]
	})

	cross := func(o, a, b [2]float64) float64 {
		return (a[0]-o[0])*(b[1]-o[1]) - (a[1]-o[1])*(b[0]-o[0])
	}

	var lower [][2]float64
	for _, p := range uniq {
		for len(lower) >= 2 && cross(lower[len(lower)-2], lower[len(lower)-1], p) <= 0 {
			lower = lower[:len(lower)-1]
		}
		lower = append(lower, p)
	}

	var upper [][2]float64
	for i := len(uniq) - 1; i >= 0; i-- {
		p := uniq[i]
		for len(upper) >= 2 && cross(upper[len(upper)-2], upper[len(upper)-1], p) <= 0 {
			upper = upper[:len(upper)-1]
		}
		upper = append(upper, p)
	}

	hull := make([][2]float64, 0, len(lower)+len(upper)-2)
	hull = append(hull, lower...)
	if len(upper) > 0 {
		hull = append(hull, upper[1:len(upper)-1]...)
	}

	res := make([][]float64, 0, len(hull))
	for _, p := range hull {
		res = append(res, []float64{p[0], p[1]})
	}
	return res
}

func (s *AirspaceConflictService) createBoundingBox(geometries []model.GeometryData) model.GeometryData {
	minLon, minLat := 180.0, 90.0
	maxLon, maxLat := -180.0, -90.0

	for _, g := range geometries {
		if g.Type == "Circle" {

			radiusDeg := g.Radius / 111000
			minLon = math.Min(minLon, g.Center[0]-radiusDeg)
			maxLon = math.Max(maxLon, g.Center[0]+radiusDeg)
			minLat = math.Min(minLat, g.Center[1]-radiusDeg)
			maxLat = math.Max(maxLat, g.Center[1]+radiusDeg)
		} else if g.Type == "Polygon" {

			for _, coord := range g.Coordinates {
				minLon = math.Min(minLon, coord[0])
				maxLon = math.Max(maxLon, coord[0])
				minLat = math.Min(minLat, coord[1])
				maxLat = math.Max(maxLat, coord[1])
			}
		}
	}

	return model.GeometryData{
		Type: "Polygon",
		Coordinates: [][]float64{
			{minLon, minLat},
			{maxLon, minLat},
			{maxLon, maxLat},
			{minLon, maxLat},
			{minLon, minLat},
		},
	}
}

func (s *AirspaceConflictService) CallConformityAssessmentForChildren(
	ctx context.Context,
	childDomains []*model.UaslReservation,
	vehicleDetail model.VehicleDetailInfo,
) (model.ConformityAssessmentList, error) {
	if s.conformityAssessmentGW == nil {
		logger.LogInfo("ConformityAssessment Gateway is not configured, skipping conformity assessment")
		return nil, nil
	}

	if vehicleDetail.Maker == "" || vehicleDetail.ModelNumber == "" {
		logger.LogInfo("Aircraft info is incomplete or not provided, but will attempt conformity assessment",
			"maker", vehicleDetail.Maker,
			"model_number", vehicleDetail.ModelNumber)
	}

	var length float64
	if vehicleDetail.Length != "" {
		if l, err := strconv.ParseFloat(vehicleDetail.Length, 64); err == nil {
			length = l
		}
	}
	conformityAircraft := &model.ConformityAssessmentAircraft{
		Maker:       vehicleDetail.Maker,
		ModelNumber: vehicleDetail.ModelNumber,
		Name:        vehicleDetail.Name,
		Type:        vehicleDetail.Type,
		Length:      length,
	}

	var aggregated model.ConformityAssessmentList
	for _, child := range childDomains {
		if child == nil {
			continue
		}

		if child.ExUaslSectionID == nil || *child.ExUaslSectionID == "" {
			logger.LogInfo("ExUaslSectionID is not set, skipping conformity assessment for child")
			continue
		}

		req := model.ConformityAssessmentRequest{
			UaslSectionID: *child.ExUaslSectionID,
			StartAt:       child.StartAt.Format(time.RFC3339),
			EndAt:         child.EndAt.Format(time.RFC3339),
			AircraftInfo:  *conformityAircraft,
		}

		logger.LogInfo("Calling ConformityAssessment conformity assessment API",
			"uasl_section_id", req.UaslSectionID,
			"aircraft_maker", conformityAircraft.Maker,
			"aircraft_model", conformityAircraft.ModelNumber)

		resp, err := s.conformityAssessmentGW.PostConformityAssessment(ctx, "", req)
		if err != nil {

			logger.LogError("ConformityAssessment conformity assessment API failed, continuing with other children",
				"uasl_section_id", req.UaslSectionID,
				"error", err)
			continue
		}

		assessmentItem := model.ConformityAssessmentItem{
			UaslSectionID:     *child.ExUaslSectionID,
			StartAt:           child.StartAt,
			EndAt:             child.EndAt,
			AircraftInfo:      vehicleDetail,
			EvaluationResults: resp.EvaluationResults,
			Type:              resp.Type,
			Reasons:           resp.Reasons,
		}

		aggregated = append(aggregated, assessmentItem)

		logger.LogInfo("ConformityAssessment conformity assessment completed",
			"uasl_section_id", req.UaslSectionID,
			"evaluation_results", resp.EvaluationResults,
			"type", resp.Type)
	}

	return aggregated, nil
}

func (s *AirspaceConflictService) CheckUaslReservationConflict(
	ctx context.Context,
	uaslRequests []model.UaslCheckRequest,
) (*model.ResourceConflictResult, error) {
	if s.uaslReservationRepo == nil {
		return nil, nil
	}

	queries := make([]model.UaslCheckRequest, 0, len(uaslRequests))
	for _, ar := range uaslRequests {
		if err := ar.TimeRange.Validate(); err != nil {
			return nil, fmt.Errorf("invalid uasl time range: %w", err)
		}
		q, err := model.NewUaslCheckRequest(ar.ExUaslSectionID, ar.TimeRange)
		if err != nil {
			return nil, fmt.Errorf("invalid uasl check request: %w", err)
		}
		queries = append(queries, q)
	}

	existingReservations, err := s.uaslReservationRepo.CheckConflictReservations(queries)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch uasl reservations: %w", err)
	}

	if len(existingReservations) > 0 {

		sectionIDSet := make(map[string]struct{})
		for _, r := range existingReservations {
			if r.ExUaslSectionID != nil && *r.ExUaslSectionID != "" {
				sectionIDSet[*r.ExUaslSectionID] = struct{}{}
			}
		}
		conflictedIDs := make([]string, 0, len(sectionIDSet))
		for id := range sectionIDSet {
			conflictedIDs = append(conflictedIDs, id)
		}
		return &model.ResourceConflictResult{
			ConflictType:  "UASL",
			ConflictedIDs: conflictedIDs,
		}, nil
	}

	return nil, nil
}
