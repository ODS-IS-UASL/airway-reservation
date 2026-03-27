package model

import (
	"encoding/json"
	"fmt"
	"strings"
)

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

func (c *ConformityAssessmentResponse) UnmarshalJSON(data []byte) error {
	type alias struct {
		EvaluationResults json.RawMessage `json:"evaluationResults"`
		Type              string          `json:"type"`
		Reasons           string          `json:"reasons"`
	}

	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}

	var evalBool bool
	if len(a.EvaluationResults) > 0 {
		if err := json.Unmarshal(a.EvaluationResults, &evalBool); err != nil {
			var evalStr string
			if errStr := json.Unmarshal(a.EvaluationResults, &evalStr); errStr != nil {
				return fmt.Errorf("invalid evaluationResults type: %w", err)
			}
			switch strings.ToLower(strings.TrimSpace(evalStr)) {
			case "true":
				evalBool = true
			case "false":
				evalBool = false
			default:
				return fmt.Errorf("invalid evaluationResults string value: %s", evalStr)
			}
		}
	}

	c.EvaluationResults = evalBool
	c.Type = a.Type
	c.Reasons = a.Reasons
	return nil
}
