package model

type ConformityAssessmentRequest struct {
	UaslSectionID string                       `json:"uaslSectionId"`
	StartAt       string                       `json:"startAt"`
	EndAt         string                       `json:"endAt"`
	AircraftInfo  ConformityAssessmentAircraft `json:"aircraftInfo"`
}

type ConformityAssessmentAircraft struct {
	Maker       string  `json:"maker"`
	ModelNumber string  `json:"modelNumber"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Length      float64 `json:"length"`
}

type ConformityAssessmentResponse struct {
	EvaluationResults bool   `json:"evaluationResults"`
	Type              string `json:"type"`
	Reasons           string `json:"reasons"`
}
