package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"uasl-reservation/internal/app/uasl_reservation/application/converter"
	gatewayIF "uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/repositoryIF"
	"uasl-reservation/internal/pkg/logger"
	"uasl-reservation/internal/pkg/retry"
	"uasl-reservation/internal/pkg/util"
	"uasl-reservation/internal/pkg/value"

	"github.com/google/uuid"
)

type InterconnectReservationService struct {
	ouranosDiscoveryGW  gatewayIF.OuranosDiscoveryGatewayIF
	ouranosProxyGW      gatewayIF.OuranosProxyGatewayIF
	uaslDesignGateway   gatewayIF.UaslDesignGatewayIF
	externalUaslDefRepo repositoryIF.ExternalUaslDefinitionRepositoryIF
	externalUaslResRepo repositoryIF.ExternalUaslResourceRepositoryIF
	uaslAdminRepo       repositoryIF.UaslAdministratorRepositoryIF
	orchestrator        *CompositeReservationOrchestrator
}

func NewInterconnectReservationService(
	ouranosDiscoveryGW gatewayIF.OuranosDiscoveryGatewayIF,
	ouranosProxyGW gatewayIF.OuranosProxyGatewayIF,
	uaslDesignGateway gatewayIF.UaslDesignGatewayIF,
	externalUaslDefRepo repositoryIF.ExternalUaslDefinitionRepositoryIF,
	externalUaslResRepo repositoryIF.ExternalUaslResourceRepositoryIF,
	uaslAdminRepo repositoryIF.UaslAdministratorRepositoryIF,
	orchestrator *CompositeReservationOrchestrator,
) *InterconnectReservationService {
	return &InterconnectReservationService{
		ouranosDiscoveryGW:  ouranosDiscoveryGW,
		ouranosProxyGW:      ouranosProxyGW,
		uaslDesignGateway:   uaslDesignGateway,
		externalUaslDefRepo: externalUaslDefRepo,
		externalUaslResRepo: externalUaslResRepo,
		uaslAdminRepo:       uaslAdminRepo,
		orchestrator:        orchestrator,
	}
}

type urlGroup struct {
	url        string
	requestIDs []string
}

type AdministratorGroup struct {
	AdministratorID       string
	IsInternal            bool
	URL                   string
	ChildDomains          []*model.UaslReservation
	PortRequests          []model.PortReservationRequest
	OriginAdministratorID string
	OriginUaslID          string
	ConfirmationData      *model.UaslReservationData
}

type InterconnectReservationResult struct {
	AdministratorGroups map[string]*AdministratorGroup
	AdministratorOrder  []string
}

func (s *InterconnectReservationService) FillParentOriginFromChildren(
	parent *model.UaslReservation,
	children []*model.UaslReservation,
) {
	if parent == nil || len(children) == 0 {
		return
	}

	needsUaslID := parent.ExUaslID == nil || *parent.ExUaslID == ""
	needsAdminID := parent.ExAdministratorID == nil || *parent.ExAdministratorID == ""
	if !needsUaslID && !needsAdminID {
		return
	}

	originChild := pickOriginChild(children)
	if originChild == nil {
		return
	}

	if needsUaslID && originChild.ExUaslID != nil && *originChild.ExUaslID != "" {
		parent.ExUaslID = originChild.ExUaslID
	}
	if needsAdminID && originChild.ExAdministratorID != nil && *originChild.ExAdministratorID != "" {
		parent.ExAdministratorID = originChild.ExAdministratorID
	}
}

func pickOriginChild(children []*model.UaslReservation) *model.UaslReservation {
	var withSeq *model.UaslReservation
	var withoutSeq *model.UaslReservation

	for _, c := range children {
		if c == nil {
			continue
		}
		if c.Sequence == nil {
			if withoutSeq == nil {
				withoutSeq = c
			}
			continue
		}
		if withSeq == nil || *c.Sequence < *withSeq.Sequence {
			withSeq = c
		}
	}

	if withSeq != nil {
		return withSeq
	}
	return withoutSeq
}

func normalizeInterconnectReservationData(data *model.UaslReservationData) {
	if data == nil {
		return
	}
	if len(data.DestinationReservations) > 0 {
		return
	}
	if data.OriginReservation == nil {
		return
	}

	data.DestinationReservations = []model.ReservationEntity{*data.OriginReservation}
	data.OriginReservation = nil
}

func (s *InterconnectReservationService) ResolveExUaslIDs(ctx context.Context, children []*model.UaslReservation) error {
	if len(children) == 0 {
		return fmt.Errorf("no child reservations found")
	}
	sectionIDsNeedingResolution := make([]string, 0)
	sectionIDToChildIndex := make(map[string][]int)

	for i, child := range children {
		if (child.ExUaslID == nil || *child.ExUaslID == "") && child.ExUaslSectionID != nil && *child.ExUaslSectionID != "" {
			sectionID := *child.ExUaslSectionID
			sectionIDsNeedingResolution = append(sectionIDsNeedingResolution, sectionID)
			sectionIDToChildIndex[sectionID] = append(sectionIDToChildIndex[sectionID], i)
		}
	}

	if len(sectionIDsNeedingResolution) == 0 {
		return nil
	}

	definitions, err := s.externalUaslDefRepo.FindByExSectionIDs(ctx, sectionIDsNeedingResolution)
	if err != nil {
		return nil
	}

	for _, def := range definitions {
		if indices, ok := sectionIDToChildIndex[def.ExUaslSectionID]; ok {
			for _, idx := range indices {

				if def.ExUaslID.Valid && def.ExUaslID.String != nil && *def.ExUaslID.String != "" {
					exUaslID := *def.ExUaslID.String
					children[idx].ExUaslID = &exUaslID
				}

				if def.ExAdministratorID != "" {
					children[idx].ExAdministratorID = &def.ExAdministratorID
				}
			}
		}
	}

	return nil
}

func (s *InterconnectReservationService) ClassifyInterconnectReservation(
	ctx context.Context,
	childDomains []*model.UaslReservation,
	portReservations []model.PortReservationRequest,
	adminResolution *AvailabilityAdminResolution,
) *InterconnectReservationResult {

	uaslDetailsMap := make(map[string]string)
	if adminResolution != nil {
		for exUaslID, endpoints := range adminResolution.ExternalServices {
			if endpoints.BaseURL != "" {
				uaslDetailsMap[exUaslID] = endpoints.BaseURL
			}
		}
	}

	var originAdministratorID, originUaslID string
	if len(childDomains) > 0 {
		if childDomains[0].ExAdministratorID != nil {
			originAdministratorID = *childDomains[0].ExAdministratorID
		}
		if childDomains[0].ExUaslID != nil {
			originUaslID = *childDomains[0].ExUaslID
		}
	}

	administratorGroups, administratorOrder := s.classifyAndOrganize(
		ctx,
		childDomains,
		uaslDetailsMap,
		portReservations,
		originAdministratorID,
		originUaslID,
	)

	return &InterconnectReservationResult{
		AdministratorGroups: administratorGroups,
		AdministratorOrder:  administratorOrder,
	}
}

func (s *InterconnectReservationService) classifyAndOrganize(
	ctx context.Context,
	childDomains []*model.UaslReservation,
	uaslDetailsMap map[string]string,
	portReservations []model.PortReservationRequest,
	originAdministratorID string,
	originUaslID string,
) (map[string]*AdministratorGroup, []string) {

	sort.Slice(childDomains, func(i, j int) bool {
		if childDomains[i].Sequence == nil {
			return false
		}
		if childDomains[j].Sequence == nil {
			return true
		}
		return *childDomains[i].Sequence < *childDomains[j].Sequence
	})

	administratorGroups := make(map[string]*AdministratorGroup)
	administratorOrder := make([]string, 0)

	adminMap := make(map[string]*model.UaslAdministrator)
	if s.uaslAdminRepo != nil {
		adminIDSet := make(map[string]struct{})
		for _, child := range childDomains {
			if child == nil || child.ExAdministratorID == nil || *child.ExAdministratorID == "" {
				continue
			}
			adminIDSet[*child.ExAdministratorID] = struct{}{}
		}
		if len(adminIDSet) > 0 {
			adminIDs := make([]string, 0, len(adminIDSet))
			for id := range adminIDSet {
				adminIDs = append(adminIDs, id)
			}
			admins, err := s.uaslAdminRepo.FindByExAdministratorIDs(ctx, adminIDs)
			if err == nil {
				for _, a := range admins {
					if a != nil {
						adminMap[a.ExAdministratorID] = a
					}
				}
			}
		}
	}

	for _, child := range childDomains {

		if child.ExAdministratorID == nil || *child.ExAdministratorID == "" {
			continue
		}

		adminID := *child.ExAdministratorID
		url := ""
		if child.ExUaslID != nil && *child.ExUaslID != "" {
			if reservationURL, ok := uaslDetailsMap[*child.ExUaslID]; ok && reservationURL != "" {
				url = reservationURL
			}
		}

		group, exists := administratorGroups[adminID]
		if !exists {
			group = &AdministratorGroup{
				AdministratorID:       adminID,
				IsInternal:            false,
				URL:                   url,
				ChildDomains:          make([]*model.UaslReservation, 0),
				PortRequests:          make([]model.PortReservationRequest, 0),
				OriginAdministratorID: originAdministratorID,
				OriginUaslID:          originUaslID,
			}
			if adminID != "unknown" {
				if admin, ok := adminMap[adminID]; ok && admin != nil {

					group.IsInternal = admin.IsInternal
				} else {

					logger.LogError("Failed to find uasl administrator", "administrator_id", adminID, "error", "not found in batch result")
				}
			}
			administratorGroups[adminID] = group
			administratorOrder = append(administratorOrder, adminID)
		}

		group.ChildDomains = append(group.ChildDomains, child)
	}

	if len(portReservations) > 0 {
		for _, adminID := range administratorOrder {
			group := administratorGroups[adminID]
			if group == nil || group.IsInternal {

				continue
			}
			group.PortRequests = append(group.PortRequests, portReservations...)
			logger.LogInfo("Ports broadcast to external administrator group",
				"administrator_id", adminID,
				"port_count", len(portReservations))
		}
	}

	for _, adminID := range administratorOrder {
		group := administratorGroups[adminID]
		if group != nil {
			logger.LogInfo("Administrator group classified",
				"administrator_id", group.AdministratorID,
				"is_internal", group.IsInternal,
				"url", group.URL,
				"child_domains_count", len(group.ChildDomains),
				"port_requests_count", len(group.PortRequests))
		}
	}

	return administratorGroups, administratorOrder
}

func (s *InterconnectReservationService) ReserveExternalUasl(
	ctx context.Context,
	adminGroup AdministratorGroup,
	vehicleDetail *model.VehicleDetailInfo,
	originAdministratorID string,
	originUaslID string,
	sagaOrchestrator *retry.Orchestrator,
	ignoreFlightPlanConflict bool,
) (map[string]string, *model.UaslReservationData, error) {
	reservations := make(map[string]string)

	if len(adminGroup.ChildDomains) == 0 {
		return reservations, nil, nil
	}

	if s.ouranosProxyGW == nil {
		return nil, nil, fmt.Errorf("ouranos proxy gateway not configured")
	}

	if adminGroup.URL == "" {
		return nil, nil, fmt.Errorf("URL is required")
	}

	var reservationResp *model.UaslReservationResponse
	op := func(rc context.Context) error {
		var err error
		reservationResp, err = s.ouranosProxyGW.CreateUaslReservation(rc, adminGroup.URL, adminGroup.ChildDomains, adminGroup.PortRequests, originAdministratorID, originUaslID, vehicleDetail, ignoreFlightPlanConflict)
		return err
	}

	if err := retry.WithBackoff(ctx, op, retry.DefaultConfig()); err != nil {
		return nil, nil, fmt.Errorf("failed to create UASL reservation via Ouranos Proxy (url=%s): %w", adminGroup.URL, err)
	}

	if reservationResp != nil && reservationResp.Error != nil {
		return nil, nil, fmt.Errorf("ouranos API error (url=%s, code=%s): %s",
			adminGroup.URL, reservationResp.Error.Code, reservationResp.Error.Message)
	}

	if reservationResp != nil && reservationResp.Data != nil {
		normalizeInterconnectReservationData(reservationResp.Data)
	}

	if reservationResp != nil && reservationResp.Data != nil && reservationResp.Data.RequestID != "" {
		for _, child := range adminGroup.ChildDomains {
			if child == nil || child.ExUaslSectionID == nil {
				continue
			}
			reservations[*child.ExUaslSectionID] = reservationResp.Data.RequestID
		}

		resID := reservationResp.Data.RequestID
		handle := model.ReservationHandle{
			ID:   resID,
			Type: model.ResourceTypeInterConnectUasl,
			URL:  adminGroup.URL,
		}
		sagaOrchestrator.RecordSuccess(retry.Step{
			Name: fmt.Sprintf("InterConnectUasl-%s", resID),
			Rollback: func(rc context.Context) error {
				if s.ouranosProxyGW == nil {
					return nil
				}
				return s.ouranosProxyGW.DeleteUaslReservation(rc, handle.URL, handle.ID)
			},
			Metadata: map[string]interface{}{
				"type":           "inter_connect_uasl",
				"reservation_id": resID,
				"url":            adminGroup.URL,
			},
		})
	}

	return reservations, reservationResp.Data, nil
}

