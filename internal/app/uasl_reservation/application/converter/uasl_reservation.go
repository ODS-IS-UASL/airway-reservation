package converter

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/pkg/logger"
	"uasl-reservation/internal/pkg/util"
)

type uaslReservationConverter struct{}

func NewUaslReservationDomain() *uaslReservationConverter {
	return &uaslReservationConverter{}
}

func (uaslReservationConverter) ToUaslReservationResponse(uaslReservation *model.UaslReservation) *model.UaslReservationMessage {
	if uaslReservation == nil {
		return &model.UaslReservationMessage{}
	}

	var exUaslSectionID, exUaslID, exAdministratorID string
	if uaslReservation.ExUaslSectionID != nil {
		exUaslSectionID = *uaslReservation.ExUaslSectionID
	}
	if uaslReservation.ExUaslID != nil {
		exUaslID = *uaslReservation.ExUaslID
	}
	if uaslReservation.ExAdministratorID != nil {
		exAdministratorID = *uaslReservation.ExAdministratorID
	}

	var pricingRuleVersion, amount, sequence int32
	if uaslReservation.PricingRuleVersion != nil {
		v := *uaslReservation.PricingRuleVersion
		if v > (1<<31)-1 {
			logger.LogError("pricingRuleVersion exceeds int32 max, capping", "value", v)
			pricingRuleVersion = int32((1 << 31) - 1)
		} else if v < -1<<31 {
			logger.LogError("pricingRuleVersion below int32 min, capping", "value", v)
			pricingRuleVersion = int32(-1 << 31)
		} else {
			pricingRuleVersion = int32(v)
		}
	}
	if uaslReservation.Amount != nil {
		v := *uaslReservation.Amount
		if v > (1<<31)-1 {
			logger.LogError("amount exceeds int32 max, capping", "value", v)
			amount = int32((1 << 31) - 1)
		} else if v < -1<<31 {
			logger.LogError("amount below int32 min, capping", "value", v)
			amount = int32(-1 << 31)
		} else {
			amount = int32(v)
		}
	}
	if uaslReservation.Sequence != nil {
		v := *uaslReservation.Sequence
		if v > (1<<31)-1 {
			logger.LogError("sequence exceeds int32 max, capping", "value", v)
			sequence = int32((1 << 31) - 1)
		} else if v < -1<<31 {
			logger.LogError("sequence below int32 min, capping", "value", v)
			sequence = int32(-1 << 31)
		} else {
			sequence = int32(v)
		}
	}

	res := &model.UaslReservationMessage{
		ID:                      uaslReservation.ID.ToString(),
		RequestID:               uaslReservation.RequestID.ToString(),
		ParentUaslReservationID: uaslReservation.ParentUaslReservationID.ToString(),
		ExUaslSectionID:         exUaslSectionID,
		ExUaslID:                exUaslID,
		ExAdministratorID:       exAdministratorID,
		StartAt:                 uaslReservation.StartAt,
		EndAt:                   uaslReservation.EndAt,
		CreatedAt:               uaslReservation.CreatedAt,
		UpdatedAt:               uaslReservation.UpdatedAt,
		AirspaceID:              uaslReservation.AirspaceID.ToString(),
		ExReservedBy:            uaslReservation.ExReservedBy.ToString(),
		OrganizationID:          uaslReservation.OrganizationID.ToString(),
		ProjectID:               uaslReservation.ProjectID.ToString(),
		OperationID:             uaslReservation.OperationID.ToString(),
		Status:                  uaslReservation.Status.ToString(),
		PricingRuleVersion:      pricingRuleVersion,
		Amount:                  amount,
		Sequence:                sequence,
		FlightPurpose:           uaslReservation.FlightPurpose,
	}

	if uaslReservation.AcceptedAt != nil {
		t := *uaslReservation.AcceptedAt
		res.AcceptedAt = &t
	}
	if uaslReservation.EstimatedAt != nil {
		t := *uaslReservation.EstimatedAt
		res.EstimatedAt = &t
	}
	if uaslReservation.FixedAt != nil {
		t := *uaslReservation.FixedAt
		res.FixedAt = &t
	}

	if len(uaslReservation.ConformityAssessment) > 0 {
		res.ConformityAssessmentResults = toConformityAssessmentResultsFromList(uaslReservation.ConformityAssessment)
	}

	if len(uaslReservation.DestinationReservations) > 0 {
		destInfos := make([]*model.DestinationReservationInfo, 0, len(uaslReservation.DestinationReservations))
		for _, d := range uaslReservation.DestinationReservations {
			destInfos = append(destInfos, &model.DestinationReservationInfo{
				ReservationID:     d.ReservationID,
				ExUaslID:          d.ExUaslID,
				ExAdministratorID: d.ExAdministratorID,
			})
		}
		res.DestinationReservations = destInfos
	}

	return res
}

