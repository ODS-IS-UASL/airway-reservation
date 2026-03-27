package services

import (
	"context"
	"fmt"

	"uasl-reservation/internal/app/uasl_reservation/application/converter"
	gatewayIF "uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	repoIF "uasl-reservation/internal/app/uasl_reservation/domain/repositoryIF"
	"uasl-reservation/internal/pkg/logger"
	"uasl-reservation/internal/pkg/retry"
	"uasl-reservation/internal/pkg/util"
	"uasl-reservation/internal/pkg/value"
)

type CompositeReservationOrchestrator struct {
	vehicleGW  gatewayIF.VehicleOpenAPIGatewayIF
	portGW     gatewayIF.PortOpenAPIGatewayIF
	uaslRepo   repoIF.UaslReservationRepositoryIF
	extResRepo repoIF.ExternalResourceReservationRepositoryIF

	ouranosDiscoveryGW       gatewayIF.OuranosDiscoveryGatewayIF
	ouranosProxyGW           gatewayIF.OuranosProxyGatewayIF
	externalUaslDefRepo      repoIF.ExternalUaslDefinitionRepositoryIF
	uaslAdminRepo            repoIF.UaslAdministratorRepositoryIF
	externalUaslResourceRepo repoIF.ExternalUaslResourceRepositoryIF

	SagaOrchestrator *retry.Orchestrator
}

func NewCompositeReservationOrchestrator(
	vehicleGW gatewayIF.VehicleOpenAPIGatewayIF,
	portGW gatewayIF.PortOpenAPIGatewayIF,
	uaslRepo repoIF.UaslReservationRepositoryIF,
	extResRepo repoIF.ExternalResourceReservationRepositoryIF,
	ouranosDiscoveryGW gatewayIF.OuranosDiscoveryGatewayIF,
	ouranosProxyGW gatewayIF.OuranosProxyGatewayIF,
	externalUaslDefRepo repoIF.ExternalUaslDefinitionRepositoryIF,
	uaslAdminRepo repoIF.UaslAdministratorRepositoryIF,
	externalUaslResourceRepo repoIF.ExternalUaslResourceRepositoryIF,
) *CompositeReservationOrchestrator {
	return &CompositeReservationOrchestrator{
		vehicleGW:                vehicleGW,
		portGW:                   portGW,
		uaslRepo:                 uaslRepo,
		extResRepo:               extResRepo,
		ouranosDiscoveryGW:       ouranosDiscoveryGW,
		ouranosProxyGW:           ouranosProxyGW,
		externalUaslDefRepo:      externalUaslDefRepo,
		uaslAdminRepo:            uaslAdminRepo,
		externalUaslResourceRepo: externalUaslResourceRepo,
		SagaOrchestrator:         retry.NewOrchestrator(),
	}
}

func (s *CompositeReservationOrchestrator) ReserveVehicle(
	ctx context.Context,
	requestID value.ModelID,
	req model.VehicleReservationRequest,
) (model.ReservationHandle, []*model.ExternalResourceReservation, error) {
	var zero model.ReservationHandle
	if s.vehicleGW == nil {
		return zero, nil, fmt.Errorf("vehicle reservation prerequisites not met: gateway missing")
	}

	operatorID := ""
	if req.OperatorID != nil {
		operatorID = req.OperatorID.ToString()
	}

	var handle model.ReservationHandle
	op := func(rc context.Context) error {
		var err error

		handle, err = s.vehicleGW.Reserve(rc, "", req)
		return err
	}
	if err := retry.WithBackoff(ctx, op, retry.DefaultConfig()); err != nil {
		return zero, nil, fmt.Errorf("vehicle reservation failed: %w", err)
	}

	s.SagaOrchestrator.RecordSuccess(retry.Step{
		Name: fmt.Sprintf("Vehicle-%s", req.VehicleID),
		Rollback: func(rc context.Context) error {
			if s.vehicleGW == nil {
				return nil
			}

			return s.vehicleGW.Cancel(rc, "", handle, operatorID)
		},
		Metadata: map[string]interface{}{
			"type":       "vehicle",
			"handle_id":  handle.ID,
			"vehicle_id": req.VehicleID,
		},
	})

	resRow, err := model.NewExternalResourceReservation(requestID, handle.ID, model.ExternalResourceTypeVehicle)
	if err != nil {
		return zero, nil, fmt.Errorf("build external resource reservation failed: %w", err)
	}
	resRow.ExResourceID = req.VehicleID
	if !req.StartAt.IsZero() {
		resRow.StartAt = &req.StartAt
	}
	if !req.EndAt.IsZero() {
		resRow.EndAt = &req.EndAt
	}
	mappings := []*model.ExternalResourceReservation{resRow}

	return handle, mappings, nil
}

