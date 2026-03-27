package converter

import (
	"encoding/json"
	"strings"

	extModel "uasl-reservation/external/uasl/l4/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

func ToFindResourceRequestFromLocation(lat, lng, radiusKm float64) model.FindResourceRequest {
	return model.FindResourceRequest{ServiceName: "L4 Discovery Service", Lat: lat, Lng: lng, RadiusKm: radiusKm}
}
func ToFindResourceRequest(req model.FindResourceRequest) extModel.DiscoveryServiceRequest {
	criteria := map[string]interface{}{}
	if req.Lat != 0 || req.Lng != 0 {
		criteria["location"] = map[string]interface{}{"lat": req.Lat, "lng": req.Lng}
		if req.RadiusKm > 0 {
			criteria["radius"] = req.RadiusKm * 1000
		}
	}
	return extModel.DiscoveryServiceRequest{JSONRPC: "2.0", ID: 1, Method: "findResource", Params: map[string]interface{}{"criteria": criteria}}
}
func ToUaslServiceURLs(dsResp extModel.FindResourceResponse, serviceName string) []model.UaslServiceURL {
	if dsResp.Result == nil {
		return []model.UaslServiceURL{}
	}
	seen := map[string]struct{}{}
	urls := []model.UaslServiceURL{}
	for _, resource := range dsResp.Result.FoundResources {
		appID := resource.DomainApp.DomainAppID
		if appID == "" {
			continue
		}
		if _, ok := seen[appID]; ok {
			continue
		}
		seen[appID] = struct{}{}
		urls = append(urls, model.UaslServiceURL{DiscoveryServiceName: serviceName, DomainAppID: appID, BaseURL: extractBaseURLFromDomainAppRdf(resource.DomainApp.DomainAppRdf)})
	}
	return urls
}
func ToExternalServiceEndpoints(svc model.UaslServiceURL) model.ExternalServiceEndpoints {
	return model.ExternalServiceEndpoints{BaseURL: svc.BaseURL}
}
func extractBaseURLFromDomainAppRdf(domainAppRdf string) string {
	if domainAppRdf == "" {
		return ""
	}
	var doc extModel.DomainAppRdfDoc
	if json.Unmarshal([]byte(domainAppRdf), &doc) != nil {
		return ""
	}
	var fallback string
	for _, node := range doc.Graph {
		candidates := []*extModel.RdfValue{node.UaslListGetRequestUri, node.UaslGetRequestUri, node.UaslReservationsPostRequestUri, node.UaslReservationDetailGetRequestUri, node.ConfirmUaslReservationPutRequestUri, node.AircraftListRequestUri, node.AircraftDetailRequestUri, node.DronePortListRequestUri, node.DronePortEnvironmentRequestUri}
		for _, c := range candidates {
			if c != nil && c.Value != "" {
				return extractBaseURL(c.Value)
			}
		}
		if fallback == "" && node.AccessURL != nil && node.AccessURL.Value != "" {
			fallback = extractBaseURL(node.AccessURL.Value)
		}
	}
	return fallback
}
func extractBaseURL(fullURL string) string {
	if fullURL == "" {
		return ""
	}
	schemeEnd := strings.Index(fullURL, "://")
	if schemeEnd == -1 {
		return ""
	}
	hostStart := schemeEnd + 3
	pathStart := strings.Index(fullURL[hostStart:], "/")
	if pathStart == -1 {
		return fullURL
	}
	return fullURL[:hostStart+pathStart]
}