func (c uaslReservationConverter) ToCompositeUaslReservationResponse(
	parentReservation *model.UaslReservation,
	childReservations []*model.UaslReservation,
	conflict *model.AirspaceConflictResult,
	vehicles []*model.VehicleElement,
	ports []*model.PortElement,
) *model.ReserveCompositeUaslResponse {
	aircraftInfo := ToAircraftInfoModel(parentReservation)
	if aircraftInfo != nil {
		for _, v := range vehicles {
			v.AircraftInfo = aircraftInfo
		}

		if len(vehicles) == 0 && parentReservation != nil {
			ve := &model.VehicleElement{
				Name: aircraftInfo.Name,
			}
			vehicles = append(vehicles, ve)
		}
	}
	return &model.ReserveCompositeUaslResponse{
		ParentUaslReservation:   c.ToUaslReservationResponse(parentReservation),
		ChildUaslReservations:   c.ToChildReservationMessagesWithConformity(childReservations, parentReservation),
		ConflictedFlightPlanIDs: ToConflictedFlightPlanIDs(conflict),
		Vehicles:                vehicles,
		Ports:                   ports,
	}
}

func ToConformityAssessmentResults(parentReservation *model.UaslReservation) []*model.ConformityAssessmentResultMsg {
	if parentReservation == nil || len(parentReservation.ConformityAssessment) == 0 {
		return []*model.ConformityAssessmentResultMsg{}
	}

	return toConformityAssessmentResultsFromList(parentReservation.ConformityAssessment)
}

func toConformityAssessmentResultsFromList(list model.ConformityAssessmentList) []*model.ConformityAssessmentResultMsg {
	if len(list) == 0 {
		return []*model.ConformityAssessmentResultMsg{}
	}

	results := make([]*model.ConformityAssessmentResultMsg, 0, len(list))
	for _, assessment := range list {
		result := &model.ConformityAssessmentResultMsg{
			UaslSectionID:     assessment.UaslSectionID,
			EvaluationResults: assessment.EvaluationResults,
			Type:              assessment.Type,
			Reasons:           assessment.Reasons,
		}

		ai := assessment.AircraftInfo
		if ai.Maker != "" || ai.ModelNumber != "" || ai.AircraftInfoID != 0 || ai.RegistrationID != "" {
			result.AircraftInfo = vehicleDetailToReservationAircraftInfo(&ai)
		}

		results = append(results, result)
	}

	return results
}

func vehicleDetailToReservationAircraftInfo(ai *model.VehicleDetailInfo) *model.ReservationAircraftInfo {
	if ai == nil {
		return nil
	}
	length := float64(0)
	if v, err := strconv.ParseFloat(ai.Length, 64); err == nil {
		length = v
	}
	return &model.ReservationAircraftInfo{
		AircraftInfoID: ai.AircraftInfoID,
		RegistrationID: ai.RegistrationID,
		Maker:          ai.Maker,
		ModelNumber:    ai.ModelNumber,
		Name:           ai.Name,
		Type:           ai.Type,
		Length:         length,
	}
}

func ToAircraftInfoModel(parentReservation *model.UaslReservation) *model.ReservationAircraftInfo {
	if parentReservation == nil || len(parentReservation.ConformityAssessment) == 0 {
		return nil
	}
	ai := parentReservation.ConformityAssessment[0].AircraftInfo
	if ai.Maker == "" && ai.ModelNumber == "" && ai.AircraftInfoID == 0 && ai.RegistrationID == "" {
		return nil
	}
	return vehicleDetailToReservationAircraftInfo(&ai)
}

func (c uaslReservationConverter) ToChildReservationMessages(children []*model.UaslReservation) []*model.UaslReservationMessage {
	if len(children) == 0 {
		return []*model.UaslReservationMessage{}
	}

	childMessages := make([]*model.UaslReservationMessage, 0, len(children))
	for _, child := range children {
		childMessages = append(childMessages, c.ToUaslReservationResponse(child))
	}

	return childMessages
}