func (s *CompositeReservationOrchestrator) ReservePort(
	ctx context.Context,
	requestID value.ModelID,
	req model.PortReservationRequest,
) (model.ReservationHandle, []*model.ExternalResourceReservation, error) {
	var zero model.ReservationHandle
	if s.portGW == nil {
		return zero, nil, fmt.Errorf("port reservation prerequisites not met: gateway missing")
	}

	operatorID := ""
	if req.OperatorID != nil {
		operatorID = req.OperatorID.ToString()
	}

	var handle model.ReservationHandle
	op := func(rc context.Context) error {
		var err error

		handle, err = s.portGW.Reserve(rc, "", req)
		return err
	}
	if err := retry.WithBackoff(ctx, op, retry.DefaultConfig()); err != nil {
		return zero, nil, fmt.Errorf("port reservation failed: %w", err)
	}

	s.SagaOrchestrator.RecordSuccess(retry.Step{
		Name: fmt.Sprintf("Port-%s", req.PortID),
		Rollback: func(rc context.Context) error {
			if s.portGW == nil {
				return nil
			}

			return s.portGW.Cancel(rc, "", handle, operatorID)
		},
		Metadata: map[string]interface{}{
			"type":       "port",
			"handle_id":  handle.ID,
			"port_id":    req.PortID,
			"usage_type": req.UsageType,
		},
	})

	resRow, err := model.NewExternalResourceReservation(requestID, handle.ID, model.ExternalResourceTypePort)
	if err != nil {
		return zero, nil, fmt.Errorf("build external resource reservation failed: %w", err)
	}
	resRow.ExResourceID = req.PortID
	usageType := int(req.UsageType)
	resRow.UsageType = &usageType
	if !req.ReservationTimeFrom.IsZero() {
		resRow.StartAt = &req.ReservationTimeFrom
	}
	if !req.ReservationTimeTo.IsZero() {
		resRow.EndAt = &req.ReservationTimeTo
	}
	mappings := []*model.ExternalResourceReservation{resRow}

	return handle, mappings, nil
}