func (s *InterconnectReservationService) ConfirmExternalUasl(
	ctx context.Context,
	adminGroup AdministratorGroup,
	requestID string,
	sagaOrchestrator *retry.Orchestrator,
) (*model.UaslReservationData, error) {
	if len(adminGroup.ChildDomains) == 0 {
		return nil, nil
	}

	if s.ouranosProxyGW == nil {
		return nil, fmt.Errorf("ouranos proxy gateway not configured")
	}

	if adminGroup.URL == "" {
		return nil, fmt.Errorf("URL is required for external confirmation")
	}

	var confirmResp *model.UaslReservationResponse
	op := func(rc context.Context) error {
		var err error
		confirmResp, err = s.ouranosProxyGW.ConfirmUaslReservation(
			rc,
			adminGroup.URL,
			requestID,
			true,
		)
		return err
	}

	if err := retry.WithBackoff(ctx, op, retry.DefaultConfig()); err != nil {
		return nil, fmt.Errorf("failed to confirm UASL reservation via Ouranos Proxy (url=%s, requestID=%s): %w",
			adminGroup.URL, requestID, err)
	}

	if confirmResp != nil && confirmResp.Error != nil {
		return nil, fmt.Errorf("ouranos API error on confirmation (url=%s, code=%s): %s",
			adminGroup.URL, confirmResp.Error.Code, confirmResp.Error.Message)
	}

	if confirmResp != nil && confirmResp.Data != nil {
		normalizeInterconnectReservationData(confirmResp.Data)
	}

	handle := model.ReservationHandle{
		ID:   requestID,
		Type: model.ResourceTypeInterConnectUasl,
		URL:  adminGroup.URL,
	}
	sagaOrchestrator.RecordSuccess(retry.Step{
		Name: fmt.Sprintf("ConfirmInterConnectUasl-%s", requestID),
		Rollback: func(rc context.Context) error {
			if s.ouranosProxyGW == nil {
				return nil
			}

			logger.LogInfo("WARN: Rollback: cancelling confirmed external uasl reservation",
				"request_id", requestID,
				"url", adminGroup.URL)
			return s.ouranosProxyGW.DeleteUaslReservation(rc, handle.URL, handle.ID)
		},
		Metadata: map[string]interface{}{
			"type":       "confirm_inter_connect_uasl",
			"request_id": requestID,
			"url":        adminGroup.URL,
		},
	})

	logger.LogInfo("External UASL reservation confirmed successfully",
		"request_id", requestID,
		"url", adminGroup.URL,
		"administrator_id", adminGroup.AdministratorID)

	if confirmResp != nil && confirmResp.Data != nil {
		return confirmResp.Data, nil
	}

	return nil, nil
}

func (s *InterconnectReservationService) CancelExternalUasl(
	ctx context.Context,
	adminGroup AdministratorGroup,
	requestID string,
	action string,
	sagaOrchestrator *retry.Orchestrator,
) (*model.UaslReservationData, error) {
	if len(adminGroup.ChildDomains) == 0 {
		return nil, nil
	}

	if s.ouranosProxyGW == nil {
		return nil, fmt.Errorf("ouranos proxy gateway not configured")
	}

	if adminGroup.URL == "" {
		return nil, fmt.Errorf("URL is required for external cancellation")
	}

	var cancelResp *model.UaslReservationResponse
	op := func(rc context.Context) error {
		var err error
		cancelResp, err = s.ouranosProxyGW.CancelUaslReservation(
			rc,
			adminGroup.URL,
			requestID,
			true,
			action,
		)
		return err
	}

	if err := retry.WithBackoff(ctx, op, retry.DefaultConfig()); err != nil {
		return nil, fmt.Errorf("failed to cancel UASL reservation via Ouranos Proxy (url=%s, requestID=%s): %w",
			adminGroup.URL, requestID, err)
	}

	if cancelResp != nil && cancelResp.Error != nil {
		return nil, fmt.Errorf("ouranos API error on cancellation (url=%s, code=%s): %s",
			adminGroup.URL, cancelResp.Error.Code, cancelResp.Error.Message)
	}

	if cancelResp != nil && cancelResp.Data != nil {
		normalizeInterconnectReservationData(cancelResp.Data)
	}

	var cancelledPorts []model.PortReservationRequest
	if cancelResp != nil && cancelResp.Data != nil {
		for _, destRes := range cancelResp.Data.DestinationReservations {
			for _, portResp := range destRes.Ports {
				startAt, _ := time.Parse(time.RFC3339, portResp.StartAt)
				endAt, _ := time.Parse(time.RFC3339, portResp.EndAt)
				usageVal := util.SafeIntToInt32(portResp.UsageType)

				cancelledPorts = append(cancelledPorts, model.PortReservationRequest{
					PortID:              portResp.PortID,
					UsageType:           usageVal,
					ReservationTimeFrom: startAt,
					ReservationTimeTo:   endAt,
				})
			}
		}
	}

	handle := model.ReservationHandle{
		ID:   requestID,
		Type: model.ResourceTypeInterConnectUasl,
		URL:  adminGroup.URL,
	}
	sagaOrchestrator.RecordSuccess(retry.Step{
		Name: fmt.Sprintf("CancelInterConnectUasl-%s", requestID),
		Rollback: func(rc context.Context) error {
			if s.ouranosProxyGW == nil {
				return nil
			}

			logger.LogInfo("WARN: Rollback: re-confirming cancelled external uasl reservation",
				"request_id", requestID,
				"url", adminGroup.URL,
				"ports_count", len(cancelledPorts))
			_, err := s.ouranosProxyGW.ConfirmUaslReservation(
				rc,
				handle.URL,
				handle.ID,
				true,
			)
			return err
		},
		Metadata: map[string]interface{}{
			"type":       "cancel_inter_connect_uasl",
			"request_id": requestID,
			"url":        adminGroup.URL,
		},
	})

	logger.LogInfo("External UASL reservation cancelled successfully",
		"request_id", requestID,
		"url", adminGroup.URL,
		"administrator_id", adminGroup.AdministratorID)

	if cancelResp != nil && cancelResp.Data != nil {
		return cancelResp.Data, nil
	}

	return nil, nil
}

func (s *InterconnectReservationService) FetchExternalAvailability(
	ctx context.Context,
	adminResolution *AvailabilityAdminResolution,
	sections []model.AvailabilitySection,
	vehicleIDs []string,
	portIDs []string,
) (
	externalItems []model.AvailabilityItem,
	externalVehicleItems []model.VehicleAvailabilityItem,
	externalPortItems []model.PortAvailabilityItem,
	internalSections []model.AvailabilitySection,
	internalVehicleIDs []string,
	internalPortIDs []string,
	err error,
) {

	if adminResolution == nil {
		return nil, nil, nil, sections, vehicleIDs, portIDs, nil
	}

	type externalGroup struct {
		Sections []model.AvailabilitySection
		UaslID   string
		URL      string
	}
	externalGroups := make(map[string]*externalGroup)

	for _, sec := range sections {
		if sec.UaslID == "" {
			internalSections = append(internalSections, sec)
			continue
		}
		eps, isExternal := adminResolution.ExternalServices[sec.UaslID]
		if !isExternal {
			internalSections = append(internalSections, sec)
			continue
		}
		url := eps.BaseURL
		g, exists := externalGroups[sec.UaslID]
		if !exists {
			g = &externalGroup{UaslID: sec.UaslID, URL: url}
			externalGroups[sec.UaslID] = g
		}
		g.Sections = append(g.Sections, sec)
	}

	internalVehicleIDs = make([]string, 0, len(vehicleIDs))
	internalVehicleIDMap := make(map[string]bool)
	if len(vehicleIDs) > 0 {
		vehicleResources, err := s.externalUaslResRepo.FindByResourceIDsAndType(vehicleIDs, model.ExternalResourceTypeVehicle)
		if err != nil {
			logger.LogInfo("FindByResourceIDsAndType failed for vehicles, treating all as external",
				"error", err.Error(),
			)

		} else {

			for _, res := range vehicleResources {
				internalVehicleIDMap[res.ExResourceID] = true
			}

			for _, vehicleID := range vehicleIDs {
				if internalVehicleIDMap[vehicleID] {
					internalVehicleIDs = append(internalVehicleIDs, vehicleID)
				}
			}
		}
	}

	internalPortIDs = make([]string, 0, len(portIDs))
	internalPortIDMap := make(map[string]bool)
	if len(portIDs) > 0 {
		portResources, err := s.externalUaslResRepo.FindByResourceIDsAndType(portIDs, model.ExternalResourceTypePort)
		if err != nil {
			logger.LogInfo("FindByResourceIDsAndType failed for ports, treating all as external",
				"error", err.Error(),
			)

		} else {

			for _, res := range portResources {
				internalPortIDMap[res.ExResourceID] = true
			}

			for _, portID := range portIDs {
				if internalPortIDMap[portID] {
					internalPortIDs = append(internalPortIDs, portID)
				}
			}
		}
	}

	externalVehicleIDs := make([]string, 0)
	externalPortIDs := make([]string, 0)
	for _, vehicleID := range vehicleIDs {
		if !internalVehicleIDMap[vehicleID] {
			externalVehicleIDs = append(externalVehicleIDs, vehicleID)
		}
	}
	for _, portID := range portIDs {
		if !internalPortIDMap[portID] {
			externalPortIDs = append(externalPortIDs, portID)
		}
	}

	externalVehicleItems = make([]model.VehicleAvailabilityItem, 0)
	externalPortItems = make([]model.PortAvailabilityItem, 0)

	for exUaslID, group := range externalGroups {
		url := group.URL
		if url == "" {
			logger.LogInfo("No URL for external group, skipping",
				"ex_uasl_id", exUaslID,
				"uasl_id", group.UaslID,
			)
			continue
		}

		sectionItems, vehicleItems, portItems, availErr := s.ouranosProxyGW.GetAvailability(ctx, url, group.Sections, externalVehicleIDs, externalPortIDs)
		if availErr != nil {
			logger.LogInfo("GetAvailability L2 proxy failed, skipping external group",
				"ex_uasl_id", exUaslID,
				"url", url,
				"error", availErr,
			)
			continue
		}

		externalItems = append(externalItems, sectionItems...)

		if len(vehicleItems) > 0 {
			logger.LogInfo("Received vehicle availability from external system",
				"ex_uasl_id", exUaslID,
				"count", len(vehicleItems),
			)
			externalVehicleItems = append(externalVehicleItems, vehicleItems...)
		}
		if len(portItems) > 0 {
			logger.LogInfo("Received port availability from external system",
				"ex_uasl_id", exUaslID,
				"count", len(portItems),
			)
			externalPortItems = append(externalPortItems, portItems...)
		}
	}

	return externalItems, externalVehicleItems, externalPortItems, internalSections, internalVehicleIDs, internalPortIDs, nil
}

