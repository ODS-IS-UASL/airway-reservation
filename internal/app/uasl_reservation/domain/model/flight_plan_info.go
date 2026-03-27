package model

import "time"

type FlightPlanInfoRequest struct {
	Features      GeometryData
	AllFlightPlan string
	StartTime     time.Time
	FinishTime    time.Time
}

type GeometryData struct {
	Type        string
	Center      []float64
	Radius      float64
	Coordinates [][]float64
}

type FlightPlanInfoResponse struct {
	TotalCount     int
	FlightPlanList []FlightPlanItem
}

type FlightPlanItem struct {
	FlightPlanID string
	StartTime    string
	FinishTime   string
	FlyRoute     FlyRouteData
}

type FlyRouteData struct {
	Type        string
	Center      []float64
	Radius      float64
	Coordinates [][]float64
}

type AirspaceConflictResult struct {
	HasConflict             bool
	ConflictedFlightPlanIDs []string
}

func NewAirspaceConflictResult(hasConflict bool, conflictedFlightPlanIds []string) *AirspaceConflictResult {
	return &AirspaceConflictResult{
		HasConflict:             hasConflict,
		ConflictedFlightPlanIDs: conflictedFlightPlanIds,
	}
}