func (s *CompositeReservationOrchestrator) ReserveUasl(
	ctx context.Context,
	composite *model.UaslReservationBatch,
) (*model.UaslReservation, []*model.UaslReservation, error) {
	var parentDomain *model.UaslReservation
	var childDomains []*model.UaslReservation
	var insertedIDs []string
	var rollbackRequestID value.ModelID

	if composite.Parent != nil {
		rollbackRequestID = composite.Parent.RequestID
		inserted, err := s.uaslRepo.InsertOne(composite.Parent)
		if err != nil {
			return nil, nil, fmt.Errorf("parent uasl reservation failed: %w", err)
		}
		parentDomain = inserted
		if parentDomain != nil && parentDomain.RequestID != "" {
			rollbackRequestID = parentDomain.RequestID
		}
		insertedIDs = append(insertedIDs, inserted.ID.ToString())
	}
	if len(composite.Children) > 0 {
		if rollbackRequestID == "" {
			rollbackRequestID = composite.Children[0].RequestID
		}

		if parentDomain != nil {
			for i := range composite.Children {
				child := composite.Children[i]
				if child == nil {
					continue
				}
				if child.ParentUaslReservationID != nil && *child.ParentUaslReservationID != "" {
					continue
				}

				if child.ExUaslSectionID == nil || *child.ExUaslSectionID == "" {
					continue
				}
				child.ParentUaslReservationID = &parentDomain.ID
			}
		}

		inserted, err := s.uaslRepo.InsertBatch(composite.Children)
		if err != nil {
			return nil, nil, fmt.Errorf("child uasl reservations failed: %w", err)
		}
		childDomains = inserted
		if rollbackRequestID == "" && len(childDomains) > 0 && childDomains[0] != nil {
			rollbackRequestID = childDomains[0].RequestID
		}
		for _, child := range inserted {
			insertedIDs = append(insertedIDs, child.ID.ToString())
		}
	}

	s.SagaOrchestrator.RecordSuccess(retry.Step{
		Name: fmt.Sprintf("Uasl-%s", rollbackRequestID.ToString()),
		Rollback: func(rc context.Context) error {
			return s.rollbackUaslReservationsByRequestID(rc, rollbackRequestID)
		},
		Metadata: map[string]interface{}{
			"type":       "uasl",
			"request_id": rollbackRequestID.ToString(),
			"parent_id": func() string {
				if parentDomain != nil {
					return parentDomain.ID.ToString()
				}
				return ""
			}(),
			"child_count": len(childDomains),
		},
	})

	return parentDomain, childDomains, nil
}

func (s *CompositeReservationOrchestrator) rollbackUaslReservationsByRequestID(ctx context.Context, requestID value.ModelID) error {
	if requestID == "" {
		logger.LogError("failed to rollback uasl reservations: request_id is empty")
		return nil
	}
	_, err := s.uaslRepo.DeleteByRequestID(requestID)
	return err
}

func (s *CompositeReservationOrchestrator) CancelVehicle(
	ctx context.Context,
	externalID string,
	operatorID string,
) (*model.VehicleReservationDetail, error) {

	if s.vehicleGW == nil {
		return nil, fmt.Errorf("vehicle gateway not configured")
	}

	detail, err := s.vehicleGW.GetAircraftReservationDetail(ctx, "", externalID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vehicle detail before cancel (id=%s): %w", externalID, err)
	}
	if detail == nil {
		return nil, fmt.Errorf("vehicle detail is nil for externalID=%s", externalID)
	}

	handle, err := model.NewReservationHandle(externalID, model.ResourceTypeVehicle, "")
	if err != nil {
		return nil, fmt.Errorf("failed to build vehicle reservation handle: %w", err)
	}

	cancelOp := func(rc context.Context) error {
		return s.vehicleGW.Cancel(rc, "", handle, operatorID)
	}
	if err := retry.WithBackoff(ctx, cancelOp, retry.DefaultConfig()); err != nil {
		return nil, fmt.Errorf("vehicle cancellation failed after retries: %w", err)
	}

	logger.LogInfo("Vehicle cancelled successfully externalID=%s operatorID=%s", externalID, operatorID)

	capturedDetail := detail
	capturedOperatorID := operatorID
	capturedExternalID := externalID

	s.SagaOrchestrator.RecordSuccess(retry.Step{
		Name: fmt.Sprintf("CancelVehicle-%s", capturedExternalID),
		Rollback: func(rc context.Context) error {
			logger.LogInfo("WARN: Rollback: re-reserving vehicle after cancel failure externalID=%s", capturedExternalID)

			conv := converter.NewVehicleReservationConverter()
			req := conv.ToVehicleReservationRequestFromDetail(capturedDetail, capturedOperatorID)
			_, err = s.vehicleGW.Reserve(rc, "", req)
			return err
		},
		Metadata: map[string]interface{}{
			"type":        "cancel_vehicle",
			"external_id": capturedExternalID,
			"operator_id": capturedOperatorID,
		},
	})

	return detail, nil
}