func (s *InterconnectReservationService) EnrichListByOperatorWithExternalData(
	ctx context.Context,
	items []*model.UaslReservationListItem,
	adminResolution AvailabilityAdminResolution,
) error {
	if len(items) == 0 {
		return nil
	}

	resolvedURLGroups := make(map[string]*urlGroup)
	fetchSet := make(map[string]bool)

	for _, item := range items {
		hasEmptyPortName := false
		for i := range item.ExternalResources {
			resource := &item.ExternalResources[i]
			if resource.ResourceType == model.ExternalResourceTypePort && resource.ResourceName == "" {
				hasEmptyPortName = true
				break
			}
		}

		if !hasEmptyPortName || item.Parent == nil {
			continue
		}

		var lastChild *model.UaslReservation
		maxSequence := -1
		for _, child := range item.Children {
			if child != nil && child.Sequence != nil && *child.Sequence > maxSequence {
				maxSequence = *child.Sequence
				lastChild = child
			}
		}
		if lastChild == nil || lastChild.ExUaslID == nil || *lastChild.ExUaslID == "" {
			continue
		}

		exUaslID := *lastChild.ExUaslID
		endpoints, ok := adminResolution.ExternalServices[exUaslID]
		if !ok || endpoints.BaseURL == "" {
			continue
		}
		targetURL := endpoints.BaseURL

		reqIDStr := item.Parent.RequestID.ToString()
		if !fetchSet[reqIDStr] {
			if _, exists := resolvedURLGroups[targetURL]; !exists {
				resolvedURLGroups[targetURL] = &urlGroup{url: targetURL, requestIDs: []string{}}
			}
			resolvedURLGroups[targetURL].requestIDs = append(resolvedURLGroups[targetURL].requestIDs, reqIDStr)
			fetchSet[reqIDStr] = true
		}
	}

	if len(resolvedURLGroups) == 0 {
		return nil
	}

	var allExternalItems []model.ExternalReservationListItem
	for _, urlGrp := range resolvedURLGroups {
		externalItems, err := s.ouranosProxyGW.GetReservationsByRequestIDs(ctx, urlGrp.url, urlGrp.requestIDs)
		if err != nil {
			logger.LogError("Failed to get external reservations from L2",
				"url", urlGrp.url,
				"request_ids_count", len(urlGrp.requestIDs),
				"error", err.Error())
			continue
		}
		allExternalItems = append(allExternalItems, externalItems...)
	}
	if len(allExternalItems) == 0 {
		return nil
	}

	externalMap := make(map[string]*model.ExternalReservationListItem)
	for i := range allExternalItems {
		externalMap[allExternalItems[i].RequestID] = &allExternalItems[i]
	}

	for i := range items {
		item := items[i]
		if item.Parent == nil {
			continue
		}

		reqID := item.Parent.RequestID.ToString()
		extItem, exists := externalMap[reqID]
		if !exists {
			continue
		}

		for j := range item.ExternalResources {
			resource := &item.ExternalResources[j]
			if resource.ResourceType == model.ExternalResourceTypePort && resource.ResourceName == "" {
				externalPorts := make([]model.PortReservationElement, 0)
				if extItem.OriginReservation != nil {
					externalPorts = append(externalPorts, extItem.OriginReservation.Ports...)
				}
				for _, destination := range extItem.DestinationReservations {
					externalPorts = append(externalPorts, destination.Ports...)
				}
				for _, extPort := range externalPorts {
					if extPort.PortID == resource.ExResourceID {
						resource.ResourceName = extPort.Name
						break
					}
				}
			}
		}
	}

	return nil
}

func (s *InterconnectReservationService) EnrichListAdminWithExternalData(
	ctx context.Context,
	items []*model.UaslReservationListItem,
	adminResolution AvailabilityAdminResolution,
) ([]model.ExternalReservationListItem, error) {
	if len(items) == 0 {
		return nil, nil
	}

	type uaslGroup struct {
		requestIDs []string
		exUaslID   string
	}
	uaslMap := make(map[string]*uaslGroup)

	for _, item := range items {
		if item == nil || item.Parent == nil {
			continue
		}
		parent := item.Parent

		exUaslID := ""
		if parent.ExUaslID != nil && *parent.ExUaslID != "" {
			exUaslID = *parent.ExUaslID
		}

		if exUaslID == "" || adminResolution.ExternalServices[exUaslID].BaseURL == "" {
			maxSequence := -1
			for _, child := range item.Children {
				if child == nil || child.ExUaslID == nil || *child.ExUaslID == "" {
					continue
				}
				if child.Sequence != nil && *child.Sequence > maxSequence {
					maxSequence = *child.Sequence
					exUaslID = *child.ExUaslID
				}
			}
		}

		if exUaslID == "" {
			continue
		}

		if _, ok := uaslMap[exUaslID]; !ok {
			uaslMap[exUaslID] = &uaslGroup{
				requestIDs: []string{},
				exUaslID:   exUaslID,
			}
		}
		uaslMap[exUaslID].requestIDs = append(uaslMap[exUaslID].requestIDs, parent.RequestID.ToString())
	}

	if len(uaslMap) == 0 {
		return nil, nil
	}

	uaslIDToURL := make(map[string]string)
	for exUaslID, endpoints := range adminResolution.ExternalServices {
		if endpoints.BaseURL != "" {
			uaslIDToURL[exUaslID] = endpoints.BaseURL
		}
	}

	resolvedURLGroups := make(map[string]*urlGroup)
	for _, ug := range uaslMap {
		url, ok := uaslIDToURL[ug.exUaslID]
		if !ok || url == "" {
			continue
		}
		if _, exists := resolvedURLGroups[url]; !exists {
			resolvedURLGroups[url] = &urlGroup{url: url, requestIDs: []string{}}
		}
		resolvedURLGroups[url].requestIDs = append(resolvedURLGroups[url].requestIDs, ug.requestIDs...)
	}

	if len(resolvedURLGroups) == 0 {
		return nil, nil
	}

	var allExternalItems []model.ExternalReservationListItem
	for _, urlGrp := range resolvedURLGroups {
		externalItems, err := s.ouranosProxyGW.GetReservationsByRequestIDs(ctx, urlGrp.url, urlGrp.requestIDs)
		if err != nil {
			logger.LogError("Failed to get external reservations from L2",
				"url", urlGrp.url,
				"request_ids_count", len(urlGrp.requestIDs),
				"error", err.Error())
			continue
		}
		allExternalItems = append(allExternalItems, externalItems...)
	}
	return allExternalItems, nil
}

func (s *InterconnectReservationService) MergeListAdminLocalAndExternal(
	localItem *model.UaslReservationListItemMsg,
	externalItem *model.UaslReservationListItemMsg,
) *model.UaslReservationListItemMsg {
	localRequestID := ""
	externalRequestID := ""
	if localItem != nil && localItem.ParentUaslReservation != nil {
		localRequestID = localItem.ParentUaslReservation.RequestID
	}
	if externalItem != nil && externalItem.ParentUaslReservation != nil {
		externalRequestID = externalItem.ParentUaslReservation.RequestID
	}
	logger.LogInfo("MergeListAdminLocalAndExternal started",
		"local_request_id", localRequestID,
		"external_request_id", externalRequestID,
	)

	if localItem == nil {
		logger.LogInfo("MergeListAdminLocalAndExternal: local item is nil, using external item")
		return externalItem
	}
	if externalItem == nil {
		logger.LogInfo("MergeListAdminLocalAndExternal: external item is nil, using local item")
		return localItem
	}

	merged := &model.UaslReservationListItemMsg{
		ParentUaslReservation: mergeParentReservation(localItem.ParentUaslReservation, externalItem.ParentUaslReservation),
		ChildUaslReservations: mergeChildReservations(localItem.ChildUaslReservations, externalItem.ChildUaslReservations),
		Vehicles:              mergeVehicleElements(localItem.Vehicles, externalItem.Vehicles),
		Ports:                 mergePortElements(localItem.Ports, externalItem.Ports),
		FlightPurpose:         localItem.FlightPurpose,
	}
	if merged.FlightPurpose == "" {
		merged.FlightPurpose = externalItem.FlightPurpose
	}
	destCount := 0
	if merged.ParentUaslReservation != nil {
		destCount = len(merged.ParentUaslReservation.DestinationReservations)
	}
	logger.LogInfo("MergeListAdminLocalAndExternal completed",
		"request_id", func() string {
			if merged.ParentUaslReservation != nil {
				return merged.ParentUaslReservation.RequestID
			}
			return ""
		}(),
		"children_count", len(merged.ChildUaslReservations),
		"vehicles_count", len(merged.Vehicles),
		"ports_count", len(merged.Ports),
		"destination_reservations_count", destCount,
	)
	return merged
}

func mergeParentReservation(localParent, externalParent *model.UaslReservationMessage) *model.UaslReservationMessage {
	if localParent == nil {
		return cloneUaslReservationMessage(externalParent)
	}
	if externalParent == nil {
		return cloneUaslReservationMessage(localParent)
	}

	merged := cloneUaslReservationMessage(localParent)
	if merged.ID == "" {
		merged.ID = externalParent.ID
	}
	if merged.RequestID == "" {
		merged.RequestID = externalParent.RequestID
	}
	if merged.ExReservedBy == "" {
		merged.ExReservedBy = externalParent.ExReservedBy
	}
	if merged.Status == "" {
		merged.Status = externalParent.Status
	}
	if merged.FixedAt == nil {
		merged.FixedAt = externalParent.FixedAt
	}
	if merged.EstimatedAt == nil {
		merged.EstimatedAt = externalParent.EstimatedAt
	}
	if merged.UpdatedAt.IsZero() {
		merged.UpdatedAt = externalParent.UpdatedAt
	}
	if merged.ExUaslID == "" {
		merged.ExUaslID = externalParent.ExUaslID
	}
	if merged.ExAdministratorID == "" {
		merged.ExAdministratorID = externalParent.ExAdministratorID
	}

	destMap := make(map[string]*model.DestinationReservationInfo)
	destOrder := make([]string, 0, len(merged.DestinationReservations)+len(externalParent.DestinationReservations))
	for _, d := range merged.DestinationReservations {
		if d == nil {
			continue
		}
		key := d.ReservationID
		if key == "" {
			logger.LogError("mergeParentReservation: destination reservation id is empty in local data")
			continue
		}
		if _, exists := destMap[key]; !exists {
			destOrder = append(destOrder, key)
			destMap[key] = &model.DestinationReservationInfo{
				ReservationID:     d.ReservationID,
				ExUaslID:          d.ExUaslID,
				ExAdministratorID: d.ExAdministratorID,
			}
		}
	}
	for _, d := range externalParent.DestinationReservations {
		if d == nil {
			continue
		}
		key := d.ReservationID
		if key == "" {
			logger.LogError("mergeParentReservation: destination reservation id is empty in external data")
			continue
		}
		if existing, exists := destMap[key]; exists {
			if existing.ReservationID == "" {
				existing.ReservationID = d.ReservationID
			}
			if existing.ExUaslID == "" {
				existing.ExUaslID = d.ExUaslID
			}
			if existing.ExAdministratorID == "" {
				existing.ExAdministratorID = d.ExAdministratorID
			}
			continue
		}
		destOrder = append(destOrder, key)
		destMap[key] = &model.DestinationReservationInfo{
			ReservationID:     d.ReservationID,
			ExUaslID:          d.ExUaslID,
			ExAdministratorID: d.ExAdministratorID,
		}
	}
	merged.DestinationReservations = make([]*model.DestinationReservationInfo, 0, len(destOrder))
	for _, key := range destOrder {
		merged.DestinationReservations = append(merged.DestinationReservations, destMap[key])
	}

	return merged
}

