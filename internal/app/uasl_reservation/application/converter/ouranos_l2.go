package converter

import (
	"strconv"
	"time"

	ouranosModel "uasl-reservation/external/uasl/l2/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
)

func ToUaslReservationResponse(resp *ouranosModel.UaslReservationResponse) *model.UaslReservationResponse {
	if resp == nil {
		return nil
	}
	data := &model.UaslReservationData{RequestID: resp.RequestID, OperatorID: resp.OperatorID, Status: resp.Status, TotalAmount: resp.TotalAmount, FlightPurpose: resp.FlightPurpose, ReservedAt: resp.ReservedAt, UpdatedAt: resp.UpdatedAt, EstimatedAt: resp.EstimatedAt, PricingRuleVersion: resp.PricingRuleVersion}
	if resp.OriginReservation != nil {
		origin := ToReservationEntity(resp.OriginReservation)
		data.OriginReservation = &origin
	}
	for i := range resp.DestinationReservations {
		data.DestinationReservations = append(data.DestinationReservations, ToReservationEntity(&resp.DestinationReservations[i]))
	}
	return &model.UaslReservationResponse{Data: data}
}
func ToReservationEntity(src *ouranosModel.ReservationEntity) model.ReservationEntity {
	dst := model.ReservationEntity{ReservationID: src.ReservationID, UaslID: src.UaslID, AdministratorID: src.AdministratorID, SubTotalAmount: src.SubTotalAmount, ConflictedFlightPlanIds: src.ConflictedFlightPlanIds}
	for _, s := range src.UaslSections {
		dst.UaslSections = append(dst.UaslSections, model.UaslSectionReservationElement{UaslSectionID: s.UaslSectionID, Sequence: s.Sequence, StartAt: s.StartAt, EndAt: s.EndAt, Amount: s.Amount})
	}
	for _, v := range src.Vehicles {
		elem := model.VehicleReservationElement{VehicleID: v.VehicleID, ReservationID: v.ReservationID, Name: v.Name, StartAt: v.StartAt, EndAt: v.EndAt, Amount: v.Amount}
		if v.AircraftInfo != nil {
			elem.AircraftInfo = &model.ReservationAircraftInfo{AircraftInfoID: v.AircraftInfo.AircraftInfoID, RegistrationID: v.AircraftInfo.RegistrationID, Maker: v.AircraftInfo.Maker, ModelNumber: v.AircraftInfo.ModelNumber, Name: v.AircraftInfo.Name, Type: v.AircraftInfo.Type, Length: v.AircraftInfo.Length}
		}
		dst.Vehicles = append(dst.Vehicles, elem)
	}
	for _, p := range src.Ports {
		dst.Ports = append(dst.Ports, model.PortReservationElement{PortID: p.PortID, ReservationID: p.ReservationID, Name: p.Name, UsageType: p.UsageType, StartAt: p.StartAt, EndAt: p.EndAt, Amount: p.Amount})
	}
	return dst
}
func ToExternalUaslReservationRequest(childDomains []*model.UaslReservation, ports []model.PortReservationRequest, originAdministratorID string, originUaslID string, vehicleDetail *model.VehicleDetailInfo, ignoreFlightPlanConflict bool) ouranosModel.UaslReservationRequest {
	out := ouranosModel.UaslReservationRequest{OriginAdministratorID: originAdministratorID, OriginUaslID: originUaslID, IsInterConnect: true, IgnoreFlightPlanConflict: ignoreFlightPlanConflict}
	if vehicleDetail != nil {
		length, _ := strconv.ParseFloat(vehicleDetail.Length, 64)
		out.OperatingAircraft = &ouranosModel.VehicleDetail{AircraftInfoID: vehicleDetail.AircraftInfoID, RegistrationID: vehicleDetail.RegistrationID, Maker: vehicleDetail.Maker, ModelNumber: vehicleDetail.ModelNumber, Name: vehicleDetail.Name, Type: vehicleDetail.Type, Length: length}
	}
	if len(childDomains) > 0 && childDomains[0] != nil {
		out.RequestID = childDomains[0].RequestID.ToString()
		if childDomains[0].ExReservedBy != nil {
			out.OperatorID = childDomains[0].ExReservedBy.ToString()
		}
	}
	for _, child := range childDomains {
		if child == nil || child.ExUaslID == nil || child.ExUaslSectionID == nil {
			continue
		}
		elem := ouranosModel.UaslSectionElement{UaslID: *child.ExUaslID, UaslSectionID: *child.ExUaslSectionID, StartAt: child.StartAt.Format(time.RFC3339), EndAt: child.EndAt.Format(time.RFC3339)}
		if child.Sequence != nil {
			elem.Sequence = *child.Sequence
		}
		out.UaslSections = append(out.UaslSections, elem)
	}
	for _, p := range ports {
		out.Ports = append(out.Ports, ouranosModel.PortElement{PortID: p.PortID, UsageType: int(p.UsageType), StartAt: p.ReservationTimeFrom.Format(time.RFC3339), EndAt: p.ReservationTimeTo.Format(time.RFC3339)})
	}
	return out
}
func ToExternalAvailabilityRequest(sections []model.AvailabilitySection, vehicleIDs []string, portIDs []string) ouranosModel.AvailabilityRequest {
	out := ouranosModel.AvailabilityRequest{IsInterConnect: true}
	for _, s := range sections {
		out.UaslSections = append(out.UaslSections, ouranosModel.AvailabilityUaslSectionElement{UaslID: s.UaslID, UaslSectionID: s.UaslSectionID})
	}
	for _, id := range vehicleIDs {
		out.Vehicles = append(out.Vehicles, ouranosModel.AvailabilityVehicleInput{VehicleID: id})
	}
	for _, id := range portIDs {
		out.Ports = append(out.Ports, ouranosModel.AvailabilityPortInput{PortID: id})
	}
	return out
}
func ToAvailabilityItems(resp *ouranosModel.AvailabilityItemList) ([]model.AvailabilityItem, []model.VehicleAvailabilityItem, []model.PortAvailabilityItem) {
	if resp == nil {
		return nil, nil, nil
	}
	parse := func(v string) time.Time { t, _ := time.Parse(time.RFC3339, v); return t }
	sections := []model.AvailabilityItem{}
	for _, item := range resp.UaslSections {
		sections = append(sections, model.AvailabilityItem{RequestID: item.RequestID, OperatorID: item.OperatorID, FlightPurpose: item.FlightPurpose, StartAt: parse(item.StartAt), EndAt: parse(item.EndAt)})
	}
	vehicles := []model.VehicleAvailabilityItem{}
	for _, item := range resp.Vehicles {
		vehicles = append(vehicles, model.VehicleAvailabilityItem{RequestID: item.RequestID, ReservationID: item.ReservationID, Name: item.Name, VehicleID: item.VehicleID, StartAt: parse(item.StartAt), EndAt: parse(item.EndAt)})
	}
	ports := []model.PortAvailabilityItem{}
	for _, item := range resp.Ports {
		ports = append(ports, model.PortAvailabilityItem{RequestID: item.RequestID, ReservationID: item.ReservationID, Name: item.Name, PortID: item.PortID, StartAt: parse(item.StartAt), EndAt: parse(item.EndAt)})
	}
	return sections, vehicles, ports
}
func ToExternalReservationSearchRequest(req model.ExternalReservationSearchRequest) ouranosModel.SearchByRequestIDsRequest {
	return ouranosModel.SearchByRequestIDsRequest{RequestIDs: append([]string{}, req.RequestIDs...)}
}
func ToInternalExternalReservationListItems(items []ouranosModel.UaslReservationListItem) []model.ExternalReservationListItem {
	out := []model.ExternalReservationListItem{}
	for _, item := range items {
		row := model.ExternalReservationListItem{RequestID: item.RequestID, OperatorID: item.OperatorID, Status: item.Status, TotalAmount: item.TotalAmount, EstimatedAt: item.EstimatedAt, ReservedAt: item.ReservedAt, UpdatedAt: item.UpdatedAt, FlightPurpose: item.FlightPurpose}
		if item.OriginReservation != nil {
			origin := ToReservationEntity(item.OriginReservation)
			row.OriginReservation = &origin
		}
		for i := range item.DestinationReservations {
			row.DestinationReservations = append(row.DestinationReservations, ToReservationEntity(&item.DestinationReservations[i]))
		}
		out = append(out, row)
	}
	return out
}
func ToExternalUaslConfirmRequest(isInterConnect bool) ouranosModel.UaslReservationConfirmRequest {
	return ouranosModel.UaslReservationConfirmRequest{IsInterConnect: isInterConnect}
}
func ToExternalEstimateRequest(req model.ExternalEstimateRequest) ouranosModel.EstimateRequest {
	out := ouranosModel.EstimateRequest{IsInterConnect: req.IsInterConnect}
	for _, s := range req.UaslSections {
		out.UaslSections = append(out.UaslSections, ouranosModel.EstimateSectionElement{UaslID: s.UaslID, UaslSectionID: s.UaslSectionID, StartAt: s.StartAt, EndAt: s.EndAt})
	}
	for _, v := range req.Vehicles {
		out.Vehicles = append(out.Vehicles, ouranosModel.VehicleElement{VehicleID: v.VehicleID, StartAt: v.StartAt, EndAt: v.EndAt})
	}
	for _, p := range req.Ports {
		out.Ports = append(out.Ports, ouranosModel.EstimatePortElement{PortID: p.PortID, StartAt: p.StartAt, EndAt: p.EndAt})
	}
	return out
}
func ToInternalEstimateResponse(resp *ouranosModel.EstimateResponse) *model.ExternalEstimateResponse {
	if resp == nil {
		return &model.ExternalEstimateResponse{}
	}
	return &model.ExternalEstimateResponse{TotalAmount: resp.TotalAmount, EstimatedAt: resp.EstimatedAt}
}
func ToExternalDestinationReservationNotification(src model.DestinationReservationNotification) ouranosModel.DestinationReservationNotification {
	dst := ouranosModel.DestinationReservationNotification{RequestId: src.RequestId, ReservationId: src.ReservationId, OperatorId: src.OperatorId, UaslId: src.UaslId, AdministratorId: src.AdministratorId, FlightPurpose: src.FlightPurpose, Status: src.Status, SubTotalAmount: src.SubTotalAmount, ReservedAt: src.ReservedAt, EstimatedAt: src.EstimatedAt, UpdatedAt: src.UpdatedAt, ConflictedFlightPlanIds: append([]string{}, src.ConflictedFlightPlanIds...)}
	for _, s := range src.UaslSections {
		dst.UaslSections = append(dst.UaslSections, ouranosModel.UaslSectionReservationElement{UaslSectionID: s.UaslSectionID, Sequence: s.Sequence, StartAt: s.StartAt, EndAt: s.EndAt, Amount: s.Amount})
	}
	for _, p := range src.Ports {
		dst.Ports = append(dst.Ports, ouranosModel.PortReservationElement{PortID: p.PortID, ReservationID: p.ReservationID, Name: p.Name, UsageType: p.UsageType, StartAt: p.StartAt, EndAt: p.EndAt, Amount: p.Amount})
	}
	return dst
}