func (s *CompositeReservationOrchestrator) CancelPort(
	ctx context.Context,
	externalID string,
	operatorID string,
) (*model.PortReservationDetail, error) {

	if s.portGW == nil {
		return nil, fmt.Errorf("port gateway not configured")
	}

	detail, err := s.portGW.GetDronePortReservationDetail(ctx, "", externalID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch port detail before cancel (id=%s): %w", externalID, err)
	}
	if detail == nil {
		return nil, fmt.Errorf("port detail is nil for externalID=%s", externalID)
	}

	handle, err := model.NewReservationHandle(externalID, model.ResourceTypePort, "")
	if err != nil {
		return nil, fmt.Errorf("failed to build port reservation handle: %w", err)
	}

	cancelOp := func(rc context.Context) error {
		return s.portGW.Cancel(rc, "", handle, operatorID)
	}
	if err := retry.WithBackoff(ctx, cancelOp, retry.DefaultConfig()); err != nil {
		return nil, fmt.Errorf("port cancellation failed after retries: %w", err)
	}

	logger.LogInfo("Port cancelled successfully externalID=%s operatorID=%s", externalID, operatorID)

	capturedDetail := detail
	capturedOperatorID := operatorID
	capturedExternalID := externalID

	s.SagaOrchestrator.RecordSuccess(retry.Step{
		Name: fmt.Sprintf("CancelPort-%s", capturedExternalID),
		Rollback: func(rc context.Context) error {
			logger.LogInfo("WARN: Rollback: re-reserving port after cancel failure externalID=%s", capturedExternalID)

			conv := converter.NewPortReservationConverter()
			req := conv.ToPortReservationRequestFromDetail(capturedDetail, capturedOperatorID)
			_, err = s.portGW.Reserve(rc, "", req)
			return err
		},
		Metadata: map[string]interface{}{
			"type":        "cancel_port",
			"external_id": capturedExternalID,
			"operator_id": capturedOperatorID,
		},
	})

	return detail, nil
}