func mergeChildReservations(localChildren, externalChildren []*model.UaslReservationMessage) []*model.UaslReservationMessage {
	childMap := make(map[string]*model.UaslReservationMessage)
	order := make([]string, 0, len(localChildren)+len(externalChildren))

	addOrMerge := func(src *model.UaslReservationMessage) {
		if src == nil {
			return
		}
		key := src.ExUaslSectionID
		if key == "" {
			logger.LogError("mergeChildReservations: ex_uasl_section_id is empty")
			return
		}
		if existing, exists := childMap[key]; exists {
			if existing.ID == "" {
				existing.ID = src.ID
			}
			if existing.ExUaslID == "" {
				existing.ExUaslID = src.ExUaslID
			}
			if existing.ExUaslSectionID == "" {
				existing.ExUaslSectionID = src.ExUaslSectionID
			}
			if existing.ExAdministratorID == "" {
				existing.ExAdministratorID = src.ExAdministratorID
			}
			if existing.StartAt.IsZero() {
				existing.StartAt = src.StartAt
			}
			if existing.EndAt.IsZero() {
				existing.EndAt = src.EndAt
			}
			if existing.Sequence == 0 {
				existing.Sequence = src.Sequence
			}
			if existing.Amount == 0 {
				existing.Amount = src.Amount
			}
			return
		}
		order = append(order, key)
		childMap[key] = cloneUaslReservationMessage(src)
	}

	for _, c := range localChildren {
		addOrMerge(c)
	}
	for _, c := range externalChildren {
		addOrMerge(c)
	}

	out := make([]*model.UaslReservationMessage, 0, len(order))
	for _, key := range order {
		out = append(out, childMap[key])
	}
	return out
}

func mergeVehicleElements(localVehicles, externalVehicles []*model.VehicleElement) []*model.VehicleElement {
	vehicleMap := make(map[string]*model.VehicleElement)
	order := make([]string, 0, len(localVehicles)+len(externalVehicles))

	addOrMerge := func(src *model.VehicleElement) {
		if src == nil {
			return
		}
		key := src.VehicleID
		if key == "" {
			return
		}
		if existing, exists := vehicleMap[key]; exists {
			if existing.Name == "" {
				existing.Name = src.Name
			}
			if existing.Amount == 0 {
				existing.Amount = src.Amount
			}
			return
		}
		order = append(order, key)
		vehicleMap[key] = &model.VehicleElement{
			VehicleID:     src.VehicleID,
			ReservationID: src.ReservationID,
			Name:          src.Name,
			StartAt:       src.StartAt,
			EndAt:         src.EndAt,
			Amount:        src.Amount,
		}
	}

	for _, v := range localVehicles {
		addOrMerge(v)
	}
	for _, v := range externalVehicles {
		addOrMerge(v)
	}

	out := make([]*model.VehicleElement, 0, len(order))
	for _, key := range order {
		out = append(out, vehicleMap[key])
	}
	return out
}

func mergePortElements(localPorts, externalPorts []*model.PortElement) []*model.PortElement {
	portMap := make(map[string]*model.PortElement)
	order := make([]string, 0, len(localPorts)+len(externalPorts))

	addOrMerge := func(src *model.PortElement) {
		if src == nil {
			return
		}
		portID := src.PortID
		if portID == "" {
			logger.LogError("mergePortElements: port_id is empty")
			return
		}
		key := fmt.Sprintf("%s_%d", portID, src.UsageType)
		if existing, exists := portMap[key]; exists {
			if existing.Name == "" {
				existing.Name = src.Name
			}
			if existing.Amount == 0 {
				existing.Amount = src.Amount
			}
			return
		}
		order = append(order, key)
		portMap[key] = &model.PortElement{
			PortID:        src.PortID,
			UsageType:     src.UsageType,
			StartAt:       src.StartAt,
			EndAt:         src.EndAt,
			ReservationID: src.ReservationID,
			Name:          src.Name,
			Amount:        src.Amount,
		}
	}

	for _, p := range localPorts {
		addOrMerge(p)
	}
	for _, p := range externalPorts {
		addOrMerge(p)
	}

	out := make([]*model.PortElement, 0, len(order))
	for _, key := range order {
		out = append(out, portMap[key])
	}
	return out
}

func cloneUaslReservationMessage(src *model.UaslReservationMessage) *model.UaslReservationMessage {
	if src == nil {
		return nil
	}
	dst := &model.UaslReservationMessage{
		ID:                          src.ID,
		RequestID:                   src.RequestID,
		ExReservedBy:                src.ExReservedBy,
		Status:                      src.Status,
		FixedAt:                     src.FixedAt,
		EstimatedAt:                 src.EstimatedAt,
		UpdatedAt:                   src.UpdatedAt,
		ExUaslID:                    src.ExUaslID,
		ExAdministratorID:           src.ExAdministratorID,
		ExUaslSectionID:             src.ExUaslSectionID,
		StartAt:                     src.StartAt,
		EndAt:                       src.EndAt,
		Sequence:                    src.Sequence,
		Amount:                      src.Amount,
		ConformityAssessmentResults: src.ConformityAssessmentResults,
	}
	if len(src.DestinationReservations) > 0 {
		dst.DestinationReservations = make([]*model.DestinationReservationInfo, 0, len(src.DestinationReservations))
		for _, d := range src.DestinationReservations {
			if d == nil {
				continue
			}
			dst.DestinationReservations = append(dst.DestinationReservations, &model.DestinationReservationInfo{
				ReservationID:     d.ReservationID,
				ExUaslID:          d.ExUaslID,
				ExAdministratorID: d.ExAdministratorID,
			})
		}
	}
	return dst
}

func (s *InterconnectReservationService) FetchFindByRequestIDExternalItem(
	ctx context.Context,
	requestID string,
	parent *model.UaslReservation,
	children []*model.UaslReservation,
	adminResolution AvailabilityAdminResolution,
) *model.ExternalReservationListItem {
	resolveTargetURL := func(resolution AvailabilityAdminResolution) string {
		if parent != nil && parent.ExUaslID != nil && *parent.ExUaslID != "" {
			if endpoints, ok := resolution.ExternalServices[*parent.ExUaslID]; ok &&
				endpoints.BaseURL != "" {
				return endpoints.BaseURL
			}
		}
		maxSequence := -1
		targetURL := ""
		for _, child := range children {
			if child == nil || child.ExUaslID == nil || *child.ExUaslID == "" {
				continue
			}
			endpoints, isExternal := resolution.ExternalServices[*child.ExUaslID]
			if !isExternal || endpoints.BaseURL == "" {
				continue
			}
			if child.Sequence != nil && *child.Sequence > maxSequence {
				maxSequence = *child.Sequence
				targetURL = endpoints.BaseURL
			}
		}
		return targetURL
	}

	targetURL := resolveTargetURL(adminResolution)
	if targetURL == "" {
		uniqueExternalUaslIDs := make(map[string]struct{})
		for _, child := range children {
			if child == nil || child.ExUaslID == nil || *child.ExUaslID == "" {
				continue
			}
			uniqueExternalUaslIDs[*child.ExUaslID] = struct{}{}
		}
		if len(uniqueExternalUaslIDs) > 0 {
			externalUaslIDs := make([]string, 0, len(uniqueExternalUaslIDs))
			for id := range uniqueExternalUaslIDs {
				externalUaslIDs = append(externalUaslIDs, id)
			}
			resolved := s.ResolveAdministratorsForAvailability(ctx, externalUaslIDs, []string{})
			if resolved != nil {
				targetURL = resolveTargetURL(*resolved)
			}
		}
	}
	if targetURL == "" {
		return nil
	}

	externalItems, err := s.ouranosProxyGW.GetReservationsByRequestIDs(ctx, targetURL, []string{requestID})
	if err != nil {
		logger.LogError("Failed to get external reservations from L2 in FetchFindByRequestIDExternalItem",
			"url", targetURL,
			"request_id", requestID,
			"error", err.Error())
		return nil
	}
	if len(externalItems) == 0 {
		return nil
	}

	for i := range externalItems {
		if externalItems[i].RequestID == requestID {
			return &externalItems[i]
		}
	}
	return &externalItems[0]
}

func (s *InterconnectReservationService) EstimateExternalUasl(
	ctx context.Context,
	externalReq model.ExternalEstimateRequest,
	adminResolution AvailabilityAdminResolution,
) (int32, error) {
	var totalAmount int32 = 0

	if len(externalReq.UaslSections) == 0 {
		return 0, nil
	}

	uaslIDToSections := make(map[string][]model.ExternalEstimateSectionRequest)
	sectionsWithoutUaslID := make([]model.ExternalEstimateSectionRequest, 0)

	for _, section := range externalReq.UaslSections {
		if section.UaslID != "" {
			uaslIDToSections[section.UaslID] = append(uaslIDToSections[section.UaslID], section)
		} else {
			sectionsWithoutUaslID = append(sectionsWithoutUaslID, section)
		}
	}

	for exUaslIDStr, sections := range uaslIDToSections {

		endpoints, ok := adminResolution.ExternalServices[exUaslIDStr]
		if !ok || endpoints.BaseURL == "" {
			logger.LogError("Discovery URL not found for external uasl",
				"ex_uasl_id", exUaslIDStr)
			return 0, fmt.Errorf("discovery URL not found for ex_uasl_id=%s", exUaslIDStr)
		}
		baseURL := endpoints.BaseURL

		uaslSpecificReq := model.ExternalEstimateRequest{
			UaslSections:   sections,
			Vehicles:       externalReq.Vehicles,
			Ports:          externalReq.Ports,
			IsInterConnect: true,
		}

		externalResp, err := s.ouranosProxyGW.EstimateUaslReservation(ctx, baseURL, uaslSpecificReq)
		if err != nil {
			logger.LogError("Failed to call external estimate API",
				"ex_uasl_id", exUaslIDStr,
				"base_url", baseURL,
				"error", err)
			return 0, fmt.Errorf("failed to call external estimate API for ex_uasl_id=%s: %w", exUaslIDStr, err)
		}

		totalAmount += externalResp.TotalAmount
	}

	logger.LogInfo("All external API calls completed",
		"total_external_amount", totalAmount,
		"uasl_count", len(uaslIDToSections))

	return totalAmount, nil
}

func (s *InterconnectReservationService) ClassifyInternalExternalUaslIDs(
	ctx context.Context,
	allUaslIDs []string,
) (internalUaslIDs []string, externalUaslIDs []string, err error) {
	if len(allUaslIDs) == 0 {
		return nil, nil, nil
	}

	internalAdmins, err := s.uaslAdminRepo.FindInternalAdministrators(ctx)
	if err != nil {
		logger.LogError("ClassifyInternalExternalUaslIDs: FindInternalAdministrators failed", "error", err)
		return nil, nil, fmt.Errorf("failed to find internal administrators: %w", err)
	}
	if len(internalAdmins) == 0 {
		logger.LogInfo("ClassifyInternalExternalUaslIDs: no internal administrator found, all IDs treated as external")
		return nil, allUaslIDs, nil
	}

	internalUaslIDSet := make(map[string]struct{})
	totalServiceCount := 0
	for _, admin := range internalAdmins {
		servicesList, err := admin.GetExternalServicesList()
		if err != nil {
			logger.LogError("ClassifyInternalExternalUaslIDs: failed to unmarshal ExternalServicesList", "error", err, "ex_administrator_id", admin.ExAdministratorID)
			continue
		}
		if servicesList == nil {
			continue
		}
		totalServiceCount += len(servicesList)
		for _, svc := range servicesList {
			if svc.ExUaslID != "" {
				internalUaslIDSet[svc.ExUaslID] = struct{}{}
			}
		}
	}

	internalUaslIDs = make([]string, 0)
	externalUaslIDs = make([]string, 0)

	for _, uaslID := range allUaslIDs {
		if _, isInternal := internalUaslIDSet[uaslID]; isInternal {
			internalUaslIDs = append(internalUaslIDs, uaslID)
		} else {
			externalUaslIDs = append(externalUaslIDs, uaslID)
		}
	}

	logger.LogInfo("ClassifyInternalExternalUaslIDs completed",
		"total_count", len(allUaslIDs),
		"internal_count", len(internalUaslIDs),
		"external_count", len(externalUaslIDs))

	return internalUaslIDs, externalUaslIDs, nil
}

func (s *InterconnectReservationService) resolveLatLngByInternalUaslIDs(ctx context.Context, internalUaslIDs []string) (lat, lng float64, ok bool) {
	if len(internalUaslIDs) == 0 {
		return 0, 0, false
	}

	defs, err := s.externalUaslDefRepo.FindByExUaslIDs(ctx, internalUaslIDs)
	if err != nil {
		logger.LogError("resolveLatLngByInternalUaslIDs: FindByExUaslIDs failed", "error", err)
		return 0, 0, false
	}

	var sumLat, sumLng float64
	var count int

	for _, def := range defs {
		if def == nil {
			continue
		}

		if def.Geometry.IsEmpty() {
			continue
		}

		la, lo, coordOK := converter.ExtractCentroidFromWKB(def.Geometry.ToBytes(), def.Geometry.ToString())
		if !coordOK {
			continue
		}
		sumLat += la
		sumLng += lo
		count++
	}

	if count == 0 {
		logger.LogInfo("resolveLatLngByInternalUaslIDs: no valid coordinates found",
			"internal_uasl_ids", internalUaslIDs)
		return 0, 0, false
	}

	return sumLat / float64(count), sumLng / float64(count), true
}

