package converter

import (
	safetymodel "uasl-reservation/external/uasl/conformity_assessment/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

type ConformityAssessmentConverter struct{}

func NewConformityAssessmentConverter() *ConformityAssessmentConverter {
	return &ConformityAssessmentConverter{}
}

func (c *ConformityAssessmentConverter) ToExternalRequest(
	req model.ConformityAssessmentRequest,
) safetymodel.ConformityAssessmentRequest {
	return safetymodel.ConformityAssessmentRequest{
		UaslSectionID: req.UaslSectionID,
		StartAt:       req.StartAt,
		EndAt:         req.EndAt,
		AircraftInfo: safetymodel.ConformityAssessmentAircraft{
			Maker:       req.AircraftInfo.Maker,
			ModelNumber: req.AircraftInfo.ModelNumber,
			Name:        req.AircraftInfo.Name,
			Type:        req.AircraftInfo.Type,
			Length:      req.AircraftInfo.Length,
		},
	}
}

func (c *ConformityAssessmentConverter) ToInternalResponse(
	resp safetymodel.ConformityAssessmentResponse,
) model.ConformityAssessmentResponse {
	return model.ConformityAssessmentResponse{
		EvaluationResults: resp.EvaluationResults,
		Type:              resp.Type,
		Reasons:           resp.Reasons,
	}
}