func (s *CompositeReservationOrchestrator) CancelExternalResources(
	ctx context.Context,
	requestID value.ModelID,
	operatorID string,
	reservations []model.ExternalResourceReservation,
) (*model.ExternalResourceData, error) {

	if s.extResRepo == nil {
		return nil, fmt.Errorf("external resource reservation repository is not configured")
	}

	logger.LogInfo("CancelExternalResources started requestID=%s operatorID=%s",
		requestID.ToString(), operatorID)

	externalData := &model.ExternalResourceData{
		Vehicles: make([]*model.VehicleReservationDetail, 0),
		Ports:    make([]*model.PortReservationDetail, 0),
	}

	var err error
	if reservations == nil {
		reservations, err = s.extResRepo.FindByRequestID(requestID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch external resource reservations: %w", err)
		}
		logger.LogInfo("Fetched external resource reservations from DB",
			"request_id", requestID.ToString(),
			"count", len(reservations))
	} else {
		logger.LogInfo("Using pre-fetched external resource reservations",
			"request_id", requestID.ToString(),
			"count", len(reservations))
	}

	if s.uaslAdminRepo != nil {
		internalAdmins, err := s.uaslAdminRepo.FindInternalAdministrators(ctx)
		if err != nil {
			logger.LogError("CancelExternalResources: failed to resolve internal administrators, fallback to unfiltered mappings",
				"error", err,
				"request_id", requestID.ToString())
		} else if len(internalAdmins) > 0 {
			internalAdminIDSet := make(map[string]struct{}, len(internalAdmins))
			for _, a := range internalAdmins {
				if a.ExAdministratorID != "" {
					internalAdminIDSet[a.ExAdministratorID] = struct{}{}
				}
			}
			hasAnyTagged := false
			filtered := make([]model.ExternalResourceReservation, 0, len(reservations))
			for _, res := range reservations {
				if res.ExAdministratorID != "" {
					hasAnyTagged = true
				}
				if _, ok := internalAdminIDSet[res.ExAdministratorID]; ok {
					filtered = append(filtered, res)
				}
			}

			if hasAnyTagged {
				logger.LogInfo("CancelExternalResources: filtered mappings by internal administrators",
					"request_id", requestID.ToString(),
					"internal_admin_ids", len(internalAdminIDSet),
					"before_count", len(reservations),
					"after_count", len(filtered))
				reservations = filtered
			}
		}
	}

	logger.LogInfo("CancelExternalResources: using provided base URLs",
		"request_id", requestID.ToString())

	for _, res := range reservations {

		normalizedType := res.ResourceType
		switch normalizedType {
		case model.ResourceTypeVehicle:
			if res.ExReservationID == "" {
				logger.LogInfo("WARN: skip vehicle cancel because ex_reservation_id is empty or nil requestID=%s", requestID.ToString())
				continue
			}

			if s.vehicleGW == nil {
				logger.LogInfo("WARN: vehicle gateway not configured, skip cancel externalID=%s", res.ExReservationID)
				continue
			}

			detail, err := s.CancelVehicle(ctx, res.ExReservationID, operatorID)
			if err != nil {
				logger.LogError("CRITICAL: vehicle cancellation failed - attempting rollback",
					"error", err,
					"externalID", res.ExReservationID,
					"requestID", requestID.ToString(),
					"severity", "CRITICAL",
					"requires_manual_intervention", true,
				)

				s.SagaOrchestrator.Rollback(ctx)

				return nil, fmt.Errorf(
					"vehicle cancellation failed - rollback attempted but may require manual cleanup: %w",
					err,
				)
			}

			if res.Amount != nil {
				v := *res.Amount

				detail.Amount = util.SafeIntToInt32(v)
			}

			if detail.VehicleID == "" && res.ExResourceID != "" {
				detail.VehicleID = res.ExResourceID
			}

			if res.StartAt != nil {
				detail.StartAt = *res.StartAt
			}
			if res.EndAt != nil {
				detail.EndAt = *res.EndAt
			}

			externalData.Vehicles = append(externalData.Vehicles, detail)

		case model.ResourceTypePort:
			if res.ExReservationID == "" {
				logger.LogInfo("WARN: skip port cancel because ex_reservation_id is empty or nil requestID=%s", requestID.ToString())
				continue
			}

			if s.portGW == nil {
				logger.LogInfo("WARN: port gateway not configured, skip cancel externalID=%s", res.ExReservationID)
				continue
			}

			detail, err := s.CancelPort(ctx, res.ExReservationID, operatorID)
			if err != nil {
				logger.LogError("CRITICAL: port cancellation failed - attempting rollback",
					"error", err,
					"externalID", res.ExReservationID,
					"requestID", requestID.ToString(),
					"severity", "CRITICAL",
					"requires_manual_intervention", true,
				)

				s.SagaOrchestrator.Rollback(ctx)

				return nil, fmt.Errorf(
					"port cancellation failed - rollback attempted but may require manual cleanup: %w",
					err,
				)
			}

			if res.Amount != nil {
				v := *res.Amount

				detail.Amount = util.SafeIntToInt32(v)
			}

			if res.UsageType != nil && detail.UsageType == 0 {
				uv := *res.UsageType

				detail.UsageType = util.SafeIntToInt32(uv)
			}

			if detail.PortID == "" && res.ExResourceID != "" {
				detail.PortID = res.ExResourceID
			}

			if res.StartAt != nil {
				detail.StartAt = *res.StartAt
			}
			if res.EndAt != nil {
				detail.EndAt = *res.EndAt
			}
			if detail.ExAdministratorID == "" && res.ExAdministratorID != "" {
				detail.ExAdministratorID = res.ExAdministratorID
			}

			externalData.Ports = append(externalData.Ports, detail)

		default:
			logger.LogInfo("WARN: skip cancel for unsupported resource type resourceType=%s normalizedType=%s requestID=%s", res.ResourceType.String(), normalizedType.String(), requestID.ToString())
		}
	}

	var vehicleExResourceIDsForName []string
	var portExResourceIDsForName []string

	for _, vehicle := range externalData.Vehicles {
		if vehicle.VehicleName == "" && vehicle.VehicleID != "" {
			vehicleExResourceIDsForName = append(vehicleExResourceIDsForName, vehicle.VehicleID)
		}
	}

	for _, port := range externalData.Ports {
		if port.PortName == "" && port.PortID != "" {
			portExResourceIDsForName = append(portExResourceIDsForName, port.PortID)
		}
	}

	if s.externalUaslResourceRepo != nil {

		if len(vehicleExResourceIDsForName) > 0 {
			vehicleResources, err := s.externalUaslResourceRepo.FindByResourceIDsAndType(vehicleExResourceIDsForName, model.ResourceTypeVehicle)
			if err != nil {
				logger.LogError("Failed to fetch vehicle resources from external_uasl_resources",
					"error", err,
					"vehicle_ids", vehicleExResourceIDsForName)
			} else {
				vehicleResourceMap := make(map[string]*model.ExternalUaslResource)
				for _, res := range vehicleResources {
					vehicleResourceMap[res.ExResourceID] = res
				}

				for _, vehicle := range externalData.Vehicles {
					if vehicle.VehicleName == "" {
						if vehicleResource, ok := vehicleResourceMap[vehicle.VehicleID]; ok {
							vehicle.VehicleName = vehicleResource.Name
						}
					}
				}
			}
		}

		if len(portExResourceIDsForName) > 0 {
			portResources, err := s.externalUaslResourceRepo.FindByResourceIDsAndType(portExResourceIDsForName, model.ResourceTypePort)
			if err != nil {
				logger.LogError("Failed to fetch port resources from external_uasl_resources",
					"error", err,
					"port_ids", portExResourceIDsForName)
			} else {
				portResourceMap := make(map[string]*model.ExternalUaslResource)
				for _, res := range portResources {
					portResourceMap[res.ExResourceID] = res
				}

				for _, port := range externalData.Ports {
					if port.PortName == "" {
						if portResource, ok := portResourceMap[port.PortID]; ok {
							port.PortName = portResource.Name
						}
					}
				}
			}
		}
	}

	logger.LogInfo("CancelExternalResources completed successfully requestID=%s cancelledCount=%d", requestID.ToString(), len(reservations))
	return externalData, nil
}