func (c uaslReservationConverter) ToChildReservationMessagesWithConformity(
	children []*model.UaslReservation,
	parent *model.UaslReservation,
) []*model.UaslReservationMessage {
	if len(children) == 0 {
		return []*model.UaslReservationMessage{}
	}

	childMessages := make([]*model.UaslReservationMessage, 0, len(children))
	for _, child := range children {
		msg := c.ToUaslReservationResponse(child)
		if len(child.ConformityAssessment) > 0 {
			msg.ConformityAssessmentResults = toConformityAssessmentResultsFromList(child.ConformityAssessment)
		} else {
			msg.ConformityAssessmentResults = toConformityAssessmentResultsFromList(
				filterConformityForChild(parent, child),
			)
		}
		childMessages = append(childMessages, msg)
	}

	return childMessages
}
func ToConflictedFlightPlanIDs(conflict *model.AirspaceConflictResult) []string {
	if conflict == nil || !conflict.HasConflict || len(conflict.ConflictedFlightPlanIDs) == 0 {
		return []string{}
	}

	result := make([]string, 0, len(conflict.ConflictedFlightPlanIDs))
	for _, fpID := range conflict.ConflictedFlightPlanIDs {
		if fpID != "" {
			result = append(result, fpID)
		}
	}

	return result
}

func filterConformityForChild(parent *model.UaslReservation, child *model.UaslReservation) model.ConformityAssessmentList {
	if parent == nil || child == nil || len(parent.ConformityAssessment) == 0 {
		return model.ConformityAssessmentList{}
	}
	if child.ExUaslSectionID == nil || *child.ExUaslSectionID == "" {
		return model.ConformityAssessmentList{}
	}
	sectionID := *child.ExUaslSectionID

	out := make(model.ConformityAssessmentList, 0)
	for _, ca := range parent.ConformityAssessment {
		if ca.UaslSectionID != sectionID {
			continue
		}
		if !ca.StartAt.IsZero() && !ca.EndAt.IsZero() {
			if !child.StartAt.Equal(ca.StartAt) || !child.EndAt.Equal(ca.EndAt) {
				continue
			}
		}
		out = append(out, ca)
	}
	return out
}

func ApplyConformityToChildren(parent *model.UaslReservation, children []*model.UaslReservation) {
	if parent == nil || len(parent.ConformityAssessment) == 0 || len(children) == 0 {
		return
	}
	for _, child := range children {
		if child == nil || child.ExUaslSectionID == nil || *child.ExUaslSectionID == "" {
			continue
		}
		sectionID := *child.ExUaslSectionID
		child.ConformityAssessment = model.ConformityAssessmentList{}
		for _, ca := range parent.ConformityAssessment {
			if ca.UaslSectionID != sectionID {
				continue
			}
			if !ca.StartAt.IsZero() && !ca.EndAt.IsZero() {
				if !child.StartAt.Equal(ca.StartAt) || !child.EndAt.Equal(ca.EndAt) {
					continue
				}
			}
			child.ConformityAssessment = append(child.ConformityAssessment, ca)
		}
	}
}