func (s *InterconnectReservationService) resolveAdminsByL4(
	ctx context.Context,
	uaslIDsToResolve []string,
	lat, lng float64,
) []*model.UaslAdministrator {
	if s.ouranosDiscoveryGW == nil || len(uaslIDsToResolve) == 0 {
		return nil
	}

	findReq := converter.ToFindResourceRequestFromLocation(lat, lng, 100.0)
	urls, findErr := s.ouranosDiscoveryGW.FindResourceFromDiscoveryService(ctx, findReq)
	if findErr != nil {
		logger.LogError("resolveAdminsByL4: FindResourceFromDiscoveryService failed",
			"lat", lat, "lng", lng,
			"error", findErr)
		return nil
	}

	if len(urls) == 0 {
		logger.LogInfo("resolveAdminsByL4: no URLs from L4")
		return nil
	}

	logger.LogInfo("resolveAdminsByL4: L4 returned URLs",
		"urls_count", len(urls))

	internalAdminSet := make(map[string]struct{})
	internalAdmins, internalErr := s.uaslAdminRepo.FindInternalAdministrators(ctx)
	if internalErr != nil {
		logger.LogError("resolveAdminsByL4: FindInternalAdministrators failed", "error", internalErr)
	} else {
		for _, a := range internalAdmins {
			if a.ExAdministratorID != "" {
				internalAdminSet[a.ExAdministratorID] = struct{}{}
			}
		}
	}

	adminServicesMap := make(map[string]model.ExternalServicesList)

	adminInfoMap := make(map[string]*model.UaslAdministrator)

	for i := range urls {
		urlEntry := urls[i]
		domainAppID := urlEntry.DomainAppID
		if domainAppID == "" {
			continue
		}

		if s.uaslDesignGateway == nil || urlEntry.BaseURL == "" {
			logger.LogInfo("resolveAdminsByL4: no UaslBaseURL, skipping definition API",
				"domain_app_id", domainAppID)
			continue
		}

		bulk, defErr := s.uaslDesignGateway.FetchUaslList(ctx, urlEntry.BaseURL, false)
		if defErr != nil {
			logger.LogError("resolveAdminsByL4: FetchUaslList failed",
				"domain_app_id", domainAppID,
				"uasl_base_url", urlEntry.BaseURL,
				"error", defErr)
			continue
		}

		for j := range bulk.Administrators {
			adminCopy := bulk.Administrators[j]
			exAdminID := adminCopy.ExAdministratorID
			if exAdminID == "" {
				continue
			}

			if _, isInternal := internalAdminSet[exAdminID]; isInternal {
				logger.LogInfo("resolveAdminsByL4: administrator is internal, skipping upsert",
					"domain_app_id", domainAppID,
					"ex_administrator_id", exAdminID)
				continue
			}

			if _, exists := adminInfoMap[exAdminID]; !exists {
				c := adminCopy
				adminInfoMap[exAdminID] = &c
			}

			if _, exists := adminServicesMap[exAdminID]; !exists {
				adminServicesMap[exAdminID] = model.ExternalServicesList{}
			}
		}

		endpoints := converter.ToExternalServiceEndpoints(urlEntry)
		existingSetPerAdmin := make(map[string]map[string]struct{})
		for exAdminID, servicesList := range adminServicesMap {
			set := make(map[string]struct{}, len(servicesList))
			for _, svc := range servicesList {
				set[svc.ExUaslID] = struct{}{}
			}
			existingSetPerAdmin[exAdminID] = set
		}

		for _, def := range bulk.Definitions {
			if !def.ExUaslID.Valid || def.ExUaslID.String == nil || *def.ExUaslID.String == "" {
				continue
			}
			defUaslID := *def.ExUaslID.String
			exAdminID := def.ExAdministratorID
			if exAdminID == "" {
				continue
			}

			if _, isInternal := internalAdminSet[exAdminID]; isInternal {
				continue
			}

			if _, exists := adminServicesMap[exAdminID]; !exists {
				continue
			}

			if _, exists := existingSetPerAdmin[exAdminID][defUaslID]; !exists {
				adminServicesMap[exAdminID] = append(adminServicesMap[exAdminID], model.ExternalService{
					ExUaslID: defUaslID,
					Services: endpoints,
				})
				existingSetPerAdmin[exAdminID][defUaslID] = struct{}{}
			}
		}
	}

	allAdminsFromL4 := make([]*model.UaslAdministrator, 0, len(adminServicesMap))

	for exAdminID, servicesList := range adminServicesMap {
		jsonBytes, marshalErr := json.Marshal(servicesList)
		if marshalErr != nil {
			logger.LogError("resolveAdminsByL4: failed to marshal ExternalServicesList",
				"ex_administrator_id", exAdminID, "error", marshalErr)
			continue
		}

		info, ok := adminInfoMap[exAdminID]
		if !ok || info.ExAdministratorID == "" {
			logger.LogInfo("resolveAdminsByL4: skipping admin with missing ExAdministratorID",
				"ex_administrator_id", exAdminID,
				"has_info", ok)
			continue
		}

		rawMsg := json.RawMessage(jsonBytes)
		adminFromL4 := &model.UaslAdministrator{
			ExAdministratorID: info.ExAdministratorID,
			Name:              info.Name,
			IsInternal:        false,
			ExternalServices: value.NullJSON{
				Valid: true,
				JSON:  &rawMsg,
			},
		}
		if info.BusinessNumber.Valid {
			adminFromL4.BusinessNumber = info.BusinessNumber
		}

		allAdminsFromL4 = append(allAdminsFromL4, adminFromL4)
	}

	if len(allAdminsFromL4) > 0 {

		exAdminIDsToUpsert := make([]string, 0, len(allAdminsFromL4))
		for _, a := range allAdminsFromL4 {
			exAdminIDsToUpsert = append(exAdminIDsToUpsert, a.ExAdministratorID)
		}
		existingForMerge, mergeErr := s.uaslAdminRepo.FindByExAdministratorIDs(ctx, exAdminIDsToUpsert)
		if mergeErr != nil {
			logger.LogError("resolveAdminsByL4: FindByExAdministratorIDs for merge failed, skipping upsert to protect existing data",
				"error", mergeErr)
			return allAdminsFromL4
		}

		existingByAdminID := make(map[string]*model.UaslAdministrator, len(existingForMerge))
		for _, existing := range existingForMerge {
			existingByAdminID[existing.ExAdministratorID] = existing
		}

		for _, a := range allAdminsFromL4 {
			existing, ok := existingByAdminID[a.ExAdministratorID]
			if !ok {

				continue
			}
			existingList, err := existing.GetExternalServicesList()
			if err != nil || existingList == nil {

				continue
			}
			l4List, err := a.GetExternalServicesList()
			if err != nil || l4List == nil {
				continue
			}
			l4UaslIDSet := make(map[string]struct{}, len(l4List))
			for _, svc := range l4List {
				l4UaslIDSet[svc.ExUaslID] = struct{}{}
			}
			merged := make(model.ExternalServicesList, len(l4List))
			copy(merged, l4List)
			for _, svc := range existingList {
				if _, exists := l4UaslIDSet[svc.ExUaslID]; !exists {
					merged = append(merged, svc)
				}
			}
			mergedBytes, marshalErr := json.Marshal(merged)
			if marshalErr != nil {
				logger.LogError("resolveAdminsByL4: failed to marshal merged ExternalServicesList",
					"ex_administrator_id", a.ExAdministratorID, "error", marshalErr)
				continue
			}
			rawMsg := json.RawMessage(mergedBytes)
			a.ExternalServices = value.NullJSON{Valid: true, JSON: &rawMsg}
			logger.LogInfo("resolveAdminsByL4: merged ExternalServicesList with existing DB data",
				"ex_administrator_id", a.ExAdministratorID,
				"existing_count", len(existingList),
				"l4_count", len(l4List),
				"merged_count", len(merged))
		}

		if insertErr := s.uaslAdminRepo.UpsertBatch(ctx, allAdminsFromL4); insertErr != nil {
			logger.LogError("resolveAdminsByL4: UpsertBatch failed",
				"count", len(allAdminsFromL4), "error", insertErr)
			return allAdminsFromL4
		}
		logger.LogInfo("resolveAdminsByL4: administrators upserted with merge",
			"count", len(allAdminsFromL4))
	} else {
		logger.LogInfo("resolveAdminsByL4: no new administrators to insert (all already exist)")
	}

	logger.LogInfo("resolveAdminsByL4: completed",
		"new_inserts", len(allAdminsFromL4))

	return allAdminsFromL4
}