func (s *CompositeReservationOrchestrator) DeleteExternalResourceMappings(
	ctx context.Context,
	requestID value.ModelID,
	mappings []model.ExternalResourceReservation,
) ([]*model.ExternalResourceReservation, error) {
	if s.extResRepo == nil {
		return nil, fmt.Errorf("external resource reservation repository is not configured")
	}

	logger.LogInfo("DeleteExternalResourceMappings started", "request_id", requestID.ToString())

	var err error
	if mappings == nil {
		mappings, err = s.extResRepo.FindByRequestID(requestID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch external resource mappings before delete: %w", err)
		}
		logger.LogInfo("Fetched external resource mappings from DB for backup",
			"request_id", requestID.ToString(),
			"count", len(mappings))
	} else {
		logger.LogInfo("Using pre-fetched external resource mappings for backup",
			"request_id", requestID.ToString(),
			"count", len(mappings))
	}

	backupMappings := make([]*model.ExternalResourceReservation, len(mappings))
	for i := range mappings {

		mapping := mappings[i]
		backupMappings[i] = &mapping
	}

	logger.LogInfo("Backup created for external resource mappings",
		"request_id", requestID.ToString(),
		"count", len(backupMappings))

	if err := s.extResRepo.DeleteByRequestID(requestID); err != nil {
		return nil, fmt.Errorf("failed to delete external resource mappings: %w", err)
	}

	logger.LogInfo("External resource mappings deleted successfully",
		"request_id", requestID.ToString(),
		"count", len(backupMappings))

	s.SagaOrchestrator.RecordSuccess(retry.Step{
		Name: fmt.Sprintf("DeleteExternalResourceMappings-%s", requestID.ToString()),
		Rollback: func(rc context.Context) error {
			logger.LogInfo("WARN: Rollback: restoring external_resource_reservations",
				"request_id", requestID.ToString(),
				"count", len(backupMappings))

			if len(backupMappings) == 0 {
				return nil
			}

			_, err := s.extResRepo.InsertBatch(backupMappings)
			if err != nil {
				return fmt.Errorf("failed to restore external resource mappings: %w", err)
			}

			logger.LogInfo("External resource mappings restored successfully",
				"request_id", requestID.ToString(),
				"count", len(backupMappings))

			return nil
		},
		Metadata: map[string]interface{}{
			"type":       "delete_external_resource_mappings",
			"request_id": requestID.ToString(),
			"count":      len(backupMappings),
		},
	})

	return backupMappings, nil
}

