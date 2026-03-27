package model

type FlightPlanInfoRequest struct {
	Features      Geometry `json:"features"`
	AllFlightPlan string   `json:"allFlightPlan,omitempty"`
	StartTime     string   `json:"startTime,omitempty"`
	FinishTime    string   `json:"finishTime,omitempty"`
	UpdateTime    string   `json:"updateTime,omitempty"`
}

type Geometry struct {
	Type        string      `json:"type"`
	Center      []float64   `json:"center,omitempty"`
	Radius      float64     `json:"radius,omitempty"`
	Coordinates [][]float64 `json:"coordinates,omitempty"`
}

type FlightPlanInfoResponse struct {
	FlightPlanInfo []FlightPlan `json:"flightPlanInfo"`
	TotalCount     int          `json:"totalCount"`
}

type FlightPlan struct {
	FlightPlanID                  string                      `json:"flightPlanId"`
	Name                          string                      `json:"name,omitempty"`
	FlightPurpose                 []int                       `json:"flightPurpose,omitempty"`
	Othergyomutext                string                      `json:"othergyomutext,omitempty"`
	Othergyomugaitext             string                      `json:"othergyomugaitext,omitempty"`
	FlightAirspace                []int                       `json:"flightAirspace,omitempty"`
	FlightType                    []int                       `json:"flightType,omitempty"`
	AssistantsNumber              int                         `json:"assistantsNumber,omitempty"`
	DeparturePoint                string                      `json:"departurePoint,omitempty"`
	StartTime                     string                      `json:"startTime"`
	FinishTime                    string                      `json:"finishTime"`
	PlannedMaxTime                int                         `json:"plannedMaxTime,omitempty"`
	PlannedFlightTime             int                         `json:"plannedFlightTime,omitempty"`
	FlightSpeed                   int                         `json:"flightSpeed,omitempty"`
	FlightAltitude                int                         `json:"flightAltitude,omitempty"`
	FlyRoute                      FlyRoute                    `json:"flyRoute"`
	DestinationPoint              string                      `json:"destinationPoint,omitempty"`
	RiskMitigationOnsiteControl   string                      `json:"riskMitigationOnsiteControl,omitempty"`
	RiskMitigationOnsiteControlL3 string                      `json:"riskMitigationOnsiteControlL3,omitempty"`
	RiskMitigationOnsiteControl2  string                      `json:"riskMitigationOnsiteControl2,omitempty"`
	ExceptionalConditionsMooring  string                      `json:"exceptionalConditionsMooring,omitempty"`
	InsuranceInformation          InsuranceInformation        `json:"insuranceInformation,omitempty"`
	Reporter                      Reporter                    `json:"reporter,omitempty"`
	OtherInformation              string                      `json:"otherInformation,omitempty"`
	PilotInfo                     []PilotInfo                 `json:"pilotInfo,omitempty"`
	AircraftInfo                  []AircraftInfo              `json:"aircraftInfo,omitempty"`
	FlightPermitApplicationInfo   FlightPermitApplicationInfo `json:"flightPermitApplicationInfo,omitempty"`
}

type FlyRoute struct {
	Type        string      `json:"type"`
	Center      []float64   `json:"center,omitempty"`
	Radius      float64     `json:"radius,omitempty"`
	Coordinates [][]float64 `json:"coordinates,omitempty"`
}

type InsuranceInformation struct {
	InsuranceCompany    string `json:"insuranceCompany,omitempty"`
	InsuranceProduct    string `json:"insuranceProduct,omitempty"`
	InterPerson         int    `json:"interPerson,omitempty"`
	InterObject         int    `json:"interObject,omitempty"`
	InsuranceAbility    string `json:"insuranceAbility,omitempty"`
	InsuranceSupplement string `json:"insuranceSupplement,omitempty"`
}

type Reporter struct {
	ContactReporterFlag string  `json:"contactReporterFlag,omitempty"`
	ContactReporter     Contact `json:"contactReporter,omitempty"`
}

type PilotInfo struct {
	PilotID                  int     `json:"pilotId,omitempty"`
	ContactPilotFlag         string  `json:"contactPilotFlag,omitempty"`
	ContactPilot             Contact `json:"contactPilot,omitempty"`
	SkillCertificationNumber string  `json:"skillCertificationNumber,omitempty"`
	FirstClass               string  `json:"firstClass,omitempty"`
	SecondClass              string  `json:"secondClass,omitempty"`
	PrivateLicense           string  `json:"privateLicense,omitempty"`
	Maker                    string  `json:"maker,omitempty"`
	Model                    string  `json:"model,omitempty"`
}

type AircraftInfo struct {
	AircraftID       int     `json:"aircraftId,omitempty"`
	Type             string  `json:"type,omitempty"`
	CertificationNum string  `json:"certificationNum,omitempty"`
	Symbol           string  `json:"symbol,omitempty"`
	Model            string  `json:"model,omitempty"`
	Maker            string  `json:"maker,omitempty"`
	Certification1   string  `json:"certification1,omitempty"`
	Certification2   string  `json:"certification2,omitempty"`
	MaxWeight        float64 `json:"maxWeight,omitempty"`
}

type FlightPermitApplicationInfo struct {
	FlightPermitApplicationNumber string  `json:"flightPermitApplicationNumber,omitempty"`
	PermitDate                    string  `json:"permitDate,omitempty"`
	StartDate                     string  `json:"startDate,omitempty"`
	FinishDate                    string  `json:"finishDate,omitempty"`
	ContactPermitFlag             string  `json:"contactPermitFlag,omitempty"`
	ContactPermit                 Contact `json:"contactPermit,omitempty"`
}

type Contact struct {
	Name             string `json:"name,omitempty"`
	Country          string `json:"country,omitempty"`
	Prefectures      string `json:"prefectures,omitempty"`
	Municipality     string `json:"municipality,omitempty"`
	TelephoneCountry string `json:"telephoneCountry,omitempty"`
	Telephone        string `json:"telephone,omitempty"`
	Email            string `json:"email,omitempty"`
}