func (s *InterconnectReservationService) resolveAdminsByFetchUaslByID(
	ctx context.Context,
	externalUaslIDs []string,
) []*model.UaslAdministrator {
	if s.uaslDesignGateway == nil || len(externalUaslIDs) == 0 {
		return nil
	}

	internalAdminSet := make(map[string]struct{})
	internalAdmins, internalErr := s.uaslAdminRepo.FindInternalAdministrators(ctx)
	if internalErr != nil {
		logger.LogError("resolveAdminsByFetchUaslByID: FindInternalAdministrators failed", "error", internalErr)
	} else {
		for _, a := range internalAdmins {
			if a.ExAdministratorID != "" {
				internalAdminSet[a.ExAdministratorID] = struct{}{}
			}
		}
	}

	adminServicesMap := make(map[string]model.ExternalServicesList)

	adminInfoMap := make(map[string]*model.UaslAdministrator)

	adminBaseURLMap := make(map[string]string)

	for _, exUaslID := range externalUaslIDs {

		bulk, fetchErr := s.uaslDesignGateway.FetchUaslByID(ctx, "", exUaslID, false)
		if fetchErr != nil {
			logger.LogError("resolveAdminsByFetchUaslByID: FetchUaslByID failed",
				"ex_uasl_id", exUaslID,
				"error", fetchErr)
			continue
		}

		if bulk == nil {
			logger.LogInfo("resolveAdminsByFetchUaslByID: no data returned",
				"ex_uasl_id", exUaslID)
			continue
		}

		for j := range bulk.Administrators {
			adminCopy := bulk.Administrators[j]
			exAdminID := adminCopy.ExAdministratorID
			if exAdminID == "" {
				continue
			}

			if _, isInternal := internalAdminSet[exAdminID]; isInternal {
				logger.LogInfo("resolveAdminsByFetchUaslByID: administrator is internal, skipping upsert",
					"ex_uasl_id", exUaslID,
					"ex_administrator_id", exAdminID)
				continue
			}

			if _, exists := adminInfoMap[exAdminID]; !exists {
				c := adminCopy
				adminInfoMap[exAdminID] = &c
			}

			if _, exists := adminServicesMap[exAdminID]; !exists {
				adminServicesMap[exAdminID] = model.ExternalServicesList{}
			}

			existingServices, err := adminCopy.GetExternalServicesList()
			if err != nil {
				logger.LogError("resolveAdminsByFetchUaslByID: failed to get ExternalServicesList",
					"ex_administrator_id", exAdminID,
					"error", err)
				continue
			}

			if existingServices != nil {
				existingSet := make(map[string]struct{})
				for _, svc := range adminServicesMap[exAdminID] {
					existingSet[svc.ExUaslID] = struct{}{}
				}

				for _, svc := range existingServices {
					if svc.ExUaslID != "" {
						if _, exists := existingSet[svc.ExUaslID]; !exists {
							adminServicesMap[exAdminID] = append(adminServicesMap[exAdminID], svc)
							existingSet[svc.ExUaslID] = struct{}{}

							if _, hasBaseURL := adminBaseURLMap[exAdminID]; !hasBaseURL && svc.Services.BaseURL != "" {
								adminBaseURLMap[exAdminID] = svc.Services.BaseURL
							}
						}
					}
				}
			}
		}
	}

	for _, requestedUaslID := range externalUaslIDs {
		found := false

		for _, servicesList := range adminServicesMap {
			for _, svc := range servicesList {
				if svc.ExUaslID == requestedUaslID {
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found {

			if s.uaslAdminRepo != nil {
				existingAdmins, err := s.uaslAdminRepo.FindByExUaslIDs(ctx, []string{requestedUaslID})
				if err == nil && len(existingAdmins) > 0 {
					for _, existingAdmin := range existingAdmins {
						existingServices, err := existingAdmin.GetExternalServicesList()
						if err != nil || existingServices == nil {
							continue
						}
						for _, svc := range existingServices {
							if svc.ExUaslID == requestedUaslID {
								exAdminID := existingAdmin.ExAdministratorID

								baseURL := svc.Services.BaseURL
								if baseURL == "" {

									if representativeURL, ok := adminBaseURLMap[exAdminID]; ok {
										baseURL = representativeURL
									}
								}

								if _, exists := adminServicesMap[exAdminID]; !exists {
									adminServicesMap[exAdminID] = model.ExternalServicesList{}
								}
								adminServicesMap[exAdminID] = append(adminServicesMap[exAdminID], model.ExternalService{
									ExUaslID: requestedUaslID,
									Services: model.ExternalServiceEndpoints{BaseURL: baseURL},
								})

								logger.LogInfo("resolveAdminsByFetchUaslByID: added missing uasl ID with existing baseURL",
									"ex_uasl_id", requestedUaslID,
									"ex_administrator_id", exAdminID,
									"base_url", baseURL)
								found = true
								break
							}
						}
						if found {
							break
						}
					}
				}
			}
		}
	}

	allAdmins := make([]*model.UaslAdministrator, 0, len(adminServicesMap))

	for exAdminID, servicesList := range adminServicesMap {
		jsonBytes, marshalErr := json.Marshal(servicesList)
		if marshalErr != nil {
			logger.LogError("resolveAdminsByFetchUaslByID: failed to marshal ExternalServicesList",
				"ex_administrator_id", exAdminID, "error", marshalErr)
			continue
		}

		info, ok := adminInfoMap[exAdminID]
		if !ok || info.ExAdministratorID == "" {
			logger.LogInfo("resolveAdminsByFetchUaslByID: skipping admin with missing ExAdministratorID",
				"ex_administrator_id", exAdminID,
				"has_info", ok)
			continue
		}

		rawMsg := json.RawMessage(jsonBytes)
		admin := &model.UaslAdministrator{
			ExAdministratorID: info.ExAdministratorID,
			Name:              info.Name,
			IsInternal:        false,
			ExternalServices: value.NullJSON{
				Valid: true,
				JSON:  &rawMsg,
			},
		}
		if info.BusinessNumber.Valid {
			admin.BusinessNumber = info.BusinessNumber
		}

		allAdmins = append(allAdmins, admin)
	}

	if len(allAdmins) > 0 {

		exAdminIDsToUpsert := make([]string, 0, len(allAdmins))
		for _, a := range allAdmins {
			exAdminIDsToUpsert = append(exAdminIDsToUpsert, a.ExAdministratorID)
		}
		existingForMerge, mergeErr := s.uaslAdminRepo.FindByExAdministratorIDs(ctx, exAdminIDsToUpsert)
		if mergeErr != nil {
			logger.LogError("resolveAdminsByFetchUaslByID: FindByExAdministratorIDs for merge failed, skipping upsert to protect existing data",
				"error", mergeErr)
			return allAdmins
		}

		existingByAdminID := make(map[string]*model.UaslAdministrator, len(existingForMerge))
		for _, existing := range existingForMerge {
			existingByAdminID[existing.ExAdministratorID] = existing
		}

		for _, a := range allAdmins {
			existing, ok := existingByAdminID[a.ExAdministratorID]
			if !ok {

				continue
			}
			existingList, err := existing.GetExternalServicesList()
			if err != nil || existingList == nil {

				continue
			}
			newList, err := a.GetExternalServicesList()
			if err != nil || newList == nil {
				continue
			}
			newUaslIDSet := make(map[string]struct{}, len(newList))
			for _, svc := range newList {
				newUaslIDSet[svc.ExUaslID] = struct{}{}
			}
			merged := make(model.ExternalServicesList, len(newList))
			copy(merged, newList)
			for _, svc := range existingList {
				if _, exists := newUaslIDSet[svc.ExUaslID]; !exists {
					merged = append(merged, svc)
				}
			}
			mergedBytes, marshalErr := json.Marshal(merged)
			if marshalErr != nil {
				logger.LogError("resolveAdminsByFetchUaslByID: failed to marshal merged ExternalServicesList",
					"ex_administrator_id", a.ExAdministratorID, "error", marshalErr)
				continue
			}
			rawMsg := json.RawMessage(mergedBytes)
			a.ExternalServices = value.NullJSON{Valid: true, JSON: &rawMsg}
			logger.LogInfo("resolveAdminsByFetchUaslByID: merged ExternalServicesList with existing DB data",
				"ex_administrator_id", a.ExAdministratorID,
				"existing_count", len(existingList),
				"new_count", len(newList),
				"merged_count", len(merged))
		}

		if upsertErr := s.uaslAdminRepo.UpsertBatch(ctx, allAdmins); upsertErr != nil {
			logger.LogError("resolveAdminsByFetchUaslByID: UpsertBatch failed",
				"count", len(allAdmins), "error", upsertErr)
			return allAdmins
		}
		logger.LogInfo("resolveAdminsByFetchUaslByID: administrators upserted with merge",
			"count", len(allAdmins))
	} else {
		logger.LogInfo("resolveAdminsByFetchUaslByID: no administrators to insert")
	}

	logger.LogInfo("resolveAdminsByFetchUaslByID: completed",
		"administrators_count", len(allAdmins))

	return allAdmins
}

type AvailabilityAdminResolution struct {
	ExternalServices        map[string]model.ExternalServiceEndpoints
	UaslIDToAdministratorID map[string]string
}

type ReservationParentOrigin struct {
	UaslID          string
	AdministratorID string
}

func (s *InterconnectReservationService) ResolveAdminResolutionByParentOrigins(
	ctx context.Context,
	origins []ReservationParentOrigin,
) *AvailabilityAdminResolution {
	result := &AvailabilityAdminResolution{
		ExternalServices:        make(map[string]model.ExternalServiceEndpoints),
		UaslIDToAdministratorID: make(map[string]string),
	}

	if len(origins) == 0 {
		return result
	}

	var internalAdminIDs map[string]struct{}
	if s.uaslAdminRepo != nil {
		admins, err := s.uaslAdminRepo.FindInternalAdministrators(ctx)
		if err != nil {
			logger.LogError("ResolveAdminResolutionByParentOrigins: failed to find internal administrators", "error", err)
		} else {
			internalAdminIDs = make(map[string]struct{}, len(admins))
			for _, a := range admins {
				if a.ExAdministratorID != "" {
					internalAdminIDs[a.ExAdministratorID] = struct{}{}
				}
			}
		}
	}

	internalUaslIDSet := make(map[string]struct{})
	externalUaslIDSet := make(map[string]struct{})

	for _, origin := range origins {
		if origin.UaslID == "" {
			continue
		}

		if len(internalAdminIDs) == 0 || origin.AdministratorID == "" {
			externalUaslIDSet[origin.UaslID] = struct{}{}
		} else if _, isInternal := internalAdminIDs[origin.AdministratorID]; isInternal {
			internalUaslIDSet[origin.UaslID] = struct{}{}
		} else {
			externalUaslIDSet[origin.UaslID] = struct{}{}
		}
	}

	internalUaslIDs := make([]string, 0, len(internalUaslIDSet))
	for id := range internalUaslIDSet {
		internalUaslIDs = append(internalUaslIDs, id)
	}

	externalUaslIDs := make([]string, 0, len(externalUaslIDSet))
	for id := range externalUaslIDSet {
		externalUaslIDs = append(externalUaslIDs, id)
	}

	return s.ResolveAdministratorsForAvailability(ctx, externalUaslIDs, internalUaslIDs)
}

func (s *InterconnectReservationService) ResolveAdministratorsForAvailability(
	ctx context.Context,
	externalUaslIDs []string,
	internalUaslIDs []string,
) *AvailabilityAdminResolution {
	result := &AvailabilityAdminResolution{
		ExternalServices:        make(map[string]model.ExternalServiceEndpoints),
		UaslIDToAdministratorID: make(map[string]string),
	}

	existingAdmins, err := s.uaslAdminRepo.FindByExUaslIDs(ctx, externalUaslIDs)
	if err != nil {
		logger.LogError("ResolveAdministratorsForAvailability: FindByExUaslIDs failed, proceeding with L4 for all",
			"error", err)
		existingAdmins = nil
	}

	resolvedIDs := make(map[string]struct{}, len(externalUaslIDs))
	for _, admin := range existingAdmins {
		servicesList, err := admin.GetExternalServicesList()
		if err != nil {
			logger.LogError("ResolveAdministratorsForAvailability: failed to unmarshal external_services",
				"ex_administrator_id", admin.ExAdministratorID, "error", err)
			continue
		}
		if servicesList == nil {
			continue
		}
		for _, svc := range servicesList {
			if svc.ExUaslID != "" {
				resolvedIDs[svc.ExUaslID] = struct{}{}
				result.ExternalServices[svc.ExUaslID] = svc.Services
				result.UaslIDToAdministratorID[svc.ExUaslID] = admin.ExAdministratorID
			}
		}
	}

	externalIDsNeedingL4 := make([]string, 0, len(externalUaslIDs))
	for _, id := range externalUaslIDs {
		if _, found := resolvedIDs[id]; !found {
			externalIDsNeedingL4 = append(externalIDsNeedingL4, id)
		}
	}

	if len(externalIDsNeedingL4) > 0 {
		var admins []*model.UaslAdministrator

		if len(internalUaslIDs) > 0 && s.ouranosDiscoveryGW != nil {

			lat, lng, coordOK := s.resolveLatLngByInternalUaslIDs(ctx, internalUaslIDs)
			if !coordOK {
				logger.LogInfo("ResolveAdministratorsForAvailability: failed to resolve coordinates from internal uasl IDs, using default (0, 0)",
					"internal_uasl_ids", internalUaslIDs)
				lat, lng = 0, 0
			}

			logger.LogInfo("ResolveAdministratorsForAvailability: calling L4",
				"external_uasl_ids_needing_l4", externalIDsNeedingL4,
				"count", len(externalIDsNeedingL4),
				"lat", lat, "lng", lng)

			admins = s.resolveAdminsByL4(ctx, externalIDsNeedingL4, lat, lng)
		} else if s.uaslDesignGateway != nil {

			logger.LogInfo("ResolveAdministratorsForAvailability: using FetchUaslByID for direct resolution",
				"reason", func() string {
					if len(internalUaslIDs) == 0 {
						return "no internal uasl IDs"
					}
					return "L4 discovery service unavailable"
				}(),
				"external_uasl_ids_needing_resolution", externalIDsNeedingL4,
				"count", len(externalIDsNeedingL4))

			admins = s.resolveAdminsByFetchUaslByID(ctx, externalIDsNeedingL4)
		}

		for _, admin := range admins {
			servicesList, err := admin.GetExternalServicesList()
			if err != nil {
				logger.LogError("ResolveAdministratorsForAvailability: failed to unmarshal external_services",
					"ex_administrator_id", admin.ExAdministratorID, "error", err)
				continue
			}
			if servicesList == nil {
				continue
			}
			for _, svc := range servicesList {
				if svc.ExUaslID != "" {
					result.ExternalServices[svc.ExUaslID] = svc.Services
					result.UaslIDToAdministratorID[svc.ExUaslID] = admin.ExAdministratorID
				}
			}
		}
	}

	logger.LogInfo("ResolveAdministratorsForAvailability completed",
		"external_services_count", len(result.ExternalServices),
		"uasl_id_to_admin_count", len(result.UaslIDToAdministratorID))

	return result
}

func (s *InterconnectReservationService) BuildAdministratorGroupsFromReservedChildren(
	ctx context.Context,
	children []*model.UaslReservation,
	callerName string,
) (map[string]*AdministratorGroup, error) {
	externalGroups := make(map[string]*AdministratorGroup)

	allUaslIds := make([]string, 0, len(children))
	for _, child := range children {
		if child.ExUaslID != nil && *child.ExUaslID != "" {
			allUaslIds = append(allUaslIds, *child.ExUaslID)
		}
	}

	if len(allUaslIds) == 0 {
		logger.LogInfo(callerName + ": no external UASL IDs found in children")
		return externalGroups, nil
	}

	internalUaslIDs, externalUaslIDs, classifyErr := s.ClassifyInternalExternalUaslIDs(ctx, allUaslIds)
	if classifyErr != nil {
		logger.LogError(callerName+": failed to classify uasl ids", "error", classifyErr)
		return externalGroups, classifyErr
	}

	if len(externalUaslIDs) == 0 {
		logger.LogInfo(callerName + ": no external UASL IDs to process")
		return externalGroups, nil
	}

	adminResolution := s.ResolveAdministratorsForAvailability(ctx, externalUaslIDs, internalUaslIDs)
	logger.LogInfo(callerName+": resolved administrator URLs",
		"external_uasl_ids_count", len(externalUaslIDs),
		"internal_uasl_ids_count", len(internalUaslIDs))

	if adminResolution == nil {
		logger.LogInfo(callerName + ": no administrator resolution available")
		return externalGroups, nil
	}

	childrenByUaslID := make(map[string][]*model.UaslReservation)
	for _, child := range children {
		if child.ExUaslID != nil && *child.ExUaslID != "" {
			childrenByUaslID[*child.ExUaslID] = append(childrenByUaslID[*child.ExUaslID], child)
		}
	}

	for exUaslID, endpoints := range adminResolution.ExternalServices {
		if endpoints.BaseURL == "" {
			continue
		}
		uaslURL := endpoints.BaseURL
		childList := childrenByUaslID[exUaslID]
		if len(childList) == 0 {
			continue
		}

		adminID := ""
		if childList[0].ExAdministratorID != nil {
			adminID = *childList[0].ExAdministratorID
		}
		if adminID == "" {
			continue
		}

		if group, exists := externalGroups[adminID]; exists {
			group.ChildDomains = append(group.ChildDomains, childList...)
		} else {
			externalGroups[adminID] = &AdministratorGroup{
				AdministratorID: adminID,
				URL:             uaslURL,
				IsInternal:      false,
				ChildDomains:    childList,
				PortRequests:    []model.PortReservationRequest{},
			}
			logger.LogInfo(callerName+": external administrator group identified",
				"administrator_id", adminID,
				"url", uaslURL,
				"children_count", len(childList))
		}
	}

	return externalGroups, nil
}

func (s *InterconnectReservationService) MergeAvailabilityItemsByRequestID(items []*model.AvailabilityItem) []*model.AvailabilityItem {
	if len(items) == 0 {
		return items
	}

	grouped := make(map[string][]*model.AvailabilityItem)
	for _, item := range items {
		grouped[item.RequestID] = append(grouped[item.RequestID], item)
	}

	merged := make([]*model.AvailabilityItem, 0, len(grouped))
	for requestID, group := range grouped {
		if len(group) == 0 {
			continue
		}

		mergedItem := &model.AvailabilityItem{
			RequestID:     requestID,
			OperatorID:    group[0].OperatorID,
			FlightPurpose: group[0].FlightPurpose,
			StartAt:       group[0].StartAt,
			EndAt:         group[0].EndAt,
		}

		for _, item := range group {
			if item.StartAt.Before(mergedItem.StartAt) {
				mergedItem.StartAt = item.StartAt
			}
			if item.EndAt.After(mergedItem.EndAt) {
				mergedItem.EndAt = item.EndAt
			}

			if mergedItem.FlightPurpose == "" && item.FlightPurpose != "" {
				mergedItem.FlightPurpose = item.FlightPurpose
			}
		}

		merged = append(merged, mergedItem)
	}

	logger.LogInfo("MergeAvailabilityItemsByRequestID completed",
		"input_count", len(items),
		"output_count", len(merged),
		"merged_groups", len(items)-len(merged))

	return merged
}

type InternalUaslGroup struct {
	ParentReservation *model.UaslReservation
	Children          []*model.UaslReservation
}

func (s *InterconnectReservationService) DetectInternalUaslGroups(
	allChildren []*model.UaslReservation,
	administratorGroups map[string]*AdministratorGroup,
) []*InternalUaslGroup {

	sort.Slice(allChildren, func(i, j int) bool {
		if allChildren[i].Sequence == nil {
			return false
		}
		if allChildren[j].Sequence == nil {
			return true
		}
		return *allChildren[i].Sequence < *allChildren[j].Sequence
	})

	groups := make([]*InternalUaslGroup, 0)
	var currentGroup *InternalUaslGroup
	var lastSeq *int

	for _, child := range allChildren {

		if child == nil || child.ExAdministratorID == nil || *child.ExAdministratorID == "" {
			continue
		}

		adminID := *child.ExAdministratorID
		adminGroup, exists := administratorGroups[adminID]
		isInternal := exists && adminGroup.IsInternal

		if isInternal {

			if currentGroup != nil && child.Sequence != nil && lastSeq != nil {
				if *child.Sequence != *lastSeq+1 {
					groups = append(groups, currentGroup)
					currentGroup = nil
				}
			}

			if currentGroup == nil {
				currentGroup = &InternalUaslGroup{
					Children: make([]*model.UaslReservation, 0),
				}
			}
			currentGroup.Children = append(currentGroup.Children, child)
			if child.Sequence != nil {
				seq := *child.Sequence
				lastSeq = &seq
			} else {
				lastSeq = nil
			}
		} else {

			if currentGroup != nil {
				groups = append(groups, currentGroup)
				currentGroup = nil
			}
			lastSeq = nil
		}
	}

	if currentGroup != nil {
		groups = append(groups, currentGroup)
	}

	logger.LogInfo("Internal uasl groups detected",
		"groups_count", len(groups))

	return groups
}

func (s *InterconnectReservationService) BuildParentReservationsForGroups(
	groups []*InternalUaslGroup,
	baseParent *model.UaslReservation,
	isInterConnect bool,
) {
	for i, group := range groups {
		if len(group.Children) == 0 {
			continue
		}

		firstChild := group.Children[0]
		lastChild := group.Children[len(group.Children)-1]

		var totalAmount int32 = 0
		for _, child := range group.Children {
			if child.Amount != nil {
				totalAmount += util.SafeInt32FromPtr(child.Amount)
			}
		}
		totalAmountInt := int(totalAmount)

		estimatedAt := time.Now()
		if baseParent != nil && baseParent.EstimatedAt != nil {
			estimatedAt = *baseParent.EstimatedAt
		}
		parentID := value.ModelID(uuid.New().String())
		exUaslID := firstChild.ExUaslID
		exAdministratorID := firstChild.ExAdministratorID
		pricingRuleVersion := firstChild.PricingRuleVersion
		flightPurpose := ""
		if isInterConnect && baseParent != nil {
			if baseParent.ExUaslID != nil {
				exUaslID = baseParent.ExUaslID
			}
			if baseParent.ExAdministratorID != nil {
				exAdministratorID = baseParent.ExAdministratorID
			}
			if baseParent.PricingRuleVersion != nil {
				pricingRuleVersion = baseParent.PricingRuleVersion
			}
			if baseParent.FlightPurpose != "" {
				flightPurpose = baseParent.FlightPurpose
			}
		}

		conformity := model.ConformityAssessmentList{}
		if baseParent != nil && len(baseParent.ConformityAssessment) > 0 {
			sectionIDs := make(map[string]struct{}, len(group.Children))
			for _, child := range group.Children {
				if child.ExUaslSectionID != nil && *child.ExUaslSectionID != "" {
					sectionIDs[*child.ExUaslSectionID] = struct{}{}
				}
			}
			for _, ca := range baseParent.ConformityAssessment {
				if _, ok := sectionIDs[ca.UaslSectionID]; ok {
					conformity = append(conformity, ca)
				}
			}
		}

		parentReservation := &model.UaslReservation{
			ID:                      parentID,
			RequestID:               baseParent.RequestID,
			ParentUaslReservationID: nil,
			ExUaslSectionID:         nil,
			ExUaslID:                exUaslID,
			ExAdministratorID:       exAdministratorID,
			StartAt:                 firstChild.StartAt,
			EndAt:                   lastChild.EndAt,
			AirspaceID:              firstChild.AirspaceID,
			ExReservedBy:            firstChild.ExReservedBy,
			OrganizationID:          firstChild.OrganizationID,
			ProjectID:               firstChild.ProjectID,
			OperationID:             firstChild.OperationID,
			Status:                  value.RESERVATION_STATUS_PENDING,
			PricingRuleVersion:      pricingRuleVersion,
			Amount:                  &totalAmountInt,
			EstimatedAt:             &estimatedAt,
			Sequence:                nil,
			ConformityAssessment:    conformity,
			DestinationReservations: model.DestinationReservationList{},
			FlightPurpose:           flightPurpose,
		}

		group.ParentReservation = parentReservation

		logger.LogInfo("Internal uasl parent reservation created",
			"group_index", i,
			"parent_id", parentReservation.ID.ToString(),
			"children_count", len(group.Children),
			"is_origin", i == 0)
	}
}

func (s *InterconnectReservationService) SetChildReservationParentsForGroups(groups []*InternalUaslGroup) {
	for _, group := range groups {
		if group.ParentReservation == nil {
			continue
		}

		for _, child := range group.Children {
			child.ParentUaslReservationID = &group.ParentReservation.ID
			child.Status = value.RESERVATION_STATUS_INHERITED
		}
	}
}

func (s *InterconnectReservationService) SetExternalChildReservationParents(
	externalReservationResults []*model.ExternalReservationResult,
) {
	for _, extResult := range externalReservationResults {
		if extResult.ReservationData == nil || len(extResult.ReservationData.DestinationReservations) == 0 {
			continue
		}

		for _, destRes := range extResult.ReservationData.DestinationReservations {
			externalParentID := destRes.ReservationID
			if externalParentID == "" || destRes.UaslID == "" {
				continue
			}

			for _, child := range extResult.ChildDomains {
				if child == nil || child.ExUaslID == nil || *child.ExUaslID == "" {
					continue
				}
				if *child.ExUaslID != destRes.UaslID {
					continue
				}

				if child.ParentUaslReservationID != nil && *child.ParentUaslReservationID != "" {
					continue
				}

				parentID, err := value.NewModelIDFromUUIDString(externalParentID)
				if err == nil {
					child.ParentUaslReservationID = &parentID
					child.Status = value.RESERVATION_STATUS_INHERITED

					logger.LogInfo("External child reservation parent set",
						"child_id", child.ID.ToString(),
						"parent_id", externalParentID,
						"ex_uasl_id", func() string {
							if child.ExUaslID != nil {
								return *child.ExUaslID
							}
							return ""
						}())
				}
			}
		}
	}
}

func MergeConformityFromExternal(parent *model.UaslReservation, data *model.UaslReservationData) {
	if parent == nil || data == nil {
		return
	}
	if len(data.DestinationReservations) == 0 {
		return
	}

	existing := make(map[string]struct{})
	for _, ca := range parent.ConformityAssessment {
		key := ca.UaslSectionID + "|" + ca.Type + "|" + ca.StartAt.Format(time.RFC3339) + "|" + ca.EndAt.Format(time.RFC3339)
		existing[key] = struct{}{}
	}

	for _, destRes := range data.DestinationReservations {
		if len(destRes.ConformityAssessmentResults) == 0 {
			continue
		}
		sectionTimes := make(map[string][2]string, len(destRes.UaslSections))
		for _, sec := range destRes.UaslSections {
			sectionTimes[sec.UaslSectionID] = [2]string{sec.StartAt, sec.EndAt}
		}
		for _, ca := range destRes.ConformityAssessmentResults {
			startEnd, ok := sectionTimes[ca.UaslSectionID]
			if !ok {
				continue
			}
			startAt, err1 := time.Parse(time.RFC3339, startEnd[0])
			endAt, err2 := time.Parse(time.RFC3339, startEnd[1])
			if err1 != nil || err2 != nil {
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

			key := ca.UaslSectionID + "|" + ca.Type + "|" + startAt.Format(time.RFC3339) + "|" + endAt.Format(time.RFC3339)
			if _, exists := existing[key]; exists {
				continue
			}
			existing[key] = struct{}{}

			parent.ConformityAssessment = append(parent.ConformityAssessment, model.ConformityAssessmentItem{
				UaslSectionID:     ca.UaslSectionID,
				StartAt:           startAt,
				EndAt:             endAt,
				AircraftInfo:      aircraftInfo,
				EvaluationResults: eval,
				Type:              ca.Type,
				Reasons:           ca.Reasons,
			})
		}
	}
}

func MergeConformityFromInternalGroups(parent *model.UaslReservation, groups []*InternalUaslGroup) {
	if parent == nil || len(groups) == 0 {
		return
	}

	existing := make(map[string]struct{})
	for _, ca := range parent.ConformityAssessment {
		key := ca.UaslSectionID + "|" + ca.Type + "|" + ca.StartAt.Format(time.RFC3339) + "|" + ca.EndAt.Format(time.RFC3339)
		existing[key] = struct{}{}
	}

	for _, g := range groups {
		if g == nil || g.ParentReservation == nil {
			continue
		}
		for _, ca := range g.ParentReservation.ConformityAssessment {
			key := ca.UaslSectionID + "|" + ca.Type + "|" + ca.StartAt.Format(time.RFC3339) + "|" + ca.EndAt.Format(time.RFC3339)
			if _, ok := existing[key]; ok {
				continue
			}
			existing[key] = struct{}{}
			parent.ConformityAssessment = append(parent.ConformityAssessment, ca)
		}
	}
}

func ApplyConformityToExternalGroups(parentDomain *model.UaslReservation, groups []*ExternalUaslGroup) {
	if parentDomain == nil || len(groups) == 0 {
		return
	}

	for _, g := range groups {
		if g == nil || g.ParentReservation == nil || len(g.Children) == 0 {
			continue
		}

		sectionIDSet := make(map[string]struct{}, len(g.Children))
		for _, child := range g.Children {
			if child == nil || child.ExUaslSectionID == nil || *child.ExUaslSectionID == "" {
				continue
			}
			sectionIDSet[*child.ExUaslSectionID] = struct{}{}
		}

		filtered := make(model.ConformityAssessmentList, 0)
		for _, ca := range parentDomain.ConformityAssessment {
			if _, ok := sectionIDSet[ca.UaslSectionID]; ok {
				filtered = append(filtered, ca)
			}
		}

		g.ParentReservation.ConformityAssessment = filtered
	}
}

func (s *InterconnectReservationService) SplitDuplicateSectionTimes(
	childDomains []*model.UaslReservation,
	administratorGroups map[string]*AdministratorGroup,
) {
	type entry struct {
		child   *model.UaslReservation
		origIdx int
	}

	sectionMap := make(map[string][]entry)

	for i, child := range childDomains {
		if child == nil || child.ExUaslSectionID == nil || *child.ExUaslSectionID == "" {
			continue
		}
		if child.ExAdministratorID == nil || *child.ExAdministratorID == "" {
			continue
		}
		adminID := *child.ExAdministratorID
		group, exists := administratorGroups[adminID]
		if !exists || !group.IsInternal {

			continue
		}
		sectionID := *child.ExUaslSectionID
		sectionMap[sectionID] = append(sectionMap[sectionID], entry{child: child, origIdx: i})
	}

	for sectionID, entries := range sectionMap {
		n := len(entries)
		if n <= 1 {

			continue
		}

		sort.Slice(entries, func(i, j int) bool {
			si := entries[i].child.Sequence
			sj := entries[j].child.Sequence
			if si == nil && sj == nil {
				return entries[i].origIdx < entries[j].origIdx
			}
			if si == nil {
				return false
			}
			if sj == nil {
				return true
			}
			return *si < *sj
		})

		baseStart := entries[0].child.StartAt
		baseEnd := entries[0].child.EndAt
		totalDuration := baseEnd.Sub(baseStart)
		if totalDuration <= 0 {
			logger.LogInfo("SplitDuplicateSectionTimes: non-positive duration, skipping",
				"section_id", sectionID,
				"start_at", baseStart,
				"end_at", baseEnd)
			continue
		}

		slotDuration := totalDuration / time.Duration(n)

		logger.LogInfo("SplitDuplicateSectionTimes: splitting section time",
			"section_id", sectionID,
			"count", n,
			"total_duration_min", int(totalDuration.Minutes()),
			"slot_duration_min", int(slotDuration.Minutes()))

		for k, e := range entries {
			slotStart := baseStart.Add(slotDuration * time.Duration(k))
			var slotEnd time.Time
			if k == n-1 {

				slotEnd = baseEnd
			} else {
				slotEnd = baseStart.Add(slotDuration * time.Duration(k+1))
			}
			e.child.StartAt = slotStart
			e.child.EndAt = slotEnd

			seq := -1
			if e.child.Sequence != nil {
				seq = *e.child.Sequence
			}
			logger.LogInfo("SplitDuplicateSectionTimes: assigned slot",
				"section_id", sectionID,
				"slot_index", k,
				"sequence", seq,
				"start_at", slotStart.Format(time.RFC3339),
				"end_at", slotEnd.Format(time.RFC3339))
		}
	}
}

type ExternalUaslGroup struct {
	AdministratorID       string
	ExUaslID              string
	ExternalReservationID string
	ParentReservation     *model.UaslReservation
	Children              []*model.UaslReservation
}

func (s *InterconnectReservationService) DetectExternalUaslGroups(
	allChildren []*model.UaslReservation,
	administratorGroups map[string]*AdministratorGroup,
	externalReservationResults []*model.ExternalReservationResult,
) []*ExternalUaslGroup {

	extReservationIDMap := make(map[string]string)
	for _, extResult := range externalReservationResults {
		if extResult == nil || extResult.ReservationData == nil {
			continue
		}
		for _, destRes := range extResult.ReservationData.DestinationReservations {
			if destRes.ReservationID == "" {
				continue
			}
			adminID := destRes.AdministratorID
			if adminID == "" {
				adminID = extResult.AdministratorID
			}
			key := adminID + "|" + destRes.UaslID
			if _, exists := extReservationIDMap[key]; !exists {
				extReservationIDMap[key] = destRes.ReservationID
			}
		}
	}

	extChildListMap := make(map[string][]*model.UaslReservation)
	for _, extResult := range externalReservationResults {
		for _, child := range extResult.ChildDomains {
			if child == nil || child.ExUaslSectionID == nil || *child.ExUaslSectionID == "" {
				continue
			}
			sID := *child.ExUaslSectionID
			extChildListMap[sID] = append(extChildListMap[sID], child)
		}
	}

	extChildUseCount := make(map[string]int)

	sorted := make([]*model.UaslReservation, 0, len(allChildren))
	sorted = append(sorted, allChildren...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Sequence == nil {
			return false
		}
		if sorted[j].Sequence == nil {
			return true
		}
		return *sorted[i].Sequence < *sorted[j].Sequence
	})

	groups := make([]*ExternalUaslGroup, 0)
	var currentGroup *ExternalUaslGroup
	var lastSeq *int

	for _, child := range sorted {
		if child == nil || child.ExAdministratorID == nil || *child.ExAdministratorID == "" {
			continue
		}

		adminID := *child.ExAdministratorID
		adminGroup, exists := administratorGroups[adminID]
		isExternal := !exists || !adminGroup.IsInternal

		if isExternal {

			if currentGroup != nil && currentGroup.AdministratorID == adminID &&
				child.Sequence != nil && lastSeq != nil && *child.Sequence == *lastSeq+1 {

			} else {

				if currentGroup != nil {
					groups = append(groups, currentGroup)
				}
				exUaslID := ""
				if child.ExUaslID != nil {
					exUaslID = *child.ExUaslID
				}
				extResID := extReservationIDMap[adminID+"|"+exUaslID]
				currentGroup = &ExternalUaslGroup{
					AdministratorID:       adminID,
					ExUaslID:              exUaslID,
					ExternalReservationID: extResID,
					Children:              make([]*model.UaslReservation, 0),
				}
			}

			actualChild := child
			if child.ExUaslSectionID != nil && *child.ExUaslSectionID != "" {
				sID := *child.ExUaslSectionID
				if list, ok := extChildListMap[sID]; ok && len(list) > 0 {
					idx := extChildUseCount[sID]
					if idx < len(list) {
						actualChild = list[idx]
						extChildUseCount[sID] = idx + 1
					}
				}
			}
			currentGroup.Children = append(currentGroup.Children, actualChild)
			if child.Sequence != nil {
				seq := *child.Sequence
				lastSeq = &seq
			} else {
				lastSeq = nil
			}
		} else {

			if currentGroup != nil {
				groups = append(groups, currentGroup)
				currentGroup = nil
			}
			lastSeq = nil
		}
	}

	if currentGroup != nil {
		groups = append(groups, currentGroup)
	}

	logger.LogInfo("External uasl groups detected",
		"groups_count", len(groups))

	return groups
}

func (s *InterconnectReservationService) BuildParentReservationsForExternalGroups(
	groups []*ExternalUaslGroup,
	baseParent *model.UaslReservation,
) {
	for i, group := range groups {
		if len(group.Children) == 0 {
			continue
		}

		firstChild := group.Children[0]
		lastChild := group.Children[len(group.Children)-1]

		var totalAmount int32 = 0
		for _, child := range group.Children {
			if child.Amount != nil {
				totalAmount += util.SafeInt32FromPtr(child.Amount)
			}
		}
		totalAmountInt := int(totalAmount)

		var parentID value.ModelID
		if group.ExternalReservationID != "" {
			parentID = value.ModelID(group.ExternalReservationID)
			logger.LogInfo("BuildParentReservationsForExternalGroups: using external reservation ID as parent ID",
				"external_reservation_id", group.ExternalReservationID,
				"administrator_id", group.AdministratorID)
		} else {
			parentID = value.ModelID(uuid.New().String())
		}

		estimatedAt := time.Now()
		if baseParent != nil && baseParent.EstimatedAt != nil {
			estimatedAt = *baseParent.EstimatedAt
		}

		exUaslID := firstChild.ExUaslID
		exAdministratorID := firstChild.ExAdministratorID
		pricingRuleVersion := firstChild.PricingRuleVersion

		parentReservation := &model.UaslReservation{
			ID:                      parentID,
			RequestID:               baseParent.RequestID,
			ParentUaslReservationID: nil,
			ExUaslSectionID:         nil,
			ExUaslID:                exUaslID,
			ExAdministratorID:       exAdministratorID,
			StartAt:                 firstChild.StartAt,
			EndAt:                   lastChild.EndAt,
			AirspaceID:              firstChild.AirspaceID,
			ExReservedBy:            firstChild.ExReservedBy,
			OrganizationID:          firstChild.OrganizationID,
			ProjectID:               firstChild.ProjectID,
			OperationID:             firstChild.OperationID,
			Status:                  value.RESERVATION_STATUS_PENDING,
			PricingRuleVersion:      pricingRuleVersion,
			Amount:                  &totalAmountInt,
			EstimatedAt:             &estimatedAt,
			Sequence:                nil,
			ConformityAssessment:    model.ConformityAssessmentList{},
			DestinationReservations: model.DestinationReservationList{},
		}

		group.ParentReservation = parentReservation

		logger.LogInfo("External uasl parent reservation created",
			"group_index", i,
			"parent_id", parentReservation.ID.ToString(),
			"administrator_id", group.AdministratorID,
			"children_count", len(group.Children))
	}
}

func (s *InterconnectReservationService) SetChildReservationParentsForExternalGroups(groups []*ExternalUaslGroup) {
	for _, group := range groups {
		if group.ParentReservation == nil {
			continue
		}
		for _, child := range group.Children {
			if child == nil {
				continue
			}
			child.ParentUaslReservationID = &group.ParentReservation.ID
			child.Status = value.RESERVATION_STATUS_INHERITED

			logger.LogInfo("External child reservation parent set (local)",
				"child_id", child.ID.ToString(),
				"parent_id", group.ParentReservation.ID.ToString(),
				"administrator_id", group.AdministratorID)
		}
	}
}