func ToExternalConformity(
	children []*model.UaslReservation,
	externalResults []*model.ExternalReservationResult,
) {
	if len(externalResults) == 0 {
		return
	}
	for _, ext := range externalResults {
		if ext == nil || ext.ReservationData == nil {
			continue
		}
		type caEntry struct {
			startAt time.Time
			endAt   time.Time
			item    model.ConformityAssessmentItem
			key     string
		}
		entriesByUaslSection := make(map[string][]caEntry)
		seen := make(map[string]struct{})

		for _, destRes := range ext.ReservationData.DestinationReservations {
			if len(destRes.ConformityAssessmentResults) == 0 {
				continue
			}
			sectionTimes := make(map[string][][2]string, len(destRes.UaslSections))
			for _, sec := range destRes.UaslSections {
				sectionTimes[sec.UaslSectionID] = append(sectionTimes[sec.UaslSectionID], [2]string{sec.StartAt, sec.EndAt})
			}
			for _, ca := range destRes.ConformityAssessmentResults {
				timeRanges, ok := sectionTimes[ca.UaslSectionID]
				if !ok || len(timeRanges) == 0 {
					continue
				}
				eval := false
				if v, err := strconv.ParseBool(ca.EvaluationResults); err == nil {
					eval = v
				}
				var aircraftInfo model.VehicleDetailInfo
				if ca.AircraftInfo != nil {
					aircraftInfo = model.VehicleDetailInfo{
						AircraftInfoID: ca.AircraftInfo.AircraftInfoID,
						RegistrationID: ca.AircraftInfo.RegistrationID,
						Maker:          ca.AircraftInfo.Maker,
						ModelNumber:    ca.AircraftInfo.ModelNumber,
						Name:           ca.AircraftInfo.Name,
						Type:           ca.AircraftInfo.Type,
						Length:         fmt.Sprintf("%v", ca.AircraftInfo.Length),
					}
				}
				for _, tr := range timeRanges {
					startAt, err1 := time.Parse(time.RFC3339, tr[0])
					endAt, err2 := time.Parse(time.RFC3339, tr[1])
					if err1 != nil || err2 != nil {
						continue
					}
					item := model.ConformityAssessmentItem{
						UaslSectionID:     ca.UaslSectionID,
						StartAt:           startAt,
						EndAt:             endAt,
						AircraftInfo:      aircraftInfo,
						EvaluationResults: eval,
						Type:              ca.Type,
						Reasons:           ca.Reasons,
					}
					key := destRes.UaslID + "|" + ca.UaslSectionID + "|" + startAt.Format(time.RFC3339) + "|" + endAt.Format(time.RFC3339) + "|" + ca.Type + "|" + ca.Reasons
					if _, ok := seen[key]; ok {
						continue
					}
					seen[key] = struct{}{}
					mapKey := destRes.UaslID + "|" + ca.UaslSectionID
					entriesByUaslSection[mapKey] = append(entriesByUaslSection[mapKey], caEntry{
						startAt: startAt,
						endAt:   endAt,
						item:    item,
						key:     key,
					})
				}
			}
		}

		for _, child := range ext.ChildDomains {
			if child == nil || child.ExUaslSectionID == nil || child.ExUaslID == nil {
				continue
			}
			sectionID := *child.ExUaslSectionID
			uaslID := *child.ExUaslID
			mapKey := uaslID + "|" + sectionID
			entries, ok := entriesByUaslSection[mapKey]
			if !ok {
				logger.LogInfo("ToExternalConformity: no entry for child",
					"uasl_id", uaslID,
					"section_id", sectionID,
					"start_at", child.StartAt.Format(time.RFC3339),
					"end_at", child.EndAt.Format(time.RFC3339),
				)
				continue
			}
			child.ConformityAssessment = model.ConformityAssessmentList{}
			matched := false
			for _, entry := range entries {
				if child.EndAt.Before(entry.startAt) || entry.endAt.Before(child.StartAt) {
					continue
				}
				matched = true
				child.ConformityAssessment = append(child.ConformityAssessment, entry.item)
			}
			if !matched {
				logger.LogInfo("ToExternalConformity: entry found but no time overlap",
					"uasl_id", uaslID,
					"section_id", sectionID,
					"start_at", child.StartAt.Format(time.RFC3339),
					"end_at", child.EndAt.Format(time.RFC3339),
				)
			}
		}
	}

	if len(children) == 0 {
		return
	}
	type key struct {
		uaslID     string
		sectionID  string
		startAtStr string
		endAtStr   string
	}
	externalCA := make(map[key]model.ConformityAssessmentList)
	for _, ext := range externalResults {
		if ext == nil || len(ext.ChildDomains) == 0 {
			continue
		}
		for _, child := range ext.ChildDomains {
			if child == nil || child.ExUaslID == nil || child.ExUaslSectionID == nil {
				continue
			}
			if len(child.ConformityAssessment) == 0 {
				continue
			}
			k := key{
				uaslID:     *child.ExUaslID,
				sectionID:  *child.ExUaslSectionID,
				startAtStr: child.StartAt.Format(time.RFC3339),
				endAtStr:   child.EndAt.Format(time.RFC3339),
			}
			externalCA[k] = child.ConformityAssessment
		}
	}

	if len(externalCA) == 0 {
		return
	}

	for _, child := range children {
		if child == nil || child.ExUaslID == nil || child.ExUaslSectionID == nil {
			continue
		}
		k := key{
			uaslID:     *child.ExUaslID,
			sectionID:  *child.ExUaslSectionID,
			startAtStr: child.StartAt.Format(time.RFC3339),
			endAtStr:   child.EndAt.Format(time.RFC3339),
		}
		if ca, ok := externalCA[k]; ok && len(ca) > 0 {
			child.ConformityAssessment = ca
		} else if child.ExUaslID != nil && child.ExUaslSectionID != nil {
			logger.LogInfo("ToExternalConformity: no external CA for response child",
				"uasl_id", *child.ExUaslID,
				"section_id", *child.ExUaslSectionID,
				"start_at", child.StartAt.Format(time.RFC3339),
				"end_at", child.EndAt.Format(time.RFC3339),
			)
		}
	}
}

