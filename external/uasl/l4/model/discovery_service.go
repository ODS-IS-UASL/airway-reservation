package model

type DiscoveryServiceRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      int                    `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
}

type FindResourceResponse struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      int                    `json:"id"`
	Result  *FindResourceResult    `json:"result,omitempty"`
	Error   *DiscoveryServiceError `json:"error,omitempty"`
}

type FindResourceResult struct {
	FoundResources []FoundResource `json:"foundResources"`
}

type FoundResource struct {
	DomainApp FoundDomainApp         `json:"domainApp"`
	Resource  FoundResource_Resource `json:"resource"`
}

type FoundDomainApp struct {
	DomainAppID          string `json:"domainAppId"`
	DomainAppName        string `json:"domainAppName"`
	DomainAppDescription string `json:"domainAppDescription"`
	DomainAppRdf         string `json:"domainAppRdf"`
}

type FoundResource_Resource struct {
	ResourceID          string `json:"resourceId"`
	ResourceName        string `json:"resourceName"`
	ResourceDescription string `json:"resourceDescription"`
	ResourceRdf         string `json:"resourceRdf"`
}

type DiscoveryServiceError struct {
	Message string                 `json:"message"`
	Path    string                 `json:"path,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
	Code    string                 `json:"code,omitempty"`
}

type DomainAppRdfDoc struct {
	Graph []DomainAppRdfNode `json:"@graph"`
}

type DomainAppRdfNode struct {
	ID   string `json:"@id"`
	Type string `json:"@type"`

	UaslGetRequestUri     *RdfValue `json:"service:uaslGetRequestUri,omitempty"`
	UaslListGetRequestUri *RdfValue `json:"service:uaslListGetRequestUri,omitempty"`

	UaslReservationsPostRequestUri        *RdfValue `json:"service:uaslReservationsPostRequestUri,omitempty"`
	UaslReservationDetailGetRequestUri    *RdfValue `json:"service:uaslReservationDetailGetRequestUri,omitempty"`
	ConfirmUaslReservationPutRequestUri   *RdfValue `json:"service:confirmUaslReservationPutRequestUri,omitempty"`
	OperatorUaslReservationsGetRequestUri *RdfValue `json:"service:operatorUaslReservationsGetRequestUri,omitempty"`

	AircraftListRequestUri         *RdfValue `json:"service:aircraftListRequestUri,omitempty"`
	AircraftDetailRequestUri       *RdfValue `json:"service:aircraftDetailRequestUri,omitempty"`
	DronePortListRequestUri        *RdfValue `json:"service:dronePortListRequestUri,omitempty"`
	DronePortDetailRequestUri      *RdfValue `json:"service:dronePortDetailRequestUri,omitempty"`
	DronePortEnvironmentRequestUri *RdfValue `json:"service:dronePortEnvironmentRequestUri,omitempty"`

	ConformityAssessmentRequestUri *RdfValue `json:"service:conformityAssessmentRequestUri,omitempty"`

	AccessURL *RdfValue `json:"ods:accessURL,omitempty"`
}

type RdfValue struct {
	Type  string `json:"@type"`
	Value string `json:"@value"`
}