func (s *CompositeReservationOrchestrator) DeleteUaslReservations(
	ctx context.Context,
	requestID value.ModelID,
	parent *model.UaslReservation,
	children []*model.UaslReservation,
) (int, error) {
	if s.uaslRepo == nil {
		return 0, fmt.Errorf("uasl reservation repository is not configured")
	}

	logger.LogInfo("DeleteUaslReservations started", "request_id", requestID.ToString())

	var err error
	if parent == nil && children == nil {
		parent, children, err = s.uaslRepo.FindByRequestID(requestID)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch uasl reservations before delete: %w", err)
		}
		logger.LogInfo("Fetched uasl reservations from DB for backup",
			"request_id", requestID.ToString(),
			"parent_exists", parent != nil,
			"children_count", len(children))
	} else {
		logger.LogInfo("Using pre-fetched uasl reservations for backup",
			"request_id", requestID.ToString(),
			"parent_exists", parent != nil,
			"children_count", len(children))
	}

	var parentBackup *model.UaslReservation
	var childrenBackup []*model.UaslReservation

	if parent != nil {

		parentCopy := *parent
		parentBackup = &parentCopy
	}

	if len(children) > 0 {
		childrenBackup = make([]*model.UaslReservation, len(children))
		for i := range children {

			childCopy := *children[i]
			childrenBackup[i] = &childCopy
		}
	}

	backupCount := 0
	if parentBackup != nil {
		backupCount++
	}
	backupCount += len(childrenBackup)

	logger.LogInfo("Backup created for uasl reservations",
		"request_id", requestID.ToString(),
		"parent_exists", parentBackup != nil,
		"children_count", len(childrenBackup),
		"total_count", backupCount)

	deletedCount, err := s.uaslRepo.DeleteByRequestID(requestID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete uasl reservations: %w", err)
	}

	logger.LogInfo("Uasl reservations deleted successfully",
		"request_id", requestID.ToString(),
		"deleted_count", deletedCount)

	s.SagaOrchestrator.RecordSuccess(retry.Step{
		Name: fmt.Sprintf("DeleteUaslReservations-%s", requestID.ToString()),
		Rollback: func(rc context.Context) error {
			logger.LogInfo("WARN: Rollback: restoring uasl_reservations",
				"request_id", requestID.ToString(),
				"parent_exists", parentBackup != nil,
				"children_count", len(childrenBackup))

			if parentBackup != nil {
				if _, err := s.uaslRepo.InsertOne(parentBackup); err != nil {
					return fmt.Errorf("failed to restore parent uasl reservation: %w", err)
				}
				logger.LogInfo("Parent uasl reservation restored", "request_id", requestID.ToString())
			}

			if len(childrenBackup) > 0 {
				if _, err := s.uaslRepo.InsertBatch(childrenBackup); err != nil {
					return fmt.Errorf("failed to restore children uasl reservations: %w", err)
				}
				logger.LogInfo("Children uasl reservations restored",
					"request_id", requestID.ToString(),
					"count", len(childrenBackup))
			}

			logger.LogInfo("Uasl reservations restored successfully",
				"request_id", requestID.ToString(),
				"total_count", backupCount)

			return nil
		},
		Metadata: map[string]interface{}{
			"type":           "delete_uasl_reservations",
			"request_id":     requestID.ToString(),
			"deleted_count":  deletedCount,
			"parent_exists":  parentBackup != nil,
			"children_count": len(childrenBackup),
		},
	})

	return deletedCount, nil
}