func (c uaslReservationConverter) ToUaslReservationListItem(item *model.UaslReservationListItem) (*model.UaslReservationListItemMsg, error) {
	if item == nil || item.Parent == nil {
		return nil, nil
	}

	vehicles := make([]*model.VehicleElement, 0)
	ports := make([]*model.PortElement, 0)

	for _, resource := range item.ExternalResources {
		switch resource.ResourceType {
		case model.ExternalResourceTypeVehicle:
			vehicle := &model.VehicleElement{
				VehicleID:     resource.ExResourceID,
				ReservationID: resource.ExReservationID,
				Name:          resource.ResourceName,
				Amount:        int(util.SafeInt32FromPtr(resource.Amount)),
			}
			if resource.StartAt != nil {
				vehicle.StartAt = resource.StartAt.Format(time.RFC3339)
			}
			if resource.EndAt != nil {
				vehicle.EndAt = resource.EndAt.Format(time.RFC3339)
			}
			if aircraft := ToAircraftInfoModel(item.Parent); aircraft != nil {
				vehicle.AircraftInfo = aircraft
			}
			vehicles = append(vehicles, vehicle)
		case model.ExternalResourceTypePort:
			port := &model.PortElement{
				PortID:            resource.ExResourceID,
				ReservationID:     resource.ExReservationID,
				Name:              resource.ResourceName,
				UsageType:         int(util.SafeInt32FromPtr(resource.UsageType)),
				Amount:            int(util.SafeInt32FromPtr(resource.Amount)),
				ExAdministratorID: resource.ExAdministratorID,
			}
			if resource.StartAt != nil {
				port.StartAt = resource.StartAt.Format(time.RFC3339)
			}
			if resource.EndAt != nil {
				port.EndAt = resource.EndAt.Format(time.RFC3339)
			}
			ports = append(ports, port)
		}
	}

	hasVehicleID := false
	for _, v := range vehicles {
		if v != nil && v.VehicleID != "" {
			hasVehicleID = true
			break
		}
	}
	if !hasVehicleID {
		if aircraft := ToAircraftInfoModel(item.Parent); aircraft != nil {
			vehicles = append(vehicles, &model.VehicleElement{
				AircraftInfo: aircraft,
			})
		}
	}

	return &model.UaslReservationListItemMsg{
		ParentUaslReservation: c.ToUaslReservationResponse(item.Parent),
		ChildUaslReservations: c.ToChildReservationMessagesWithConformity(item.Children, item.Parent),
		Vehicles:              vehicles,
		Ports:                 ports,
		FlightPurpose:         item.FlightPurpose,
	}, nil
}

func (c uaslReservationConverter) ToUaslReservationListItems(items []*model.UaslReservationListItem) []*model.UaslReservationListItemMsg {
	if len(items) == 0 {
		return []*model.UaslReservationListItemMsg{}
	}

	result := make([]*model.UaslReservationListItemMsg, 0, len(items))
	for _, item := range items {
		if item == nil || item.Parent == nil {
			continue
		}

		converted, err := c.ToUaslReservationListItem(item)
		if err != nil || converted == nil {

			continue
		}
		result = append(result, converted)
	}

	return result
}

func (c uaslReservationConverter) ToExternalReservationListItems(externalItems []model.ExternalReservationListItem) []*model.UaslReservationListItemMsg {
	if len(externalItems) == 0 {
		return []*model.UaslReservationListItemMsg{}
	}

	result := make([]*model.UaslReservationListItemMsg, 0, len(externalItems))
	for _, extItem := range externalItems {

		parent := &model.UaslReservationMessage{
			RequestID:    extItem.RequestID,
			ExReservedBy: extItem.OperatorID,
			Status:       extItem.Status,
		}
		if extItem.ReservedAt != "" {
			if t, err := time.Parse(time.RFC3339, extItem.ReservedAt); err == nil {
				parent.FixedAt = &t
			}
		}
		if extItem.EstimatedAt != "" {
			if t, err := time.Parse(time.RFC3339, extItem.EstimatedAt); err == nil {
				parent.EstimatedAt = &t
			}
		}
		if extItem.UpdatedAt != "" {
			if t, err := time.Parse(time.RFC3339, extItem.UpdatedAt); err == nil {
				parent.UpdatedAt = t
			}
		}
		if extItem.OriginReservation != nil {
			parent.ID = extItem.OriginReservation.ReservationID
			parent.ExUaslID = extItem.OriginReservation.UaslID
			parent.ExAdministratorID = extItem.OriginReservation.AdministratorID
		}
		if len(extItem.DestinationReservations) > 0 {
			destInfos := make([]*model.DestinationReservationInfo, 0, len(extItem.DestinationReservations))
			for _, reservation := range extItem.DestinationReservations {
				destInfos = append(destInfos, &model.DestinationReservationInfo{
					ReservationID:     reservation.ReservationID,
					ExUaslID:          reservation.UaslID,
					ExAdministratorID: reservation.AdministratorID,
				})
			}
			parent.DestinationReservations = destInfos
		}

		children := make([]*model.UaslReservationMessage, 0)
		if extItem.OriginReservation != nil {
			for _, section := range extItem.OriginReservation.UaslSections {
				child := &model.UaslReservationMessage{
					RequestID:         extItem.RequestID,
					ExUaslID:          extItem.OriginReservation.UaslID,
					ExUaslSectionID:   section.UaslSectionID,
					ExAdministratorID: extItem.OriginReservation.AdministratorID,
					Sequence:          util.SafeIntToInt32(section.Sequence),
					Amount:            util.SafeIntToInt32(section.Amount),
				}
				if t, err := time.Parse(time.RFC3339, section.StartAt); err == nil {
					child.StartAt = t
				}
				if t, err := time.Parse(time.RFC3339, section.EndAt); err == nil {
					child.EndAt = t
				}
				children = append(children, child)
			}
		}
		for _, reservation := range extItem.DestinationReservations {
			for _, section := range reservation.UaslSections {
				child := &model.UaslReservationMessage{
					RequestID:         extItem.RequestID,
					ExUaslID:          reservation.UaslID,
					ExUaslSectionID:   section.UaslSectionID,
					ExAdministratorID: reservation.AdministratorID,
					Sequence:          util.SafeIntToInt32(section.Sequence),
					Amount:            util.SafeIntToInt32(section.Amount),
				}
				if t, err := time.Parse(time.RFC3339, section.StartAt); err == nil {
					child.StartAt = t
				}
				if t, err := time.Parse(time.RFC3339, section.EndAt); err == nil {
					child.EndAt = t
				}
				children = append(children, child)
			}
		}

		vehicles := make([]*model.VehicleElement, 0)
		if extItem.OriginReservation != nil {
			for _, v := range extItem.OriginReservation.Vehicles {
				vehicles = append(vehicles, &model.VehicleElement{
					VehicleID:     v.VehicleID,
					ReservationID: v.ReservationID,
					Name:          v.Name,
					StartAt:       v.StartAt,
					EndAt:         v.EndAt,
					Amount:        v.Amount,
				})
			}
		}
		for _, reservation := range extItem.DestinationReservations {
			for _, v := range reservation.Vehicles {
				vehicles = append(vehicles, &model.VehicleElement{
					VehicleID:     v.VehicleID,
					ReservationID: v.ReservationID,
					Name:          v.Name,
					StartAt:       v.StartAt,
					EndAt:         v.EndAt,
					Amount:        v.Amount,
				})
			}
		}

		ports := make([]*model.PortElement, 0)
		if extItem.OriginReservation != nil {
			for _, p := range extItem.OriginReservation.Ports {
				ports = append(ports, &model.PortElement{
					PortID:        p.PortID,
					ReservationID: p.ReservationID,
					Name:          p.Name,
					UsageType:     p.UsageType,
					StartAt:       p.StartAt,
					EndAt:         p.EndAt,
					Amount:        p.Amount,
				})
			}
		}
		for _, reservation := range extItem.DestinationReservations {
			for _, p := range reservation.Ports {
				ports = append(ports, &model.PortElement{
					PortID:        p.PortID,
					ReservationID: p.ReservationID,
					Name:          p.Name,
					UsageType:     p.UsageType,
					StartAt:       p.StartAt,
					EndAt:         p.EndAt,
					Amount:        p.Amount,
				})
			}
		}

		item := &model.UaslReservationListItemMsg{
			ParentUaslReservation: parent,
			ChildUaslReservations: children,
			Vehicles:              vehicles,
			Ports:                 ports,
			FlightPurpose:         extItem.FlightPurpose,
		}

		result = append(result, item)
	}

	return result
}

func (c uaslReservationConverter) ToAvailabilityItems(
	reservations []model.UaslReservation,
) []*model.AvailabilityItem {
	if len(reservations) == 0 {
		return []*model.AvailabilityItem{}
	}

	groups := make(map[string][]*model.UaslReservation)
	for i := range reservations {
		r := &reservations[i]
		groups[r.RequestID.ToString()] = append(groups[r.RequestID.ToString()], r)
	}

	result := make([]*model.AvailabilityItem, 0, len(groups))
	for rid, grp := range groups {

		sort.Slice(grp, func(i, j int) bool {
			seqi := -1
			seqj := -1
			if grp[i].Sequence != nil {
				seqi = *grp[i].Sequence
			}
			if grp[j].Sequence != nil {
				seqj = *grp[j].Sequence
			}
			if seqi != seqj {
				return seqi < seqj
			}
			return grp[i].StartAt.Before(grp[j].StartAt)
		})

		flightPurpose := grp[0].FlightPurpose

		operatorID := ""
		if grp[0].ExReservedBy != nil {
			operatorID = grp[0].ExReservedBy.ToString()
		}

		minStart := grp[0].StartAt
		maxEnd := grp[0].EndAt
		for _, r := range grp {
			if r.StartAt.Before(minStart) {
				minStart = r.StartAt
			}
			if r.EndAt.After(maxEnd) {
				maxEnd = r.EndAt
			}
		}

		result = append(result, &model.AvailabilityItem{
			RequestID:     rid,
			OperatorID:    operatorID,
			FlightPurpose: flightPurpose,
			StartAt:       minStart,
			EndAt:         maxEnd,
		})
	}

	return result
}

func (c uaslReservationConverter) ToUaslSectionReservations(
	items []*model.AvailabilityItem,
) []*model.AvailabilityItem {
	if len(items) == 0 {
		return []*model.AvailabilityItem{}
	}
	return items
}

func (c uaslReservationConverter) ToExternalEstimateRequest(
	req *EstimateUaslReservationInput,
	externalSectionIDs []string,
	externalVehicleIDs []string,
	externalPortIDs []string,
) (model.ExternalEstimateRequest, error) {
	externalSections := make([]model.ExternalEstimateSectionRequest, 0)
	externalVehicles := make([]model.ExternalEstimateVehicleRequest, 0)
	externalPorts := make([]model.ExternalEstimatePortRequest, 0)

	var overallStart time.Time
	var overallEnd time.Time
	first := true

	for _, s := range req.UaslSections {
		if s.StartAt != "" {
			t, err := time.Parse(time.RFC3339, s.StartAt)
			if err != nil {
				return model.ExternalEstimateRequest{}, err
			}
			if first || t.Before(overallStart) {
				overallStart = t
			}
			if first || t.After(overallEnd) {
				overallEnd = t
			}
			first = false
		}
		if s.EndAt != "" {
			t, err := time.Parse(time.RFC3339, s.EndAt)
			if err != nil {
				return model.ExternalEstimateRequest{}, err
			}
			if first || t.After(overallEnd) {
				overallEnd = t
			}
			if first || t.Before(overallStart) {
				overallStart = t
			}
			first = false
		}
	}

	for _, p := range req.Ports {
		if p.StartAt != "" {
			t, err := time.Parse(time.RFC3339, p.StartAt)
			if err != nil {
				return model.ExternalEstimateRequest{}, err
			}
			if first || t.Before(overallStart) {
				overallStart = t
			}
			if first || t.After(overallEnd) {
				overallEnd = t
			}
			first = false
		}
		if p.EndAt != "" {
			t, err := time.Parse(time.RFC3339, p.EndAt)
			if err != nil {
				return model.ExternalEstimateRequest{}, err
			}
			if first || t.After(overallEnd) {
				overallEnd = t
			}
			if first || t.Before(overallStart) {
				overallStart = t
			}
			first = false
		}
	}

	contains := func(list []string, v string) bool {
		for _, x := range list {
			if x == v {
				return true
			}
		}
		return false
	}

	for _, s := range req.UaslSections {
		if !contains(externalSectionIDs, s.ExUaslSectionID) {
			continue
		}
		externalSections = append(externalSections, model.ExternalEstimateSectionRequest{
			UaslID:        s.ExUaslID,
			UaslSectionID: s.ExUaslSectionID,
			StartAt:       s.StartAt,
			EndAt:         s.EndAt,
		})
	}

	for _, v := range req.Vehicles {
		if !contains(externalVehicleIDs, v.VehicleID) {
			continue
		}
		vs := v.StartAt
		ve := v.EndAt
		if vs == "" && !overallStart.IsZero() {
			vs = overallStart.Format(time.RFC3339)
		}
		if ve == "" && !overallEnd.IsZero() {
			ve = overallEnd.Format(time.RFC3339)
		}
		externalVehicles = append(externalVehicles, model.ExternalEstimateVehicleRequest{
			VehicleID: v.VehicleID,
			StartAt:   vs,
			EndAt:     ve,
		})
	}

	for _, p := range req.Ports {
		if !contains(externalPortIDs, p.PortID) {
			continue
		}
		ps := p.StartAt
		pe := p.EndAt
		if ps == "" && !overallStart.IsZero() {
			ps = overallStart.Format(time.RFC3339)
		}
		if pe == "" && !overallEnd.IsZero() {
			pe = overallEnd.Format(time.RFC3339)
		}
		externalPorts = append(externalPorts, model.ExternalEstimatePortRequest{
			PortID:  p.PortID,
			StartAt: ps,
			EndAt:   pe,
		})
	}

	return model.ExternalEstimateRequest{
		UaslSections:   externalSections,
		Vehicles:       externalVehicles,
		Ports:          externalPorts,
		IsInterConnect: req.IsInterConnect,
	}, nil
}

func (uaslReservationConverter) ToDestinationReservationsFromGroups(
	isInterConnect bool,
	externalReservationResults []*model.ExternalReservationResult,
	externalGroups []ExternalUaslGroup,
	internalGroups []InternalUaslGroup,
	childDomains []*model.UaslReservation,
) model.DestinationReservationList {

	if isInterConnect {
		return model.DestinationReservationList{}
	}

	destReservations := make(model.DestinationReservationList, 0)

	if len(externalReservationResults) > 0 {
		for _, extResult := range externalReservationResults {
			if extResult.ReservationData == nil {
				continue
			}

			for _, destRes := range extResult.ReservationData.DestinationReservations {
				if destRes.ReservationID == "" {
					continue
				}
				exAdminID := destRes.AdministratorID
				if exAdminID == "" {

					for _, child := range childDomains {
						if child == nil || child.ExUaslID == nil || *child.ExUaslID == "" {
							continue
						}
						if *child.ExUaslID == destRes.UaslID && child.ExAdministratorID != nil && *child.ExAdministratorID != "" {
							exAdminID = *child.ExAdministratorID
							break
						}
					}
				}

				destReservations = append(destReservations, model.DestinationReservationInfo{
					ReservationID:     destRes.ReservationID,
					ExUaslID:          destRes.UaslID,
					ExAdministratorID: exAdminID,
				})
			}
		}
	}

	for i, group := range internalGroups {
		if i > 0 && group.ParentReservation != nil {

			destReservations = append(destReservations, model.DestinationReservationInfo{
				ReservationID:     group.ParentReservation.ID.ToString(),
				ExUaslID:          *group.ParentReservation.ExUaslID,
				ExAdministratorID: *group.ParentReservation.ExAdministratorID,
			})

			logger.LogInfo("Re-entry reservation added to destinationReservations",
				"group_index", i,
				"parent_id", group.ParentReservation.ID.ToString(),
				"ex_uasl_id", *group.ParentReservation.ExUaslID)
		}
	}

	return destReservations
}

type InternalUaslGroup struct {
	ParentReservation *model.UaslReservation
	Children          []*model.UaslReservation
}

type ExternalUaslGroup struct {
	AdministratorID       string
	ExUaslID              string
	ExternalReservationID string
	ParentReservation     *model.UaslReservation
	Children              []*model.UaslReservation
}

func (uaslReservationConverter) ToReservationsForBatchInsert(
	internalGroups []InternalUaslGroup,
	externalGroups []ExternalUaslGroup,
	externalReservationResults []*model.ExternalReservationResult,
	originParent *model.UaslReservation,
	isInterConnect bool,
) []*model.UaslReservation {
	allReservations := make([]*model.UaslReservation, 0)

	for _, group := range internalGroups {
		if isInterConnect && group.ParentReservation != nil {
			if originParent == nil || group.ParentReservation.ID != originParent.ID {
				allReservations = append(allReservations, group.ParentReservation)
			}
		}
		allReservations = append(allReservations, group.Children...)
	}

	for _, group := range externalGroups {
		if group.ParentReservation != nil {
			allReservations = append(allReservations, group.ParentReservation)
		}
		allReservations = append(allReservations, group.Children...)
	}

	if len(externalGroups) == 0 {
		for _, extResult := range externalReservationResults {
			allReservations = append(allReservations, extResult.ChildDomains...)
		}
	}

	logger.LogInfo("Aggregated reservations for batch insert (children + parents)",
		"total_reservations", len(allReservations),
		"internal_groups", len(internalGroups),
		"external_groups", len(externalGroups))

	return allReservations
}
