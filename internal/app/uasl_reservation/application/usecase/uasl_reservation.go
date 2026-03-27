package usecase

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"uasl-reservation/internal/app/uasl_reservation/application/converter"
	"uasl-reservation/internal/app/uasl_reservation/application/services"
	"uasl-reservation/internal/app/uasl_reservation/application/validator"
	gatewayIF "uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/repositoryIF"
	pkgIF "uasl-reservation/internal/pkg/database/interfaces"
	"uasl-reservation/internal/pkg/logger"
	"uasl-reservation/internal/pkg/myerror"
	"uasl-reservation/internal/pkg/myvalidator/baseValidator"
	"uasl-reservation/internal/pkg/retry"
	"uasl-reservation/internal/pkg/util"
	"uasl-reservation/internal/pkg/value"
)

const (
	maxInt32 = (1 << 31) - 1
	minInt32 = -1 << 31
)

type UaslReservationUsecase struct {
	ctx                        context.Context
	journalDBIF                pkgIF.JournalDBIF
	uaslReservationRepoIF      repositoryIF.UaslReservationRepositoryIF
	externalResourceRepoIF     repositoryIF.ExternalResourceReservationRepositoryIF
	externalUaslResourceRepoIF repositoryIF.ExternalUaslResourceRepositoryIF
	operationRepoIF            repositoryIF.OperationRepositoryIF
	resourceChecker            *services.ResourceAvailabilityService
	vehicleGateway             gatewayIF.VehicleOpenAPIGatewayIF
	portGateway                gatewayIF.PortOpenAPIGatewayIF
	airspaceConflictChecker    *services.AirspaceConflictService
	orchestrator               *services.CompositeReservationOrchestrator
	billingService             *services.BillingCalculationService
	interconnectReservationSvc *services.InterconnectReservationService
	externalIngestService      *services.ExternalDataIngestService
	newSagaTx                  sagaFactory
	ouranosDiscoveryGW         gatewayIF.OuranosDiscoveryGatewayIF
	gswPriceGW                 gatewayIF.ResourcePriceGatewayIF
	ouranosProxyGW             gatewayIF.OuranosProxyGatewayIF
	conformityAssessmentGW     gatewayIF.ConformityAssessmentGatewayIF
	externalUaslDefRepoIF      repositoryIF.ExternalUaslDefinitionRepositoryIF
	uaslAdminRepoIF            repositoryIF.UaslAdministratorRepositoryIF
	paymentGateway             gatewayIF.PaymentGatewayIF
	paymentConverter           *converter.PaymentConverter
}

type sagaFactory func(
	vehicleGW gatewayIF.VehicleOpenAPIGatewayIF,
	portGW gatewayIF.PortOpenAPIGatewayIF,
	uaslRepo repositoryIF.UaslReservationRepositoryIF,
	extResRepo repositoryIF.ExternalResourceReservationRepositoryIF,
	ouranosDiscoveryGW gatewayIF.OuranosDiscoveryGatewayIF,
	ouranosProxyGW gatewayIF.OuranosProxyGatewayIF,
	externalUaslDefRepoIF repositoryIF.ExternalUaslDefinitionRepositoryIF,
	uaslAdminRepoIF repositoryIF.UaslAdministratorRepositoryIF,
	externalUaslResourceRepoIF repositoryIF.ExternalUaslResourceRepositoryIF,
) *services.CompositeReservationOrchestrator

func defaultSagaFactory(
	vehicleGW gatewayIF.VehicleOpenAPIGatewayIF,
	portGW gatewayIF.PortOpenAPIGatewayIF,
	uaslRepo repositoryIF.UaslReservationRepositoryIF,
	extResRepo repositoryIF.ExternalResourceReservationRepositoryIF,
	ouranosDiscoveryGW gatewayIF.OuranosDiscoveryGatewayIF,
	ouranosProxyGW gatewayIF.OuranosProxyGatewayIF,
	externalUaslDefRepoIF repositoryIF.ExternalUaslDefinitionRepositoryIF,
	uaslAdminRepoIF repositoryIF.UaslAdministratorRepositoryIF,
	externalUaslResourceRepoIF repositoryIF.ExternalUaslResourceRepositoryIF,
) *services.CompositeReservationOrchestrator {
	return services.NewCompositeReservationOrchestrator(
		vehicleGW,
		portGW,
		uaslRepo,
		extResRepo,
		ouranosDiscoveryGW,
		ouranosProxyGW,
		externalUaslDefRepoIF,
		uaslAdminRepoIF,
		externalUaslResourceRepoIF,
	)
}

func NewUaslReservationUsecase(
	ctx context.Context,
	journalDBIF pkgIF.JournalDBIF,
	repoIF repositoryIF.UaslReservationRepositoryIF,
	externalResourceRepoIF repositoryIF.ExternalResourceReservationRepositoryIF,
	operationRepoIF repositoryIF.OperationRepositoryIF,
	vehicleGW gatewayIF.VehicleOpenAPIGatewayIF,
	portGW gatewayIF.PortOpenAPIGatewayIF,
	flightPlanGW gatewayIF.FlightPlanInfoOpenAPIGatewayIF,
	externalResourceRepo repositoryIF.ExternalUaslResourceRepositoryIF,
	ouranosDiscoveryGW gatewayIF.OuranosDiscoveryGatewayIF,
	gswPriceGW gatewayIF.ResourcePriceGatewayIF,
	ouranosProxyGW gatewayIF.OuranosProxyGatewayIF,
	conformityAssessmentGW gatewayIF.ConformityAssessmentGatewayIF,
	externalUaslDefRepoIF repositoryIF.ExternalUaslDefinitionRepositoryIF,
	uaslAdminRepoIF repositoryIF.UaslAdministratorRepositoryIF,
	uaslDesignGW gatewayIF.UaslDesignGatewayIF,
	paymentGW gatewayIF.PaymentGatewayIF,
) *UaslReservationUsecase {
	resourceChecker := services.NewResourceAvailabilityService(vehicleGW, portGW, externalResourceRepo)
	airspaceConflictChecker := services.NewAirspaceConflictService(flightPlanGW, externalUaslDefRepoIF, repoIF, conformityAssessmentGW)
	orchestrator := services.NewCompositeReservationOrchestrator(
		vehicleGW,
		portGW,
		repoIF,
		externalResourceRepoIF,
		ouranosDiscoveryGW,
		ouranosProxyGW,
		externalUaslDefRepoIF,
		uaslAdminRepoIF,
		externalResourceRepo,
	)

	billingService := services.NewBillingCalculationService(gswPriceGW, orchestrator, externalUaslDefRepoIF)

	interconnectReservationSvc := services.NewInterconnectReservationService(
		ouranosDiscoveryGW,
		ouranosProxyGW,
		uaslDesignGW,
		externalUaslDefRepoIF,
		externalResourceRepo,
		uaslAdminRepoIF,
		orchestrator,
	)

	externalIngestService := services.NewExternalDataIngestService(
		ouranosDiscoveryGW,
		uaslDesignGW,
		vehicleGW,
		portGW,
		uaslAdminRepoIF,
		externalUaslDefRepoIF,
		externalResourceRepo,
	)

	return &UaslReservationUsecase{
		ctx:                        ctx,
		journalDBIF:                journalDBIF,
		uaslReservationRepoIF:      repoIF,
		externalResourceRepoIF:     externalResourceRepoIF,
		externalUaslResourceRepoIF: externalResourceRepo,
		operationRepoIF:            operationRepoIF,
		resourceChecker:            resourceChecker,
		vehicleGateway:             vehicleGW,
		portGateway:                portGW,
		airspaceConflictChecker:    airspaceConflictChecker,
		orchestrator:               orchestrator,
		billingService:             billingService,
		interconnectReservationSvc: interconnectReservationSvc,
		externalIngestService:      externalIngestService,
		newSagaTx:                  defaultSagaFactory,
		ouranosDiscoveryGW:         ouranosDiscoveryGW,
		gswPriceGW:                 gswPriceGW,
		ouranosProxyGW:             ouranosProxyGW,
		conformityAssessmentGW:     conformityAssessmentGW,
		externalUaslDefRepoIF:      externalUaslDefRepoIF,
		uaslAdminRepoIF:            uaslAdminRepoIF,
		paymentGateway:             paymentGW,
		paymentConverter:           converter.NewPaymentConverter(),
	}
}

func (uc *UaslReservationUsecase) Register(ctx context.Context, req *converter.RegisterUaslReservationInput) (*model.RegisterUaslReservationResponse, error) {
	logger.LogInfo("Register uasl reservation request received", "request", req)

	uaslReservation, err := uc.buildUaslReservationDomain(req)
	if err != nil {
		return nil, err
	}
	logger.LogInfo("Uasl reservation domain built", "uaslReservation", uaslReservation)

	uaslReservationDomain, err := uc.uaslReservationRepoIF.InsertOne(uaslReservation)
	if err != nil {
		return nil, err
	}

	if uaslReservationDomain == nil {
		logger.LogError("Uasl reservation insertion returned nil")
		return nil, fmt.Errorf("uasl reservation insertion failed: returned nil")
	}
	logger.LogInfo("Uasl reservation inserted",
		"reservation_id", uaslReservationDomain.ID.ToString())

	domainConv := converter.NewUaslReservationDomain()

	return &model.RegisterUaslReservationResponse{
		Data: domainConv.ToUaslReservationResponse(uaslReservationDomain),
	}, nil
}

func (uc *UaslReservationUsecase) Cancel(ctx context.Context, req *converter.CancelUaslReservationInput) (*model.ReserveCompositeUaslResponse, error) {
	logger.LogInfo("Cancel uasl reservation request received",
		"id", req.ID,
		"status", req.Status,
		"is_inter_connect", req.IsInterConnect)

	valid := validator.NewUaslReservation()
	if err := valid.CancelRequest(ctx, req); err != nil {
		return nil, err
	}

	id, err := value.NewModelIDFromUUIDString(req.ID)
	if err != nil {
		return nil, err
	}

	reservationStatus := value.ReservationStatus(req.Status)

	if reservationStatus != value.RESERVATION_STATUS_CANCELED &&
		reservationStatus != value.RESERVATION_STATUS_RESCINDED {
		return nil, fmt.Errorf("invalid status: only CANCELED or RESCINDED updates are supported")
	}

	isCancellation := reservationStatus == value.RESERVATION_STATUS_CANCELED

	logger.LogInfo("Starting cancel/rescind process",
		"reservation_id", id.ToString(),
		"target_status", reservationStatus.ToString(),
		"is_cancellation", isCancellation,
		"is_inter_connect", req.IsInterConnect)

	current, children, err := uc.uaslReservationRepoIF.FindByRequestID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load reservation and children: %w", err)
	}
	if current == nil {
		return nil, fmt.Errorf("uasl reservation not found")
	}

	if current.Status != value.RESERVATION_STATUS_PENDING &&
		current.Status != value.RESERVATION_STATUS_RESERVED {
		return nil, fmt.Errorf("invalid status: cannot cancel/rescind reservation with status %s", current.Status.ToString())
	}

	var externalGroups map[string]*services.AdministratorGroup

	if !req.IsInterConnect && uc.interconnectReservationSvc != nil {

		var err error
		externalGroups, err = uc.interconnectReservationSvc.BuildAdministratorGroupsFromReservedChildren(
			ctx,
			children,
			"Cancel",
		)
		if err != nil {
			logger.LogError("Cancel: failed to build administrator groups", "error", err)
			return nil, fmt.Errorf("failed to build administrator groups: %w", err)
		}
	}

	operatorID := ""
	if current.ExReservedBy != nil {
		operatorID = current.ExReservedBy.ToString()
	}

	sagaOrchestrator := uc.newSagaTx(
		uc.vehicleGateway,
		uc.portGateway,
		uc.uaslReservationRepoIF,
		uc.externalResourceRepoIF,
		uc.ouranosDiscoveryGW,
		uc.ouranosProxyGW,
		uc.externalUaslDefRepoIF,
		uc.uaslAdminRepoIF,
		uc.externalUaslResourceRepoIF,
	)

	type CancelDataWithAdmin struct {
		Data            *model.UaslReservationData
		AdministratorID string
	}
	var cancelDataList []CancelDataWithAdmin
	if len(externalGroups) > 0 {
		logger.LogInfo("Cancelling external uasl reservations",
			"external_groups_count", len(externalGroups))

		for _, adminGroup := range externalGroups {

			externalRequestID := ""
			if len(adminGroup.ChildDomains) > 0 && adminGroup.ChildDomains[0] != nil {
				externalRequestID = adminGroup.ChildDomains[0].RequestID.ToString()
			}

			if externalRequestID == "" {
				logger.LogError("External request ID not found for administrator",
					"administrator_id", adminGroup.AdministratorID)
				continue
			}

			cancelData, err := uc.interconnectReservationSvc.CancelExternalUasl(
				ctx,
				*adminGroup,
				externalRequestID,
				reservationStatus.ToString(),
				sagaOrchestrator.SagaOrchestrator,
			)
			if err != nil {
				logger.LogError("CRITICAL: external uasl cancellation failed - attempting rollback",
					"administrator_id", adminGroup.AdministratorID,
					"external_request_id", externalRequestID,
					"error", err)

				sagaOrchestrator.SagaOrchestrator.Rollback(ctx)
				return nil, fmt.Errorf("failed to cancel external uasl reservation for %s: %w",
					adminGroup.AdministratorID, err)
			}
			if cancelData != nil {
				cancelDataList = append(cancelDataList, CancelDataWithAdmin{
					Data:            cancelData,
					AdministratorID: adminGroup.AdministratorID,
				})
			}
		}
	}

	allParents, err := uc.uaslReservationRepoIF.FindParentsByRequestID(current.RequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to load all parent reservations: %w", err)
	}

	logger.LogInfo("Cancelling all parent reservations",
		"request_id", current.RequestID.ToString(),
		"parents_count", len(allParents),
		"target_status", reservationStatus.ToString())

	now := time.Now()
	reservationsToUpdate := make([]*model.UaslReservation, 0, len(allParents))

	type parentBackup struct {
		parent         *model.UaslReservation
		originalStatus value.ReservationStatus
		originalTime   time.Time
	}
	backups := make([]parentBackup, 0, len(allParents))

	for _, parent := range allParents {
		backups = append(backups, parentBackup{
			parent:         parent,
			originalStatus: parent.Status,
			originalTime:   parent.UpdatedAt,
		})

		parent.Status = reservationStatus
		parent.UpdatedAt = now
		reservationsToUpdate = append(reservationsToUpdate, parent)
	}

	updatedReservations, err := uc.uaslReservationRepoIF.UpdateBatch(reservationsToUpdate)
	if err != nil {
		logger.LogError("CRITICAL: reservation status update failed - attempting rollback",
			"error", err,
			"parents_count", len(allParents),
			"target_status", reservationStatus.ToString())
		sagaOrchestrator.SagaOrchestrator.Rollback(ctx)
		return nil, fmt.Errorf("failed to update reservation status: %w", err)
	}

	originalFlightPurpose := current.FlightPurpose

	for _, updated := range updatedReservations {
		if updated != nil && updated.ID == current.ID {
			current = updated
			current.FlightPurpose = originalFlightPurpose
			break
		}
	}

	sagaOrchestrator.SagaOrchestrator.RecordSuccess(retry.Step{
		Name: "ReservationStatusUpdate-AllParents",
		Rollback: func(rollbackCtx context.Context) error {

			rollbackList := make([]*model.UaslReservation, 0, len(backups))
			for _, backup := range backups {
				backup.parent.Status = backup.originalStatus
				backup.parent.UpdatedAt = backup.originalTime
				rollbackList = append(rollbackList, backup.parent)
			}
			_, err := uc.uaslReservationRepoIF.UpdateBatch(rollbackList)
			return err
		},
		Metadata: map[string]interface{}{
			"parents_count": len(allParents),
			"target_status": reservationStatus.ToString(),
		},
	})

	logger.LogInfo("Reservation status updated",
		"parents_count", len(updatedReservations),
		"status", reservationStatus.ToString())

	externalData, err := sagaOrchestrator.CancelExternalResources(ctx, current.RequestID, operatorID, nil)
	if err != nil {
		logger.LogError("CRITICAL: GSW resource cancellation failed - attempting rollback",
			"error", err,
			"request_id", current.RequestID.ToString())
		sagaOrchestrator.SagaOrchestrator.Rollback(ctx)
		return nil, fmt.Errorf("failed to cancel GSW resources: %w", err)
	}

	var originalMappings []model.ExternalResourceReservation
	if uc.externalResourceRepoIF != nil {

		mappings, err := uc.externalResourceRepoIF.FindByRequestID(current.RequestID)
		if err != nil {
			logger.LogError("Failed to fetch external resource mappings before clearing", "error", err)
		} else {
			originalMappings = mappings
		}

		if len(mappings) > 0 {

			clearedMappings := make([]*model.ExternalResourceReservation, 0, len(mappings))
			for _, mapping := range mappings {
				clearedMapping := mapping
				clearedMapping.ExReservationID = ""
				clearedMappings = append(clearedMappings, &clearedMapping)
			}

			if _, err := uc.externalResourceRepoIF.UpdateBatch(clearedMappings); err != nil {
				logger.LogError("CRITICAL: external resource mapping ex_reservation_id clear failed - attempting rollback",
					"error", err,
					"request_id", current.RequestID.ToString())
				sagaOrchestrator.SagaOrchestrator.Rollback(ctx)
				return nil, fmt.Errorf("failed to clear external resource mappings: %w", err)
			}
		}

		if len(originalMappings) > 0 {
			sagaOrchestrator.SagaOrchestrator.RecordSuccess(retry.Step{
				Name: fmt.Sprintf("ExternalResourceMappingClear-%s", current.RequestID.ToString()),
				Rollback: func(rollbackCtx context.Context) error {

					restoredMappings := make([]*model.ExternalResourceReservation, 0, len(originalMappings))
					for _, mapping := range originalMappings {
						restored := mapping
						restoredMappings = append(restoredMappings, &restored)
					}
					if _, err := uc.externalResourceRepoIF.UpdateBatch(restoredMappings); err != nil {
						return fmt.Errorf("failed to restore ex_reservation_id: %w", err)
					}
					return nil
				},
				Metadata: map[string]interface{}{
					"request_id":     current.RequestID.ToString(),
					"mappings_count": len(originalMappings),
				},
			})
		}
	}

	domainConv := converter.NewUaslReservationDomain()
	vehicleConv := converter.NewVehicleReservationConverter()
	portConv := converter.NewPortReservationConverter()

	vehicleDetails := make([]*model.VehicleReservationDetail, 0)
	portDetails := make([]*model.PortReservationDetail, 0)
	if externalData != nil {
		vehicleDetails = append(vehicleDetails, externalData.Vehicles...)
		portDetails = append(portDetails, externalData.Ports...)
	}

	for _, cancelDataWithAdmin := range cancelDataList {
		if cancelDataWithAdmin.Data == nil {
			continue
		}
		cancelData := cancelDataWithAdmin.Data
		administratorID := cancelDataWithAdmin.AdministratorID

		services.MergeConformityFromExternal(current, cancelData)

		var allVehicles []model.VehicleReservationElement
		var allPorts []model.PortReservationElement
		for _, destRes := range cancelData.DestinationReservations {
			allVehicles = append(allVehicles, destRes.Vehicles...)
			for _, p := range destRes.Ports {
				p.ExAdministratorID = administratorID
				allPorts = append(allPorts, p)
			}
		}
		if len(allVehicles) > 0 {
			externalVehicleDetails := vehicleConv.ToVehicleReservationDetailsFromExternalElements(allVehicles)
			vehicleDetails = append(vehicleDetails, externalVehicleDetails...)
		}
		if len(allPorts) > 0 {
			externalPortDetails := portConv.ToPortReservationDetailsFromExternalElements(allPorts)
			portDetails = append(portDetails, externalPortDetails...)
		}

		logger.LogInfo("External cancellation data processed",
			"administrator_id", administratorID,
			"vehicles_count", len(allVehicles),
			"ports_count", len(allPorts))
	}

	vehicles := vehicleConv.ToVehicleElements(vehicleDetails)
	ports := portConv.ToPortElements(portDetails)

	logger.LogInfo("Cancel/rescind operation completed",
		"reservation_id", current.ID.ToString(),
		"status", current.Status.ToString(),
		"is_cancellation", isCancellation)

	return domainConv.ToCompositeUaslReservationResponse(
		current,
		children,
		nil,
		vehicles,
		ports,
	), nil
}

func (uc *UaslReservationUsecase) Delete(ctx context.Context, req *converter.DeleteUaslReservationInput) (*model.DeleteUaslReservationResponse, error) {
	valid := validator.NewUaslReservation()
	if err := valid.DeleteRequest(ctx, req); err != nil {
		return nil, err
	}
	requestID, err := value.NewModelIDFromUUIDString(req.ID)
	if err != nil {
		return nil, err
	}

	parent, children, err := uc.uaslReservationRepoIF.FindByRequestID(requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to load reservation and children: %w", err)
	}
	if parent == nil {
		return nil, myerror.Wrap(myerror.BadParams, fmt.Errorf("uasl reservation not found"), "uasl reservation not found")
	}

	var externalResourceReservations []model.ExternalResourceReservation
	if uc.externalResourceRepoIF != nil {
		externalResourceReservations, err = uc.externalResourceRepoIF.FindByRequestID(requestID)
		if err != nil {
			return nil, fmt.Errorf("failed to load external resource reservations: %w", err)
		}
		logger.LogInfo("Loaded external resource reservations for deletion",
			"request_id", requestID.ToString(),
			"count", len(externalResourceReservations))
	}

	operatorID := ""
	if parent.ExReservedBy != nil {
		operatorID = parent.ExReservedBy.ToString()
	}
	sagaOrchestrator := uc.newSagaTx(
		uc.vehicleGateway,
		uc.portGateway,
		uc.uaslReservationRepoIF,
		uc.externalResourceRepoIF,
		uc.ouranosDiscoveryGW,
		uc.ouranosProxyGW,
		uc.externalUaslDefRepoIF,
		uc.uaslAdminRepoIF,
		uc.externalUaslResourceRepoIF,
	)

	if uc.interconnectReservationSvc != nil {
		allUaslIds := make([]string, 0, len(children))
		for _, child := range children {
			if child.ExUaslID != nil && *child.ExUaslID != "" {
				allUaslIds = append(allUaslIds, *child.ExUaslID)
			}
		}
		internalUaslIDs, externalUaslIDs, classifyErr := uc.interconnectReservationSvc.ClassifyInternalExternalUaslIDs(ctx, allUaslIds)
		if classifyErr != nil {
			logger.LogError("ConfirmCompositeUasl: failed to classify uasl ids", "error", classifyErr)
		}

		if len(externalUaslIDs) > 0 {
			_ = uc.interconnectReservationSvc.ResolveAdministratorsForAvailability(ctx, externalUaslIDs, internalUaslIDs)
			logger.LogInfo("Delete: resolved administrator URLs for GSW cancellation",
				"external_uasl_ids_count", len(externalUaslIDs),
				"internal_uasl_ids_count", len(internalUaslIDs))
		}
	}

	if _, err := sagaOrchestrator.CancelExternalResources(ctx, requestID, operatorID, externalResourceReservations); err != nil {

		sagaOrchestrator.SagaOrchestrator.Rollback(ctx)
		return nil, fmt.Errorf("failed to delete GSW resources: %w", err)
	}

	if uc.externalResourceRepoIF != nil {
		if _, err := sagaOrchestrator.DeleteExternalResourceMappings(ctx, requestID, externalResourceReservations); err != nil {

			logger.LogError("CRITICAL: failed to delete external resource mappings - attempting rollback",
				"error", err,
				"request_id", requestID.ToString(),
				"severity", "CRITICAL",
				"requires_manual_intervention", true,
			)
			sagaOrchestrator.SagaOrchestrator.Rollback(ctx)
			return nil, fmt.Errorf("failed to delete external resource mappings - rollback attempted: %w", err)
		}
	}

	deletedCount, err := sagaOrchestrator.DeleteUaslReservations(ctx, requestID, parent, children)
	if err != nil {

		logger.LogError("CRITICAL: failed to delete uasl reservations - attempting rollback",
			"error", err,
			"request_id", requestID.ToString(),
			"severity", "CRITICAL",
			"requires_manual_intervention", true,
		)
		sagaOrchestrator.SagaOrchestrator.Rollback(ctx)
		return nil, fmt.Errorf("failed to delete uasl reservations - rollback attempted: %w", err)
	}

	if deletedCount == 0 {
		logger.LogInfo("No reservations deleted by request_id (idempotent)",
			"request_id", requestID.ToString())
	} else {
		logger.LogInfo("Delete operation completed successfully",
			"request_id", requestID.ToString(),
			"deleted_count", deletedCount)
	}

	return &model.DeleteUaslReservationResponse{
		Data: &model.UaslReservationMessage{
			ID: requestID.ToString(),
		},
	}, nil
}

func (uc *UaslReservationUsecase) FindByRequestID(ctx context.Context, req *converter.FindUaslReservationInput) (*model.FindUaslReservationResponse, error) {
	requestID, err := value.NewModelIDFromUUIDString(req.ID)
	if err != nil {
		return nil, err
	}

	parent, children, err := uc.uaslReservationRepoIF.FindByRequestID(requestID)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return &model.FindUaslReservationResponse{
			Parent:                  nil,
			Children:                []*model.UaslReservationMessage{},
			ConflictedFlightPlanIDs: []string{},
			Vehicles:                []*model.VehicleElement{},
			Ports:                   []*model.PortElement{},
		}, nil
	}

	var conflictResult *model.AirspaceConflictResult
	if uc.airspaceConflictChecker != nil {
		uaslRequests := make([]model.UaslCheckRequest, 0)
		if parent.ExUaslSectionID != nil {
			exUaslSectionID, err := value.NewModelIDFromUUIDString(*parent.ExUaslSectionID)
			if err == nil {
				uaslRequests = append(uaslRequests, model.UaslCheckRequest{
					ExUaslSectionID: exUaslSectionID.ToString(),
					TimeRange: model.TimeRange{
						Start: parent.StartAt,
						End:   parent.EndAt,
					},
				})
			}
		}

		for _, child := range children {
			if child.ExUaslSectionID == nil {
				continue
			}
			exUaslSectionID, err := value.NewModelIDFromUUIDString(*child.ExUaslSectionID)
			if err == nil {
				uaslRequests = append(uaslRequests, model.UaslCheckRequest{
					ExUaslSectionID: exUaslSectionID.ToString(),
					TimeRange: model.TimeRange{
						Start: child.StartAt,
						End:   child.EndAt,
					},
				})
			}
		}

		if len(uaslRequests) > 0 {
			conflictResult, err = uc.airspaceConflictChecker.CheckAirspaceConflict(ctx, uaslRequests)
			if err != nil {
				logger.LogError("airspace conflict check failed on GetUaslReservationDetail", "error", err)
				conflictResult = nil
			}
		}
	}

	var vehicleDetails []*model.VehicleReservationDetail
	var portDetails []*model.PortReservationDetail

	vehicleConverter := converter.NewVehicleReservationConverter()
	portConverter := converter.NewPortReservationConverter()

	if uc.externalResourceRepoIF != nil {
		externalResources, err := uc.externalResourceRepoIF.FindByRequestID(requestID)
		if err != nil {
			logger.LogError("failed to fetch external resource reservations", "error", err, "request_id", requestID.ToString())
		} else {

			for _, res := range externalResources {
				switch res.ResourceType {
				case model.ExternalResourceTypeVehicle:
					detail, err := vehicleConverter.ToVehicleReservationDetailFromExternal(&res)
					if err != nil {
						logger.LogError("failed to convert vehicle reservation", "error", err, "reservation_id", res.ExReservationID)
						return nil, myerror.Wrap(myerror.Internal, err, "failed to convert vehicle reservation")
					}
					vehicleDetails = append(vehicleDetails, detail)

				case model.ExternalResourceTypePort:
					detail, err := portConverter.ToPortReservationDetailFromExternal(&res)
					if err != nil {
						logger.LogError("failed to convert port reservation", "error", err, "reservation_id", res.ExReservationID)
						return nil, myerror.Wrap(myerror.Internal, err, "failed to convert port reservation")
					}
					portDetails = append(portDetails, detail)
				}
			}
		}
	}

	parentOrigin := services.ReservationParentOrigin{}
	if parent.ExUaslID != nil {
		parentOrigin.UaslID = *parent.ExUaslID
	}
	if parent.ExAdministratorID != nil {
		parentOrigin.AdministratorID = *parent.ExAdministratorID
	}
	findAdminResolution := &services.AvailabilityAdminResolution{
		ExternalServices:        make(map[string]model.ExternalServiceEndpoints),
		UaslIDToAdministratorID: make(map[string]string),
	}
	if uc.interconnectReservationSvc != nil {
		findAdminResolution = uc.interconnectReservationSvc.ResolveAdminResolutionByParentOrigins(ctx, []services.ReservationParentOrigin{parentOrigin})
	}

	domainConv := converter.NewUaslReservationDomain()
	vehicleConv := converter.NewVehicleReservationConverter()
	portConv := converter.NewPortReservationConverter()

	responseParent := domainConv.ToUaslReservationResponse(parent)
	responseChildren := domainConv.ToChildReservationMessagesWithConformity(children, parent)
	responseVehicles := vehicleConv.ToVehicleElements(vehicleDetails)
	responsePorts := portConv.ToPortElements(portDetails)

	hasVehicleID := false
	for _, v := range responseVehicles {
		if v != nil && v.VehicleID != "" {
			hasVehicleID = true
			break
		}
	}
	if !hasVehicleID {
		if aircraft := converter.ToAircraftInfoModel(parent); aircraft != nil {
			responseVehicles = append(responseVehicles, &model.VehicleElement{
				AircraftInfo: aircraft,
			})
		}
	}

	parentIsExternal := false
	if parent.ExUaslID != nil && *parent.ExUaslID != "" {
		_, parentIsExternal = findAdminResolution.ExternalServices[*parent.ExUaslID]
	}
	hasMissingPortName := false
	for _, p := range portDetails {
		if p != nil && p.PortName == "" {
			hasMissingPortName = true
			break
		}
	}

	if uc.interconnectReservationSvc != nil && findAdminResolution != nil && (parentIsExternal || hasMissingPortName) {
		externalItem := uc.interconnectReservationSvc.FetchFindByRequestIDExternalItem(
			ctx,
			requestID.ToString(),
			parent,
			children,
			*findAdminResolution,
		)
		if externalItem != nil {
			externalProtoItems := domainConv.ToExternalReservationListItems([]model.ExternalReservationListItem{*externalItem})
			if len(externalProtoItems) > 0 && externalProtoItems[0] != nil {
				externalProto := externalProtoItems[0]

				localVehicleNameByID := make(map[string]string)
				for _, v := range vehicleDetails {
					if v == nil || v.VehicleID == "" || v.VehicleName == "" {
						continue
					}
					localVehicleNameByID[v.VehicleID] = v.VehicleName
				}
				for _, v := range externalProto.Vehicles {
					if v == nil || v.VehicleID == "" || v.Name != "" {
						continue
					}
					if name, ok := localVehicleNameByID[v.VehicleID]; ok {
						v.Name = name
					}
				}

				localPortNameByID := make(map[string]string)
				for _, p := range portDetails {
					if p == nil || p.PortID == "" || p.PortName == "" {
						continue
					}
					localPortNameByID[p.PortID] = p.PortName
				}
				for _, p := range externalProto.Ports {
					if p == nil || p.PortID == "" || p.Name != "" {
						continue
					}
					if name, ok := localPortNameByID[p.PortID]; ok {
						p.Name = name
					}
				}

				if parentIsExternal {
					responseParent = externalProto.ParentUaslReservation
					responseChildren = externalProto.ChildUaslReservations
					responseVehicles = externalProto.Vehicles
					responsePorts = externalProto.Ports
				} else {
					vehicleNameByID := make(map[string]string)
					for _, v := range externalProto.Vehicles {
						if v == nil || v.VehicleID == "" || v.Name == "" {
							continue
						}
						vehicleNameByID[v.VehicleID] = v.Name
					}
					for _, v := range responseVehicles {
						if v == nil || v.VehicleID == "" || v.Name != "" {
							continue
						}
						if name, ok := vehicleNameByID[v.VehicleID]; ok {
							v.Name = name
						}
					}

					portNameByID := make(map[string]string)
					for _, p := range externalProto.Ports {
						if p == nil || p.PortID == "" || p.Name == "" {
							continue
						}
						portNameByID[p.PortID] = p.Name
					}
					for _, p := range responsePorts {
						if p == nil || p.PortID == "" || p.Name != "" {
							continue
						}
						if name, ok := portNameByID[p.PortID]; ok {
							p.Name = name
						}
					}
				}
			}
		}
	}

	var conflictedFlightPlanIDs []string
	if conflictResult != nil && conflictResult.HasConflict && len(conflictResult.ConflictedFlightPlanIDs) > 0 {
		conflictedFlightPlanIDs = conflictResult.ConflictedFlightPlanIDs
	} else {
		conflictedFlightPlanIDs = []string{}
	}

	return &model.FindUaslReservationResponse{
		Parent:                  responseParent,
		Children:                responseChildren,
		ConflictedFlightPlanIDs: conflictedFlightPlanIDs,
		Vehicles:                responseVehicles,
		Ports:                   responsePorts,
	}, nil
}

func (uc *UaslReservationUsecase) buildUaslReservationDomain(req interface{}) (*model.UaslReservation, error) {
	r, ok := req.(*converter.RegisterUaslReservationInput)
	if !ok {
		return nil, fmt.Errorf("invalid request type for createUaslReservation")
	}

	valid := validator.NewUaslReservation()
	if err := valid.RegisterRequest(uc.ctx, r); err != nil {
		return nil, err
	}

	var requestID value.ModelID
	if r.RequestID != "" {
		id, err := value.NewModelIDFromUUIDString(r.RequestID)
		if err != nil {
			return nil, err
		}
		requestID = id
	} else {

		requestID = value.ModelID("")
	}

	var parentUaslReservationID *value.ModelID
	if r.ParentUaslReservationID != "" {
		id, err := value.NewModelIDFromUUIDString(r.ParentUaslReservationID)
		if err != nil {
			return nil, err
		}
		parentUaslReservationID = &id
	}
	var exUaslSectionID *value.ModelID
	if r.ExUaslSectionID != "" {
		id, err := value.NewModelIDFromUUIDString(r.ExUaslSectionID)
		if err != nil {
			return nil, err
		}
		exUaslSectionID = &id
	}
	var exReservedBy *value.ModelID
	if r.ExReservedBy != "" {
		id, err := value.NewModelIDFromUUIDString(r.ExReservedBy)
		if err != nil {
			return nil, err
		}
		exReservedBy = &id
	}
	var organizationID *value.ModelID
	if r.OrganizationID != "" {
		id, err := value.NewModelIDFromUUIDString(r.OrganizationID)
		if err != nil {
			return nil, err
		}
		organizationID = &id
	}
	var projectID *value.ModelID
	if r.ProjectID != "" {
		id, err := value.NewModelIDFromUUIDString(r.ProjectID)
		if err != nil {
			return nil, err
		}
		projectID = &id
	}
	var operationID *value.ModelID
	if r.OperationID != "" {
		id, err := value.NewModelIDFromUUIDString(r.OperationID)
		if err != nil {
			return nil, err
		}
		operationID = &id

		operation := &model.Operation{}
		if err := uc.operationRepoIF.FindByID(id.ToString(), operation); err != nil {
			return nil, myerror.Wrap(myerror.Database, err, "operation not found")
		}
	}

	startAt, err := time.Parse(time.RFC3339, r.StartAt)
	if err != nil {
		return nil, err
	}
	endAt, err := time.Parse(time.RFC3339, r.EndAt)
	if err != nil {
		return nil, err
	}
	acceptedAtParsed, err := time.Parse(time.RFC3339, r.AcceptedAt)
	if err != nil {
		return nil, err
	}

	acceptedAt := &acceptedAtParsed

	airspaceID, err := value.NewModelIDFromUUIDString(r.AirspaceID)
	if err != nil {
		return nil, err
	}

	var exUaslSectionIDStr *string
	if exUaslSectionID != nil {
		str := exUaslSectionID.ToString()
		exUaslSectionIDStr = &str
	}

	var exUaslIDStr *string
	if r.ExUaslID != "" {
		exUaslIDStr = &r.ExUaslID
	}

	var exAdministratorIDStr *string
	if r.ExAdministratorID != "" {
		exAdministratorIDStr = &r.ExAdministratorID
	}

	var sequence *int
	if r.Sequence != 0 {
		seq := int(r.Sequence)
		sequence = &seq
	}

	reservationStatus := value.ReservationStatus(r.Status)
	logger.LogInfo("Parsed reservation status",
		"reservation_status", reservationStatus.ToString())

	uaslReservation := &model.UaslReservation{
		RequestID:               requestID,
		ParentUaslReservationID: parentUaslReservationID,
		ExUaslSectionID:         exUaslSectionIDStr,
		ExUaslID:                exUaslIDStr,
		ExAdministratorID:       exAdministratorIDStr,
		StartAt:                 startAt,
		EndAt:                   endAt,
		AcceptedAt:              acceptedAt,
		AirspaceID:              airspaceID,
		ExReservedBy:            exReservedBy,
		Status:                  reservationStatus,
		OrganizationID:          organizationID,
		ProjectID:               projectID,
		OperationID:             operationID,
		Sequence:                sequence,
	}
	logger.LogInfo("Built uasl reservation domain object",
		"uasl_reservation_id", uaslReservation.ID.ToString())

	return uaslReservation, nil
}

func (uc *UaslReservationUsecase) ConfirmCompositeUasl(
	ctx context.Context,
	req *converter.ConfirmUaslReservationInput,
) (*model.ReserveCompositeUaslResponse, error) {
	logger.LogInfo("ConfirmCompositeUasl started", "request", req)

	id, err := value.NewModelIDFromUUIDString(req.ID)
	if err != nil {
		return nil, err
	}

	current, children, err := uc.uaslReservationRepoIF.FindByRequestID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load reservation and children: %w", err)
	}
	if current == nil {
		return nil, fmt.Errorf("uasl reservation not found for ID: %s", id.ToString())
	}

	valid := validator.NewUaslReservation()
	if err := valid.ConfirmRequest(ctx, req); err != nil {
		return nil, err
	}

	reservationStatus := value.ReservationStatus(req.Status)
	if reservationStatus != value.RESERVATION_STATUS_RESERVED {
		return nil, myerror.Wrap(myerror.BadParams,
			fmt.Errorf("invalid target status: expected RESERVED, got %s", reservationStatus.ToString()),
			"target status must be RESERVED for confirmation")
	}

	if current.Status != value.RESERVATION_STATUS_PENDING {
		return nil, myerror.Wrap(myerror.BadParams,
			fmt.Errorf("invalid status: expected PENDING, got %s", current.Status.ToString()),
			"reservation status is not PENDING")
	}

	logger.LogInfo("Loaded pending reservation",
		"reservation_id", current.ID.ToString(),
		"status", current.Status.ToString(),
		"children_count", len(children))

	existingMappings, err := uc.externalResourceRepoIF.FindByRequestID(current.RequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch existing resource mappings: %w", err)
	}

	requestBuilder := converter.NewCompositeReservationRequestBuilder()
	vehicleReqs, portReqs, err := requestBuilder.ToConfirmResourcesFromMappings(existingMappings, current)
	if err != nil {
		return nil, fmt.Errorf("failed to build resources from mappings: %w", err)
	}

	logger.LogInfo("Resources loaded from mappings",
		"vehicles_count", len(vehicleReqs),
		"ports_count", len(portReqs))

	isInterConnect := false
	if req != nil {
		isInterConnect = req.IsInterConnect
	}

	logger.LogInfo("isInterConnect flag",
		"is_inter_connect", isInterConnect)

	if !isInterConnect {

		internalAdminsForCredit, err := uc.uaslAdminRepoIF.FindInternalAdministrators(ctx)
		if err != nil {
			logger.LogError("Failed to get internal administrators for credit check", "error", err.Error())
			return nil, fmt.Errorf("failed to get internal administrators for credit check: %w", err)
		}
		if len(internalAdminsForCredit) == 0 {
			return nil, fmt.Errorf("internal administrator not found for credit check")
		}

		internalAdminIDSet := make(map[string]*model.UaslAdministrator, len(internalAdminsForCredit))
		for _, a := range internalAdminsForCredit {
			if a.ExAdministratorID != "" {
				internalAdminIDSet[a.ExAdministratorID] = a
			}
		}
		var internalAdminForCredit *model.UaslAdministrator
		for _, child := range children {
			if child == nil || child.ExAdministratorID == nil || *child.ExAdministratorID == "" {
				continue
			}
			if a, ok := internalAdminIDSet[*child.ExAdministratorID]; ok {
				internalAdminForCredit = a
				break
			}
		}
		if internalAdminForCredit == nil {

			internalAdminForCredit = internalAdminsForCredit[0]
			logger.LogInfo("Credit check: no matching internal admin found in children, using fallback",
				"fallback_ex_administrator_id", internalAdminForCredit.ExAdministratorID,
				"children_count", len(children))
		} else {
			logger.LogInfo("Credit check: matched internal admin from child reservation",
				"ex_administrator_id", internalAdminForCredit.ExAdministratorID)
		}

		if current.Amount == nil {
			return nil, fmt.Errorf("reservation amount is required for credit check")
		}

		creditCheckReq := uc.paymentConverter.ToTransactionEligibilityRequest(
			internalAdminForCredit.ExAdministratorID,
			current.ExReservedBy.ToString(),
			*current.Amount,
		)

		logger.LogInfo("Calling credit check API",
			"provider_id", creditCheckReq.ProviderID,
			"consumer_id", creditCheckReq.ConsumerID,
			"amount", creditCheckReq.Amount)

		creditCheckResp, err := uc.paymentGateway.CheckTransactionEligibility(ctx, creditCheckReq)
		if err != nil {
			logger.LogError("Credit check API call failed", "error", err.Error())
			return nil, fmt.Errorf("credit check failed: %w", err)
		}

		if creditCheckResp.Status == "denied" {
			logger.LogError("Transaction denied by payment service",
				"detail", creditCheckResp.Detail)
			return nil, myerror.Wrap(myerror.BadParams,
				fmt.Errorf("transaction denied: %s", creditCheckResp.Detail),
				"payment service denied the transaction")
		}

		logger.LogInfo("Credit check passed",
			"status", creditCheckResp.Status,
			"detail", creditCheckResp.Detail)
	} else {
		logger.LogInfo("Credit check skipped: isInterConnect=true")
	}

	sagaTx := uc.newSagaTx(
		uc.vehicleGateway,
		uc.portGateway,
		uc.uaslReservationRepoIF,
		uc.externalResourceRepoIF,
		uc.ouranosDiscoveryGW,
		uc.ouranosProxyGW,
		uc.externalUaslDefRepoIF,
		uc.uaslAdminRepoIF,
		uc.externalUaslResourceRepoIF,
	)

	var administratorGroups map[string]*services.AdministratorGroup
	var internalAdmin *model.UaslAdministrator

	if !isInterConnect && uc.interconnectReservationSvc != nil {

		externalGroups, err := uc.interconnectReservationSvc.BuildAdministratorGroupsFromReservedChildren(
			ctx,
			children,
			"ConfirmCompositeUasl",
		)
		if err != nil {
			logger.LogError("ConfirmCompositeUasl: failed to build administrator groups", "error", err)
			return nil, fmt.Errorf("failed to build administrator groups: %w", err)
		}
		administratorGroups = externalGroups
	}

	if administratorGroups == nil {
		administratorGroups = make(map[string]*services.AdministratorGroup)
	}

	logger.LogInfo("Administrator groups organized for confirmation",
		"groups_count", len(administratorGroups),
		"is_interconnect", isInterConnect)

	admins, err := uc.uaslAdminRepoIF.FindInternalAdministrators(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch internal administrator info: %w", err)
	}
	if len(admins) > 0 {

		adminByID := make(map[string]*model.UaslAdministrator, len(admins))
		for _, a := range admins {
			if a.ExAdministratorID != "" {
				adminByID[a.ExAdministratorID] = a
			}
		}

		for _, child := range children {
			if child == nil || child.ExAdministratorID == nil || *child.ExAdministratorID == "" {
				continue
			}
			if a, ok := adminByID[*child.ExAdministratorID]; ok {
				internalAdmin = a
				break
			}
		}

		if internalAdmin == nil {
			internalAdmin = admins[0]
			logger.LogInfo("ConfirmCompositeUasl: no matching internal admin in children, using fallback",
				"fallback_ex_administrator_id", internalAdmin.ExAdministratorID)
		}
	}

	for adminID, adminGroup := range administratorGroups {
		if adminGroup == nil || adminGroup.IsInternal {
			continue
		}
		logger.LogInfo("Confirming external administrator group",
			"administrator_id", adminID)

		confirmData, err := uc.interconnectReservationSvc.ConfirmExternalUasl(
			ctx,
			*adminGroup,
			current.RequestID.ToString(),
			sagaTx.SagaOrchestrator,
		)
		if err != nil {
			logger.LogError("External confirmation failed (executing compensation)",
				"administrator_id", adminGroup.AdministratorID,
				"error", err,
			)
			sagaTx.SagaOrchestrator.Rollback(ctx)
			return nil, fmt.Errorf("external confirmation for administrator %s failed: %w", adminGroup.AdministratorID, err)
		}

		if confirmData != nil {
			adminGroup.ConfirmationData = confirmData

			services.MergeConformityFromExternal(current, confirmData)

			var vehiclesCount, portsCount int
			for _, destRes := range confirmData.DestinationReservations {
				vehiclesCount += len(destRes.Vehicles)
				portsCount += len(destRes.Ports)
			}
			logger.LogInfo("External confirmation data received",
				"administrator_id", adminGroup.AdministratorID,
				"pricing_rule_version", confirmData.PricingRuleVersion,
				"vehicles_count", vehiclesCount,
				"ports_count", portsCount,
				"is_interconnect", isInterConnect)
		}

		logger.LogInfo("External UASL reservation confirmed",
			"administrator_id", adminGroup.AdministratorID,
			"request_id", current.RequestID.ToString())
	}

	internalAdminIDForPort := make(map[string]struct{}, len(admins))
	for _, a := range admins {
		if a.ExAdministratorID != "" {
			internalAdminIDForPort[a.ExAdministratorID] = struct{}{}
		}
	}
	internalPortReqs := make([]model.PortReservationRequest, 0)
	for _, portReq := range portReqs {

		for i := range existingMappings {
			m := &existingMappings[i]
			if m.ResourceType != model.ExternalResourceTypePort || m.ExResourceID != portReq.PortID {
				continue
			}
			if m.UsageType == nil {
				continue
			}
			usageType := util.SafeIntToInt32(*m.UsageType)
			if usageType != portReq.UsageType {
				continue
			}

			if _, ok := internalAdminIDForPort[m.ExAdministratorID]; ok {
				internalPortReqs = append(internalPortReqs, portReq)
			}
			break
		}
	}

	if len(administratorGroups) == 0 {
		internalPortReqs = portReqs
	}

	logger.LogInfo("Internal resource requests extracted",
		"total_vehicles", len(vehicleReqs),
		"total_ports", len(portReqs),
		"internal_ports", len(internalPortReqs))

	resourceConflict, err := uc.resourceChecker.CheckCompositeResourcesAvailability(
		ctx,
		vehicleReqs,
		internalPortReqs,
	)
	if err != nil {
		logger.LogError("composite availability check failed - attempting rollback", "error", err)
		sagaTx.SagaOrchestrator.Rollback(ctx)
		return nil, fmt.Errorf("composite availability check failed: %w", err)
	}
	if resourceConflict != nil {
		logger.LogError("resource conflict detected during confirmation - attempting rollback",
			"conflict_type", resourceConflict.ConflictType,
			"conflicted_ids", resourceConflict.ConflictedIDs)
		sagaTx.SagaOrchestrator.Rollback(ctx)
		return &model.ReserveCompositeUaslResponse{
			ConflictType:            resourceConflict.ConflictType,
			ConflictedResourceIds:   resourceConflict.ConflictedIDs,
			ConflictedFlightPlanIDs: []string{},
			ParentUaslReservation:   nil,
		}, nil
	}

	logger.LogInfo("GSW resource availability check passed")

	vehicleConv := converter.NewVehicleReservationConverter()
	portConv := converter.NewPortReservationConverter()
	var externalResourceMappings []*model.ExternalResourceReservation

	vehicleReservationIDs := make(map[string]string)
	portReservationIDs := make(map[string]string)

	for _, vehicleReq := range vehicleReqs {
		handle, mappings, err := sagaTx.ReserveVehicle(ctx, current.RequestID, vehicleReq)
		if err != nil {
			logger.LogError("CRITICAL: vehicle reservation failed - attempting rollback",
				"error", err,
				"vehicle_id", vehicleReq.VehicleID,
				"request_id", current.RequestID.ToString(),
			)
			sagaTx.SagaOrchestrator.Rollback(ctx)
			return nil, fmt.Errorf("vehicle reservation failed: %w", err)
		}

		vehicleReservationIDs[vehicleReq.VehicleID] = handle.ID
		externalResourceMappings = append(externalResourceMappings, mappings...)
		logger.LogInfo("Vehicle reserved",
			"vehicle_id", vehicleReq.VehicleID,
			"reservation_handle_id", handle.ID)
	}

	type portReserveKey struct {
		portID string
		from   time.Time
		to     time.Time
	}

	portReqGroups := make(map[portReserveKey][]model.PortReservationRequest)
	portReqGroupOrder := make([]portReserveKey, 0, len(internalPortReqs))
	for _, portReq := range internalPortReqs {
		key := portReserveKey{
			portID: portReq.PortID,
			from:   portReq.ReservationTimeFrom,
			to:     portReq.ReservationTimeTo,
		}
		if _, exists := portReqGroups[key]; !exists {
			portReqGroupOrder = append(portReqGroupOrder, key)
		}
		portReqGroups[key] = append(portReqGroups[key], portReq)
	}

	for _, key := range portReqGroupOrder {
		group := portReqGroups[key]

		representative := group[0]
		handle, mappings, err := sagaTx.ReservePort(ctx, current.RequestID, representative)
		if err != nil {
			logger.LogError("CRITICAL: port reservation failed - attempting rollback",
				"error", err,
				"port_id", representative.PortID,
				"request_id", current.RequestID.ToString(),
			)
			sagaTx.SagaOrchestrator.Rollback(ctx)
			return nil, fmt.Errorf("port reservation failed: %w", err)
		}
		portReservationIDs[representative.PortID] = handle.ID
		externalResourceMappings = append(externalResourceMappings, mappings...)
		logger.LogInfo("Port reserved",
			"port_id", representative.PortID,
			"reservation_handle_id", handle.ID,
			"group_size", len(group))

		for _, dup := range group[1:] {
			dupRow, buildErr := model.NewExternalResourceReservation(current.RequestID, handle.ID, model.ExternalResourceTypePort)
			if buildErr != nil {
				logger.LogError("Failed to build duplicate port mapping",
					"error", buildErr,
					"port_id", dup.PortID,
					"usage_type", dup.UsageType)
				continue
			}
			dupRow.ExResourceID = dup.PortID
			usageType := int(dup.UsageType)
			dupRow.UsageType = &usageType
			if !dup.ReservationTimeFrom.IsZero() {
				dupRow.StartAt = &dup.ReservationTimeFrom
			}
			if !dup.ReservationTimeTo.IsZero() {
				dupRow.EndAt = &dup.ReservationTimeTo
			}
			externalResourceMappings = append(externalResourceMappings, dupRow)
			logger.LogInfo("Duplicate port mapping generated with same handle ID",
				"port_id", dup.PortID,
				"usage_type", dup.UsageType,
				"reservation_handle_id", handle.ID)
		}
	}

	logger.LogInfo("All GSW resources reserved successfully",
		"vehicles_count", len(vehicleReqs),
		"ports_count", len(internalPortReqs))

	var totalUaslAmount int32
	for _, child := range children {
		if child.Amount != nil {
			v := *child.Amount
			if v > maxInt32 {
				logger.LogError("amount value exceeds int32 max, capping to MaxInt32", "amount", v)
				totalUaslAmount += int32(maxInt32)
			} else if v < minInt32 {
				logger.LogError("amount value below int32 min, capping to MinInt32", "amount", v)
				totalUaslAmount += int32(minInt32)
			} else {
				totalUaslAmount += util.SafeIntToInt32(v)
			}
		}
	}

	totalAmountInt := int(totalUaslAmount)

	logger.LogInfo("Using pending reservation pricing (no recalculation)",
		"total_uasl_amount", totalUaslAmount,
		"total_amount", totalAmountInt)

	allParents, err := uc.uaslReservationRepoIF.FindParentsByRequestID(current.RequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to load all parent reservations: %w", err)
	}

	logger.LogInfo("Confirming all parent reservations",
		"request_id", current.RequestID.ToString(),
		"parents_count", len(allParents))

	now := time.Now()
	reservationsToUpdate := []*model.UaslReservation{}

	for _, parent := range allParents {
		parent.Status = value.RESERVATION_STATUS_RESERVED
		parent.AcceptedAt = &now
		parent.FixedAt = &now
		reservationsToUpdate = append(reservationsToUpdate, parent)
	}

	for _, child := range children {
		child.AcceptedAt = &now
		child.FixedAt = &now
		reservationsToUpdate = append(reservationsToUpdate, child)
	}

	logger.LogInfo("Batch update prepared",
		"parents_count", len(allParents),
		"children_count", len(children),
		"total_count", len(reservationsToUpdate))

	updatedReservations, err := uc.uaslReservationRepoIF.UpdateBatch(reservationsToUpdate)
	if err != nil {
		logger.LogError("CRITICAL: batch update of reservations failed - attempting rollback",
			"error", err,
			"parent_reservation_id", current.ID.ToString(),
			"total_count", len(reservationsToUpdate),
		)
		sagaTx.SagaOrchestrator.Rollback(ctx)
		return nil, fmt.Errorf("batch update of reservations failed: %w", err)
	}

	originalFlightPurpose := current.FlightPurpose

	for _, updated := range updatedReservations {
		if updated != nil && updated.ID == current.ID {
			current = updated
			current.FlightPurpose = originalFlightPurpose
			break
		}
	}

	sagaTx.SagaOrchestrator.RecordSuccess(retry.Step{
		Name: fmt.Sprintf("ConfirmReservation-%s", current.ID.ToString()),
		Rollback: func(rc context.Context) error {

			current.Status = value.RESERVATION_STATUS_PENDING
			current.AcceptedAt = nil
			current.FixedAt = nil

			rollbackList := []*model.UaslReservation{current}
			for _, child := range children {
				child.AcceptedAt = nil
				child.FixedAt = nil
				rollbackList = append(rollbackList, child)
			}

			_, err := uc.uaslReservationRepoIF.UpdateBatch(rollbackList)
			return err
		},
		Metadata: map[string]interface{}{
			"parent_reservation_id": current.ID.ToString(),
			"children_count":        len(children),
			"action":                "confirm_reservation",
		},
	})

	logger.LogInfo("Reservation confirmed (batch update)",
		"parent_reservation_id", current.ID.ToString(),
		"parent_status", current.Status.ToString(),
		"children_count", len(children),
		"total_updated", len(updatedReservations),
		"amount", current.Amount)

	buildResourceKey := func(resourceType model.ExternalResourceType, resourceID string, usageType *int) string {
		if resourceType == model.ExternalResourceTypePort {
			if usageType != nil {
				return fmt.Sprintf("%s:%s:%d", resourceType, resourceID, *usageType)
			}

			return fmt.Sprintf("%s:%s:nil", resourceType, resourceID)
		}
		return fmt.Sprintf("%s:%s", resourceType, resourceID)
	}

	newReservationIDMap := make(map[string]string)
	if len(externalResourceMappings) > 0 {
		for _, newMapping := range externalResourceMappings {
			key := buildResourceKey(newMapping.ResourceType, newMapping.ExResourceID, newMapping.UsageType)
			newReservationIDMap[key] = newMapping.ExReservationID
		}
	}

	for _, adminGroup := range administratorGroups {
		if adminGroup != nil && !adminGroup.IsInternal && adminGroup.ConfirmationData != nil {

			for _, destRes := range adminGroup.ConfirmationData.DestinationReservations {

				for _, port := range destRes.Ports {
					if port.PortID != "" && port.ReservationID != "" {
						usageType := port.UsageType
						key := buildResourceKey(model.ExternalResourceTypePort, port.PortID, &usageType)
						newReservationIDMap[key] = port.ReservationID
						logger.LogInfo("External port reservation ID extracted",
							"administrator_id", adminGroup.AdministratorID,
							"port_id", port.PortID,
							"usage_type", port.UsageType,
							"reservation_id", port.ReservationID)
					}
				}

				for _, vehicle := range destRes.Vehicles {
					if vehicle.VehicleID != "" && vehicle.ReservationID != "" {
						key := string(model.ExternalResourceTypeVehicle) + ":" + vehicle.VehicleID
						newReservationIDMap[key] = vehicle.ReservationID
						logger.LogInfo("External vehicle reservation ID extracted",
							"administrator_id", adminGroup.AdministratorID,
							"vehicle_id", vehicle.VehicleID,
							"reservation_id", vehicle.ReservationID)
					}
				}
			}
		}
	}

	if len(newReservationIDMap) > 0 {
		updatedMappings := make([]*model.ExternalResourceReservation, 0, len(existingMappings))
		type mappingSnapshot struct {
			ID              value.ModelID
			ExReservationID string
			UpdatedAt       time.Time
		}
		beforeUpdate := make([]mappingSnapshot, 0, len(existingMappings))

		for i := range existingMappings {

			key := buildResourceKey(existingMappings[i].ResourceType, existingMappings[i].ExResourceID, existingMappings[i].UsageType)
			if newExReservationID, ok := newReservationIDMap[key]; ok {

				if existingMappings[i].ExReservationID != newExReservationID {
					beforeUpdate = append(beforeUpdate, mappingSnapshot{
						ID:              existingMappings[i].ID,
						ExReservationID: existingMappings[i].ExReservationID,
						UpdatedAt:       existingMappings[i].UpdatedAt,
					})

					existingMappings[i].ExReservationID = newExReservationID
					existingMappings[i].UpdatedAt = time.Now()
					updatedMappings = append(updatedMappings, &existingMappings[i])
				}
			}
		}

		if len(updatedMappings) > 0 {
			if _, err := uc.externalResourceRepoIF.UpdateBatch(updatedMappings); err != nil {
				logger.LogError("CRITICAL: external resource mappings update failed - attempting rollback",
					"error", err,
					"request_id", current.RequestID.ToString(),
				)
				sagaTx.SagaOrchestrator.Rollback(ctx)
				return nil, fmt.Errorf("external resource mappings update failed: %w", err)
			}

			snapshotByID := make(map[value.ModelID]mappingSnapshot, len(beforeUpdate))
			for _, snapshot := range beforeUpdate {
				snapshotByID[snapshot.ID] = snapshot
			}

			sagaTx.SagaOrchestrator.RecordSuccess(retry.Step{
				Name: fmt.Sprintf("UpdateExternalResourceMappings-%s", current.RequestID.ToString()),
				Rollback: func(rc context.Context) error {
					restoreMappings := make([]*model.ExternalResourceReservation, 0, len(updatedMappings))
					for _, mapping := range updatedMappings {
						if snapshot, ok := snapshotByID[mapping.ID]; ok {
							mappingCopy := *mapping
							mappingCopy.ExReservationID = snapshot.ExReservationID
							mappingCopy.UpdatedAt = snapshot.UpdatedAt
							restoreMappings = append(restoreMappings, &mappingCopy)
						}
					}

					if len(restoreMappings) > 0 {
						_, err := uc.externalResourceRepoIF.UpdateBatch(restoreMappings)
						if err != nil {
							logger.LogError("Failed to restore external resource mappings",
								"error", err,
								"request_id", current.RequestID.ToString(),
							)
						}
						return err
					}
					return nil
				},
				Metadata: map[string]interface{}{
					"request_id":      current.RequestID.ToString(),
					"mappings_count":  len(updatedMappings),
					"snapshots_count": len(beforeUpdate),
					"action":          "update_external_resource_mappings",
				},
			})

			logger.LogInfo("External resource mappings updated",
				"mappings_count", len(updatedMappings),
				"total_reservation_ids", len(newReservationIDMap))
		}
	}

	logger.LogInfo("UASL reservation confirmed successfully",
		"parent_id", current.ID.ToString(),
		"children_count", len(children),
		"status", current.Status.ToString())

	var vehicleDetails []*model.VehicleReservationDetail
	var portDetails []*model.PortReservationDetail

	var internalAdminID string
	if internalAdmin != nil {
		internalAdminID = internalAdmin.ExAdministratorID
	}

	for i := range existingMappings {
		mapping := &existingMappings[i]

		if !isInterConnect && internalAdminID != "" && mapping.ExAdministratorID != internalAdminID {
			continue
		}

		if mapping.ResourceType == model.ExternalResourceTypeVehicle {
			vehicleDetail := &model.VehicleReservationDetail{
				VehicleID:     mapping.ExResourceID,
				ReservationID: mapping.ExReservationID,
				VehicleName:   mapping.ResourceName,
			}
			if mapping.StartAt != nil {
				vehicleDetail.StartAt = *mapping.StartAt
			}
			if mapping.EndAt != nil {
				vehicleDetail.EndAt = *mapping.EndAt
			}
			if mapping.Amount != nil {
				vehicleDetail.Amount = util.SafeIntToInt32(*mapping.Amount)
			}
			vehicleDetails = append(vehicleDetails, vehicleDetail)
			logger.LogInfo("Internal vehicle reservation added to response",
				"vehicle_id", mapping.ExResourceID,
				"vehicle_name", mapping.ResourceName,
				"reservation_id", mapping.ExReservationID,
				"amount", mapping.Amount,
				"ex_administrator_id", mapping.ExAdministratorID)
		} else if mapping.ResourceType == model.ExternalResourceTypePort {
			portDetail := &model.PortReservationDetail{
				PortID:            mapping.ExResourceID,
				ReservationID:     mapping.ExReservationID,
				PortName:          mapping.ResourceName,
				ExAdministratorID: mapping.ExAdministratorID,
			}
			if mapping.StartAt != nil {
				portDetail.StartAt = *mapping.StartAt
			}
			if mapping.EndAt != nil {
				portDetail.EndAt = *mapping.EndAt
			}
			if mapping.Amount != nil {
				portDetail.Amount = util.SafeIntToInt32(*mapping.Amount)
			}
			if mapping.UsageType != nil {
				portDetail.UsageType = util.SafeInt32FromPtr(mapping.UsageType)
			}
			portDetails = append(portDetails, portDetail)
			logger.LogInfo("Internal port reservation added to response",
				"port_id", mapping.ExResourceID,
				"port_name", mapping.ResourceName,
				"reservation_id", mapping.ExReservationID,
				"usage_type", mapping.UsageType,
				"amount", mapping.Amount,
				"ex_administrator_id", mapping.ExAdministratorID)
		}
	}

	for _, adminGroup := range administratorGroups {
		if adminGroup != nil && !adminGroup.IsInternal && adminGroup.ConfirmationData != nil {

			var allVehicles []model.VehicleReservationElement
			var allPorts []model.PortReservationElement
			for _, destRes := range adminGroup.ConfirmationData.DestinationReservations {
				allVehicles = append(allVehicles, destRes.Vehicles...)

				for _, p := range destRes.Ports {
					p.ExAdministratorID = adminGroup.AdministratorID
					allPorts = append(allPorts, p)
				}
			}

			logger.LogInfo("Processing external administrator confirmation data",
				"administrator_id", adminGroup.AdministratorID,
				"vehicles_count", len(allVehicles),
				"ports_count", len(allPorts))

			externalVehicleDetails := vehicleConv.ToVehicleReservationDetailsFromExternalElements(allVehicles)
			vehicleDetails = append(vehicleDetails, externalVehicleDetails...)

			externalPortDetails := portConv.ToPortReservationDetailsFromExternalElements(allPorts)
			portDetails = append(portDetails, externalPortDetails...)
		}
	}

	vehicleElements := vehicleConv.ToVehicleElements(vehicleDetails)
	portElements := portConv.ToPortElements(portDetails)

	logger.LogInfo("Converted resource details to elements",
		"vehicle_elements_count", len(vehicleElements),
		"port_elements_count", len(portElements))

	domainConv := converter.NewUaslReservationDomain()

	response := domainConv.ToCompositeUaslReservationResponse(
		current,
		children,
		nil,
		vehicleElements,
		portElements,
	)

	logger.LogInfo("ConfirmCompositeUasl completed successfully",
		"reservation_id", response.ParentUaslReservation.ID,
	)

	return response, nil
}

func (uc *UaslReservationUsecase) ListByOperator(
	ctx context.Context,
	req *converter.ListByOperatorInput,
) (*model.ListUaslReservationsResponse, error) {
	logger.LogInfo("ListByOperator request received",
		"operator_id", req.OperatorID,
		"page", req.Page)

	type validStruct struct {
		OperatorID string `validate:"required,model-id"`
	}
	validate, err := baseValidator.New()
	if err != nil {
		logger.LogError("Failed to create validator",
			"error", err.Error())
		return nil, myerror.Wrap(myerror.Internal, err, "failed to create validator")
	}
	if err := validate.Struct(validStruct{
		OperatorID: req.OperatorID,
	}); err != nil {
		logger.LogError("Invalid operator_id format",
			"operator_id", req.OperatorID,
			"error", err.Error())
		return nil, myerror.Wrap(myerror.BadParams, err, "invalid operator_id format")
	}

	const perPage int32 = 20

	if req.Page <= 0 {
		req.Page = 1
	}

	internalAdminsForList, err := uc.uaslAdminRepoIF.FindInternalAdministrators(ctx)
	if err != nil {
		logger.LogError("Failed to find internal administrators",
			"error", err.Error())
		return nil, myerror.Wrap(myerror.Database, err, "failed to find internal administrators")
	}
	internalAdminIDs := make([]string, 0, len(internalAdminsForList))
	for _, a := range internalAdminsForList {
		if a.ExAdministratorID != "" {
			internalAdminIDs = append(internalAdminIDs, a.ExAdministratorID)
		}
	}

	items, total, err := uc.uaslReservationRepoIF.ListByOperator(
		ctx,
		req.OperatorID,
		internalAdminIDs,
		req.Page,
		perPage,
	)
	if err != nil {
		logger.LogError("Failed to list reservations by operator",
			"operator_id", req.OperatorID,
			"error", err.Error())
		return nil, err
	}

	domainConv := converter.NewUaslReservationDomain()
	itemsProto := domainConv.ToUaslReservationListItems(items)

	if len(items) > 0 && uc.interconnectReservationSvc != nil {
		uaslIDSet := make(map[string]struct{})
		for _, batch := range items {
			if batch == nil {
				continue
			}
			for _, child := range batch.Children {
				if child == nil || child.ExUaslID == nil || *child.ExUaslID == "" {
					continue
				}
				uaslIDSet[*child.ExUaslID] = struct{}{}
			}
		}

		if len(uaslIDSet) > 0 {
			uaslIDs := make([]string, 0, len(uaslIDSet))
			for uaslID := range uaslIDSet {
				uaslIDs = append(uaslIDs, uaslID)
			}

			internalUaslIDs, externalUaslIDs, classifyErr := uc.interconnectReservationSvc.ClassifyInternalExternalUaslIDs(ctx, uaslIDs)
			if classifyErr != nil {
				logger.LogError("ListByOperator: failed to classify internal/external uasl IDs",
					"error", classifyErr)
			} else {
				adminResolution := uc.interconnectReservationSvc.ResolveAdministratorsForAvailability(ctx, externalUaslIDs, internalUaslIDs)
				if err := uc.interconnectReservationSvc.EnrichListByOperatorWithExternalData(ctx, items, *adminResolution); err != nil {
					logger.LogError("Failed to enrich list by operator with external data",
						"error", err.Error())
				} else {

					itemsProto = domainConv.ToUaslReservationListItems(items)
				}
			}
		}
	}

	lastPage := total / int64(perPage)
	if total%int64(perPage) != 0 {
		lastPage++
	}

	return &model.ListUaslReservationsResponse{
		Result: itemsProto,
		PageInfo: &model.PaginationInfo{
			CurrentPage: req.Page,
			LastPage:    util.SafeIntToInt32(int(lastPage)),
			PerPage:     perPage,
			Total:       util.SafeIntToInt32(int(total)),
		},
	}, nil
}

func (uc *UaslReservationUsecase) ListAdmin(
	ctx context.Context,
	req *converter.ListAdminInput,
) (*model.ListUaslReservationsResponse, error) {

	const perPage int32 = 20

	if req.Page <= 0 {
		req.Page = 1
	}

	batches, total, err := uc.uaslReservationRepoIF.ListAdmin(
		ctx,
		req.Page,
		perPage,
	)
	if err != nil {
		return nil, err
	}

	domainConv := converter.NewUaslReservationDomain()
	items := domainConv.ToUaslReservationListItems(batches)

	if len(items) > 0 {
		origins := make([]services.ReservationParentOrigin, 0, len(batches))
		uaslIDSet := make(map[string]struct{})
		for _, batch := range batches {
			if batch == nil {
				continue
			}
			if batch.Parent != nil {
				parentUaslID := ""
				if batch.Parent.ExUaslID != nil {
					parentUaslID = *batch.Parent.ExUaslID
					if parentUaslID != "" {
						uaslIDSet[parentUaslID] = struct{}{}
					}
				}
				parentAdminID := ""
				if batch.Parent.ExAdministratorID != nil {
					parentAdminID = *batch.Parent.ExAdministratorID
				}
				origins = append(origins, services.ReservationParentOrigin{
					UaslID:          parentUaslID,
					AdministratorID: parentAdminID,
				})
			}
			for _, child := range batch.Children {
				if child == nil || child.ExUaslID == nil || *child.ExUaslID == "" {
					continue
				}
				uaslIDSet[*child.ExUaslID] = struct{}{}
			}
		}

		var allExternalItems []model.ExternalReservationListItem
		if uc.interconnectReservationSvc != nil && len(uaslIDSet) > 0 {
			uaslIDs := make([]string, 0, len(uaslIDSet))
			for uaslID := range uaslIDSet {
				uaslIDs = append(uaslIDs, uaslID)
			}

			internalUaslIDs, externalUaslIDs, classifyErr := uc.interconnectReservationSvc.ClassifyInternalExternalUaslIDs(ctx, uaslIDs)
			if classifyErr != nil {
				logger.LogError("ListAdmin: failed to classify internal/external uasl IDs",
					"error", classifyErr)
			} else {
				listAdminResolution := uc.interconnectReservationSvc.ResolveAdministratorsForAvailability(ctx, externalUaslIDs, internalUaslIDs)

				if len(origins) > 0 {
					originResolution := uc.interconnectReservationSvc.ResolveAdminResolutionByParentOrigins(ctx, origins)
					if originResolution != nil {
						for k, v := range originResolution.ExternalServices {
							listAdminResolution.ExternalServices[k] = v
						}
						for k, v := range originResolution.UaslIDToAdministratorID {
							listAdminResolution.UaslIDToAdministratorID[k] = v
						}
					}
				}
				allExternalItems, err = uc.interconnectReservationSvc.EnrichListAdminWithExternalData(ctx, batches, *listAdminResolution)
				if err != nil {
					logger.LogError("Failed to enrich list admin with external data",
						"error", err.Error())
				}
			}
		}

		if len(allExternalItems) > 0 {

			externalItems := domainConv.ToExternalReservationListItems(allExternalItems)
			externalItemsMap := make(map[string]*model.UaslReservationListItemMsg)
			for _, item := range externalItems {
				if item != nil && item.ParentUaslReservation != nil {
					externalItemsMap[item.ParentUaslReservation.RequestID] = item
				}
			}

			for i, item := range items {
				if item != nil && item.ParentUaslReservation != nil {
					if extItem, ok := externalItemsMap[item.ParentUaslReservation.RequestID]; ok {

						localSectionMap := make(map[string]*model.UaslReservationMessage)
						for _, localChild := range item.ChildUaslReservations {
							if localChild != nil && localChild.ExUaslSectionID != "" {
								localSectionMap[localChild.ExUaslSectionID] = localChild
							}
						}
						for _, extSection := range extItem.ChildUaslReservations {
							if extSection == nil || extSection.ExUaslSectionID == "" {
								continue
							}
							if localChild, ok := localSectionMap[extSection.ExUaslSectionID]; ok {

								if extSection.StartAt != localChild.StartAt || extSection.EndAt != localChild.EndAt {
									logger.LogError("Reservation time mismatch detected between external API and local DB",
										"request_id", item.ParentUaslReservation.RequestID,
										"ex_uasl_section_id", extSection.ExUaslSectionID,
										"local_start_at", localChild.StartAt,
										"external_start_at", extSection.StartAt,
										"local_end_at", localChild.EndAt,
										"external_end_at", extSection.EndAt)
								}

								if extSection.Amount != localChild.Amount {
									logger.LogError("Reservation amount mismatch detected",
										"request_id", item.ParentUaslReservation.RequestID,
										"ex_uasl_section_id", extSection.ExUaslSectionID,
										"local_amount", localChild.Amount,
										"external_amount", extSection.Amount)
								}
							}
						}

						if extItem.ParentUaslReservation.Status != item.ParentUaslReservation.Status {
							logger.LogError("Reservation status mismatch detected",
								"request_id", item.ParentUaslReservation.RequestID,
								"local_status", item.ParentUaslReservation.Status,
								"external_status", extItem.ParentUaslReservation.Status)
						}

						type resourceInfo struct {
							Name   string
							Amount int32
						}
						localPortMap := make(map[string]resourceInfo)
						localVehicleMap := make(map[string]resourceInfo)
						for _, v := range item.Vehicles {
							if v != nil {
								localVehicleMap[v.VehicleID] = resourceInfo{
									Name:   v.Name,
									Amount: int32(v.Amount),
								}
							}
						}
						for _, p := range item.Ports {
							if p != nil {
								localPortMap[p.PortID] = resourceInfo{
									Name:   p.Name,
									Amount: int32(p.Amount),
								}
							}
						}

						for _, extVehicle := range extItem.Vehicles {
							if extVehicle != nil {
								if localInfo, ok := localVehicleMap[extVehicle.VehicleID]; ok {
									if int32(extVehicle.Amount) != localInfo.Amount {
										logger.LogError("Vehicle reservation amount mismatch detected",
											"request_id", item.ParentUaslReservation.RequestID,
											"vehicle_id", extVehicle.VehicleID,
											"local_amount", localInfo.Amount,
											"external_amount", extVehicle.Amount)
									}
								}
							}
						}

						for _, extPort := range extItem.Ports {
							if extPort != nil {
								if localInfo, ok := localPortMap[extPort.PortID]; ok {
									if int32(extPort.Amount) != localInfo.Amount {
										logger.LogError("Port reservation amount mismatch detected",
											"request_id", item.ParentUaslReservation.RequestID,
											"port_id", extPort.PortID,
											"local_amount", localInfo.Amount,
											"external_amount", extPort.Amount)
									}
								}
							}
						}

						for _, v := range extItem.Vehicles {
							if v != nil && v.Name == "" {
								if localInfo, ok := localVehicleMap[v.VehicleID]; ok {
									v.Name = localInfo.Name
								}
							}
						}
						for _, p := range extItem.Ports {
							if p != nil && p.Name == "" {
								if localInfo, ok := localPortMap[p.PortID]; ok {
									p.Name = localInfo.Name
								}
							}
						}

						if uc.interconnectReservationSvc != nil {
							items[i] = uc.interconnectReservationSvc.MergeListAdminLocalAndExternal(item, extItem)
						}
					}
				}
			}
		}
	}

	lastPage := total / int64(perPage)
	if total%int64(perPage) != 0 {
		lastPage++
	}

	return &model.ListUaslReservationsResponse{
		Result: items,
		PageInfo: &model.PaginationInfo{
			CurrentPage: req.Page,
			LastPage:    util.SafeIntToInt32(int(lastPage)),
			PerPage:     perPage,
			Total:       util.SafeIntToInt32(int(total)),
		},
	}, nil
}

func (uc *UaslReservationUsecase) GetAvailability(
	ctx context.Context,
	req *converter.GetAvailabilityInput,
) (*model.GetAvailabilityResponse, error) {
	logger.LogInfo("GetAvailability started",
		"sections_count", len(req.UaslSections),
		"vehicle_ids_count", len(req.VehicleIDs),
		"port_ids_count", len(req.PortIDs),
		"is_inter_connect", req.IsInterConnect)

	valid := validator.NewUaslReservation()
	if err := valid.AvailabilityRequest(ctx, req); err != nil {
		return nil, myerror.Wrap(myerror.BadParams, err, "validation failed")
	}

	if len(req.UaslSections) == 0 {
		return &model.GetAvailabilityResponse{
			UaslSections: []*model.AvailabilityItem{},
			Vehicles:     []*model.VehicleAvailabilityItem{},
			Ports:        []*model.PortAvailabilityItem{},
		}, nil
	}

	sections := req.UaslSections
	vehicleIDs := req.VehicleIDs
	portIDs := req.PortIDs

	uaslIDs := make([]string, 0, len(sections))
	for _, s := range sections {
		if s.UaslID != "" {
			uaslIDs = append(uaslIDs, s.UaslID)
		}
	}

	targets := services.IngestTargets{
		UaslIDs:    uaslIDs,
		VehicleIDs: vehicleIDs,
		PortIDs:    portIDs,
	}

	if uc.externalIngestService != nil {
		if err := uc.externalIngestService.IngestForGetAvailability(ctx, targets); err != nil {
			logger.LogError("external ingest failed", "error", err)

		}
	}

	var adminResolution *services.AvailabilityAdminResolution
	if uc.interconnectReservationSvc != nil && len(uaslIDs) > 0 {
		internalUaslIDs, externalUaslIDs, err := uc.interconnectReservationSvc.ClassifyInternalExternalUaslIDs(ctx, uaslIDs)
		if err != nil {
			logger.LogError("GetAvailability: failed to classify uasl IDs", "error", err)
			return nil, myerror.Wrap(myerror.Internal, err, "failed to classify uasl IDs")
		}

		logger.LogInfo("GetAvailability: classified uasl IDs",
			"total", len(uaslIDs),
			"internal", len(internalUaslIDs),
			"external", len(externalUaslIDs))

		if len(externalUaslIDs) > 0 {
			adminResolution = uc.interconnectReservationSvc.ResolveAdministratorsForAvailability(ctx, externalUaslIDs, internalUaslIDs)
		}
	}

	var allAvailabilityItems []*model.AvailabilityItem
	var allVehicleReservations []model.VehicleReservationInfo
	var allPortReservations []model.PortReservationInfo

	conv := converter.NewUaslReservationDomain()

	var internalVehicles []string
	var internalPorts []string
	var internalSectionIDs []string

	if req.IsInterConnect {

		internalSectionIDs = make([]string, 0, len(sections))
		for _, s := range sections {
			internalSectionIDs = append(internalSectionIDs, s.UaslSectionID)
		}

		internalVehicles = vehicleIDs
		internalPorts = portIDs

	} else {

		availabilitySections := make([]model.AvailabilitySection, 0, len(sections))
		for _, s := range sections {
			availabilitySections = append(availabilitySections, model.AvailabilitySection{
				UaslID:        s.UaslID,
				UaslSectionID: s.UaslSectionID,
			})
		}

		externalItems, externalVehicleItems, externalPortItems, internalSections, externalInternalVehicles, externalInternalPorts, err := uc.interconnectReservationSvc.FetchExternalAvailability(
			ctx,
			adminResolution,
			availabilitySections,
			vehicleIDs,
			portIDs,
		)
		if err != nil {
			return nil, err
		}

		internalVehicles = externalInternalVehicles
		internalPorts = externalInternalPorts

		for i := range externalItems {
			allAvailabilityItems = append(allAvailabilityItems, &externalItems[i])
		}

		if len(externalVehicleItems) > 0 {
			logger.LogInfo("External vehicle availability received",
				"count", len(externalVehicleItems),
			)
			for _, extVehicle := range externalVehicleItems {
				reservationID := extVehicle.ReservationID
				allVehicleReservations = append(allVehicleReservations, model.VehicleReservationInfo{
					ReservationID: &reservationID,
					VehicleID:     extVehicle.VehicleID,
					VehicleName:   extVehicle.Name,
					StartAt:       extVehicle.StartAt,
					EndAt:         extVehicle.EndAt,
				})
			}
		}

		if len(externalPortItems) > 0 {
			logger.LogInfo("External port availability received",
				"count", len(externalPortItems),
			)
			for _, extPort := range externalPortItems {
				reservationID := extPort.ReservationID
				allPortReservations = append(allPortReservations, model.PortReservationInfo{
					ReservationID: &reservationID,
					PortID:        extPort.PortID,
					PortName:      extPort.Name,
					VehicleID:     nil,
					UsageType:     0,
					StartAt:       extPort.StartAt,
					EndAt:         extPort.EndAt,
				})
			}
		}

		if len(internalSections) > 0 {
			internalSectionIDs = make([]string, 0, len(internalSections))
			for _, s := range internalSections {
				internalSectionIDs = append(internalSectionIDs, s.UaslSectionID)
			}
		}
	}

	if len(internalSectionIDs) > 0 {

		reservations, err := uc.uaslReservationRepoIF.FindByUaslSectionIDs(internalSectionIDs, nil)
		if err != nil {
			return nil, fmt.Errorf("FindByUaslSectionIDs failed: %w", err)
		}

		if len(reservations) > 0 {

			items := conv.ToAvailabilityItems(reservations)
			allAvailabilityItems = append(allAvailabilityItems, items...)
		}
	}

	if len(internalVehicles) > 0 {
		internalVehicleReservations, err := uc.resourceChecker.FetchVehicleReservations(ctx, internalVehicles)
		if err != nil {
			logger.LogError("Failed to fetch internal vehicle reservations", "error", err)
		} else {
			allVehicleReservations = append(allVehicleReservations, internalVehicleReservations...)
		}
	}

	if len(internalPorts) > 0 {
		internalPortReservations, err := uc.resourceChecker.FetchPortReservations(ctx, internalPorts)
		if err != nil {
			logger.LogError("Failed to fetch internal port reservations", "error", err)
		} else {
			allPortReservations = append(allPortReservations, internalPortReservations...)
		}
	}

	portConv := converter.NewPortReservationConverter()
	vehicleConv := converter.NewVehicleReservationConverter()

	reservationIDToRequestID := make(map[string]string)

	var exReservationIDs []string
	for _, v := range allVehicleReservations {
		if v.ReservationID != nil && *v.ReservationID != "" {
			exReservationIDs = append(exReservationIDs, *v.ReservationID)
		}
	}
	for _, p := range allPortReservations {
		if p.ReservationID != nil && *p.ReservationID != "" {
			exReservationIDs = append(exReservationIDs, *p.ReservationID)
		}
	}

	if len(exReservationIDs) > 0 {
		externalReservations, err := uc.externalResourceRepoIF.FindByExReservationIDs(exReservationIDs)
		if err != nil {
			logger.LogError("Failed to find request_id by ex_reservation_ids", "error", err)

		} else {
			for _, exRes := range externalReservations {
				if exRes.ExReservationID != "" {
					reservationIDToRequestID[exRes.ExReservationID] = exRes.RequestID.ToString()
				}
			}
		}
	}

	var mergedItems []*model.AvailabilityItem
	if uc.interconnectReservationSvc != nil {
		mergedItems = uc.interconnectReservationSvc.MergeAvailabilityItemsByRequestID(allAvailabilityItems)
	} else {
		mergedItems = allAvailabilityItems
	}

	resultProto := conv.ToUaslSectionReservations(mergedItems)

	vehicleReservations := vehicleConv.ToProtoVehicleReservations(allVehicleReservations)
	portReservations := portConv.ToProtoPortReservations(allPortReservations)

	for i := range vehicleReservations {
		if vehicleReservations[i].ReservationID != "" {
			if reqID, ok := reservationIDToRequestID[vehicleReservations[i].ReservationID]; ok {
				vehicleReservations[i].RequestID = reqID
			}
		}
	}
	for i := range portReservations {
		if portReservations[i].ReservationID != "" {
			if reqID, ok := reservationIDToRequestID[portReservations[i].ReservationID]; ok {
				portReservations[i].RequestID = reqID
			}
		}
	}

	vehiclePtrs := make([]*model.VehicleAvailabilityItem, len(vehicleReservations))
	for i := range vehicleReservations {
		v := vehicleReservations[i]
		vehiclePtrs[i] = &v
	}
	portPtrs := make([]*model.PortAvailabilityItem, len(portReservations))
	for i := range portReservations {
		p := portReservations[i]
		portPtrs[i] = &p
	}

	logger.LogInfo("GetAvailability completed",
		"uasl_sections_count", len(resultProto),
		"vehicles_count", len(vehicleReservations),
		"ports_count", len(portReservations))

	return &model.GetAvailabilityResponse{
		UaslSections: resultProto,
		Vehicles:     vehiclePtrs,
		Ports:        portPtrs,
	}, nil
}

func (uc *UaslReservationUsecase) TryHoldCompositeUasl(ctx context.Context, req *converter.TryHoldCompositeUaslInput) (*model.ReserveCompositeUaslResponse, error) {

	requestBuilder := converter.NewCompositeReservationRequestBuilder()
	parentDomain, childDomains, err := requestBuilder.ToDomainModels(req.ParentUaslReservation, req.ChildUaslReservations, uc.buildUaslReservationDomain)
	if err != nil {
		return nil, err
	}

	reqData, err := requestBuilder.ToRequestData(req, parentDomain, childDomains)
	if err != nil {
		return nil, err
	}

	compositeValidator := validator.NewCompositeReservationValidator()
	if err := compositeValidator.ValidateCompositeReservations(
		ctx,
		reqData.Vehicles,
		reqData.PortReservations,
	); err != nil {
		return nil, err
	}

	isInterConnect := false
	if req != nil {
		isInterConnect = req.IsInterConnect
	}

	logger.LogInfo("TryHoldCompositeUasl: interconnect flag", "is_interconnect", isInterConnect)

	confirmedInternalPorts := make([]model.PortReservationRequest, 0)
	unknownPorts := make([]model.PortReservationRequest, 0)
	if len(reqData.PortReservations) > 0 {
		allPortIDs := make([]string, 0, len(reqData.PortReservations))
		for _, p := range reqData.PortReservations {
			allPortIDs = append(allPortIDs, p.PortID)
		}
		existingIDs, lookupErr := uc.externalUaslResourceRepoIF.FindExistingExResourceIDs(ctx, allPortIDs)
		if lookupErr != nil {
			logger.LogError("TryHoldCompositeUasl: failed to look up port IDs in external_uasl_resources",
				"error", lookupErr)

			unknownPorts = append(unknownPorts, reqData.PortReservations...)
		} else {
			existingIDSet := make(map[string]bool, len(existingIDs))
			for _, id := range existingIDs {
				existingIDSet[id] = true
			}
			for _, p := range reqData.PortReservations {
				if existingIDSet[p.PortID] {
					confirmedInternalPorts = append(confirmedInternalPorts, p)
				} else {
					unknownPorts = append(unknownPorts, p)
				}
			}
		}
	}

	var adminResolution *services.AvailabilityAdminResolution
	if uc.interconnectReservationSvc != nil {
		uaslIDsForResolution := make([]string, 0, len(childDomains))
		for _, child := range childDomains {
			if child.ExUaslID != nil && *child.ExUaslID != "" {
				uaslIDsForResolution = append(uaslIDsForResolution, *child.ExUaslID)
			}
		}

		var internalUaslIDs []string
		var externalUaslIDs []string
		if len(uaslIDsForResolution) > 0 {
			var classifyErr error
			internalUaslIDs, externalUaslIDs, classifyErr = uc.interconnectReservationSvc.ClassifyInternalExternalUaslIDs(ctx, uaslIDsForResolution)
			if classifyErr != nil {
				logger.LogError("TryHoldCompositeUasl: failed to classify uasl IDs", "error", classifyErr)
				return nil, myerror.Wrap(myerror.Internal, classifyErr, "failed to classify uasl IDs")
			}

			if len(internalUaslIDs) > 0 {

				internalAdmins, err := uc.uaslAdminRepoIF.FindInternalAdministrators(ctx)
				if err != nil {
					logger.LogError("TryHoldCompositeUasl: failed to find internal administrators", "error", err)
				} else if len(internalAdmins) > 0 {

					uaslIDToAdminID := make(map[string]string)
					for _, admin := range internalAdmins {
						servicesList, svcErr := admin.GetExternalServicesList()
						if svcErr != nil || servicesList == nil {
							continue
						}
						for _, svc := range servicesList {
							if svc.ExUaslID != "" {
								uaslIDToAdminID[svc.ExUaslID] = admin.ExAdministratorID
							}
						}
					}

					for i, child := range childDomains {
						if child == nil || child.ExUaslID == nil || *child.ExUaslID == "" {
							continue
						}

						if adminID, ok := uaslIDToAdminID[*child.ExUaslID]; ok {
							if child.ExAdministratorID == nil || *child.ExAdministratorID == "" {
								aid := adminID
								childDomains[i].ExAdministratorID = &aid
								logger.LogInfo("TryHoldCompositeUasl: set internal administrator ID to child",
									"ex_uasl_id", *child.ExUaslID,
									"ex_administrator_id", adminID,
									"is_interconnect", isInterConnect)
							}
						}
					}
				}
			}

			if !isInterConnect && len(externalUaslIDs) > 0 {
				adminResolution = uc.interconnectReservationSvc.ResolveAdministratorsForAvailability(ctx, externalUaslIDs, internalUaslIDs)

				if adminResolution != nil && len(adminResolution.UaslIDToAdministratorID) > 0 {
					for i, child := range childDomains {
						if child == nil {
							continue
						}

						if child.ExUaslID != nil && *child.ExUaslID != "" {
							if child.ExAdministratorID == nil || *child.ExAdministratorID == "" {

								if adminID, ok := adminResolution.UaslIDToAdministratorID[*child.ExUaslID]; ok {
									childDomains[i].ExAdministratorID = &adminID
								}
							}
						}
					}
				}
			}
		}
	}

	sagaTx := uc.newSagaTx(uc.vehicleGateway, uc.portGateway, uc.uaslReservationRepoIF, uc.externalResourceRepoIF, uc.ouranosDiscoveryGW, uc.ouranosProxyGW, uc.externalUaslDefRepoIF, uc.uaslAdminRepoIF, uc.externalUaslResourceRepoIF)

	externalReservationResults := make([]*model.ExternalReservationResult, 0)
	var administratorGroups map[string]*services.AdministratorGroup
	var administratorOrder []string
	var originAdministratorID string
	var lastSectionAdministratorID string

	portsForExternalClassify := unknownPorts
	if isInterConnect {
		portsForExternalClassify = []model.PortReservationRequest{}
	}
	interconnectResult := uc.interconnectReservationSvc.ClassifyInterconnectReservation(
		ctx,
		childDomains,
		portsForExternalClassify,
		adminResolution,
	)

	administratorGroups = interconnectResult.AdministratorGroups
	administratorOrder = interconnectResult.AdministratorOrder

	logger.LogInfo("TryHoldCompositeUasl: classified administrator groups",
		"groups_count", len(administratorGroups),
		"order_count", len(administratorOrder),
		"is_interconnect", isInterConnect)

	if len(administratorGroups) > 0 {
		uc.interconnectReservationSvc.SplitDuplicateSectionTimes(childDomains, administratorGroups)
	}

	var requestedPortIDs map[string]bool
	var processedPortIDs map[string]bool
	if !isInterConnect {
		requestedPortIDs = make(map[string]bool)
		for _, portReq := range unknownPorts {
			requestedPortIDs[fmt.Sprintf("%s_%d", portReq.PortID, portReq.UsageType)] = true
		}

		processedPortIDs = make(map[string]bool)
	}

	if uc.interconnectReservationSvc != nil {
		uc.interconnectReservationSvc.FillParentOriginFromChildren(parentDomain, childDomains)
	}

	resolveLastSectionAdminID := func() string {
		var lastChild *model.UaslReservation
		for _, child := range childDomains {
			if child == nil || child.ExAdministratorID == nil || *child.ExAdministratorID == "" || *child.ExAdministratorID == "unknown" {
				continue
			}
			if lastChild == nil {
				lastChild = child
				continue
			}
			if child.Sequence != nil && lastChild.Sequence != nil {
				if *child.Sequence > *lastChild.Sequence {
					lastChild = child
				}
				continue
			}
			if child.Sequence != nil && lastChild.Sequence == nil {
				lastChild = child
				continue
			}
			if child.Sequence == nil && lastChild.Sequence != nil {
				continue
			}
			if child.EndAt.After(lastChild.EndAt) {
				lastChild = child
			}
		}
		if lastChild != nil && lastChild.ExAdministratorID != nil && *lastChild.ExAdministratorID != "" {
			return *lastChild.ExAdministratorID
		}
		return ""
	}
	lastSectionAdministratorID = resolveLastSectionAdminID()

	if parentDomain != nil && parentDomain.ExAdministratorID != nil && *parentDomain.ExAdministratorID != "" && *parentDomain.ExAdministratorID != "unknown" {
		originAdministratorID = *parentDomain.ExAdministratorID
	} else if len(administratorOrder) > 0 {
		originAdministratorID = administratorOrder[0]
	}

	internalResourceAdministratorID := originAdministratorID
	if isInterConnect && lastSectionAdministratorID != "" && lastSectionAdministratorID != "unknown" {
		internalResourceAdministratorID = lastSectionAdministratorID
	}

	for _, adminID := range administratorOrder {
		adminGroup := administratorGroups[adminID]
		if adminGroup == nil || adminGroup.IsInternal {
			continue
		}

		var vehicleDetail *model.VehicleDetailInfo
		if req != nil {

			if req.OperatingAircraft != nil {

				vehicleDetail = req.OperatingAircraft
			} else if len(req.Vehicles) > 0 {
				v := req.Vehicles[0]
				if v.AircraftInfo != nil {
					vehicleDetail = v.AircraftInfo
				}
			}
		}

		reservationsMap, reservationData, err := uc.interconnectReservationSvc.ReserveExternalUasl(
			ctx,
			*adminGroup,
			vehicleDetail,
			adminGroup.OriginAdministratorID,
			adminGroup.OriginUaslID,
			sagaTx.SagaOrchestrator,
			req.IgnoreFlightPlanConflict,
		)

		if err != nil {
			logger.LogError("External reservation failed (executing compensation)",
				"administrator_id", adminGroup.AdministratorID,
				"error", err,
			)
			sagaTx.SagaOrchestrator.Rollback(ctx)

			return nil, fmt.Errorf("external reservation for administrator %s failed: %w", adminGroup.AdministratorID, err)
		}

		now := time.Now()

		type uaslPriceInfo struct {
			Amount             int
			PricingRuleVersion int
		}
		uaslPriceMap := make(map[string]uaslPriceInfo)
		if reservationData != nil {
			for _, destRes := range reservationData.DestinationReservations {
				for _, uasl := range destRes.UaslSections {
					uaslPriceMap[uasl.UaslSectionID] = uaslPriceInfo{
						Amount:             uasl.Amount,
						PricingRuleVersion: reservationData.PricingRuleVersion,
					}
				}
			}
		}

		for _, child := range adminGroup.ChildDomains {
			if child.ExUaslSectionID == nil || *child.ExUaslSectionID == "" {
				continue
			}

			if priceInfo, ok := uaslPriceMap[*child.ExUaslSectionID]; ok {
				child.Amount = &priceInfo.Amount
				child.PricingRuleVersion = &priceInfo.PricingRuleVersion
			}

			child.EstimatedAt = &now
		}

		res := &model.ExternalReservationResult{
			AdministratorID: adminGroup.AdministratorID,
			URL:             adminGroup.URL,
			Reservations:    reservationsMap,
			ChildDomains:    adminGroup.ChildDomains,
			PortRequests:    adminGroup.PortRequests,
			ReservationData: reservationData,
			Error:           err,
		}

		if !isInterConnect && processedPortIDs != nil {
			portsRecorded := false
			if reservationData != nil {
				for _, destRes := range reservationData.DestinationReservations {
					for _, port := range destRes.Ports {
						processedPortIDs[fmt.Sprintf("%s_%d", port.PortID, port.UsageType)] = true
						portsRecorded = true
					}
				}
			}
			if !portsRecorded {
				for _, portReq := range adminGroup.PortRequests {
					processedPortIDs[fmt.Sprintf("%s_%d", portReq.PortID, portReq.UsageType)] = true
				}
				if len(adminGroup.PortRequests) > 0 {
					logger.LogInfo("TryHoldCompositeUasl: used PortRequests as fallback for processedPortIDs",
						"administrator_id", adminGroup.AdministratorID,
						"port_count", len(adminGroup.PortRequests))
				}
			}
		}

		externalReservationResults = append(externalReservationResults, res)
	}

	internalPorts := confirmedInternalPorts

	logger.LogInfo("Port classification result",
		"total_ports", len(reqData.PortReservations),
		"confirmed_internal_ports", len(internalPorts),
		"unknown_ports_sent_to_external", len(unknownPorts))

	resourceNameMap, err := uc.resourceChecker.ValidateCompositeSynchronization(
		ctx,
		reqData.Vehicles,
		internalPorts,
	)
	if err != nil {
		sagaTx.SagaOrchestrator.Rollback(ctx)
		return nil, fmt.Errorf("pre-check failed: %w", err)
	}

	internalUaslRequests := make([]model.UaslCheckRequest, 0)
	if len(administratorGroups) == 0 {
		for _, child := range childDomains {
			if child.ExUaslSectionID == nil || *child.ExUaslSectionID == "" {
				continue
			}
			internalUaslRequests = append(internalUaslRequests, model.UaslCheckRequest{
				ExUaslSectionID: *child.ExUaslSectionID,
				TimeRange: model.TimeRange{
					Start: child.StartAt,
					End:   child.EndAt,
				},
			})
		}
	} else {
		for _, child := range childDomains {
			if child.ExUaslSectionID == nil || *child.ExUaslSectionID == "" {
				continue
			}

			if child.ExAdministratorID == nil || *child.ExAdministratorID == "" {
				continue
			}
			adminID := *child.ExAdministratorID
			if group, exists := administratorGroups[adminID]; exists && group.IsInternal {
				internalUaslRequests = append(internalUaslRequests, model.UaslCheckRequest{
					ExUaslSectionID: *child.ExUaslSectionID,
					TimeRange: model.TimeRange{
						Start: child.StartAt,
						End:   child.EndAt,
					},
				})
			}
		}
	}

	uaslConflict, err := uc.airspaceConflictChecker.CheckUaslReservationConflict(ctx, internalUaslRequests)
	if err != nil {
		sagaTx.SagaOrchestrator.Rollback(ctx)
		return nil, fmt.Errorf("uasl reservation conflict check failed: %w", err)
	}
	if uaslConflict != nil {
		sagaTx.SagaOrchestrator.Rollback(ctx)
		return &model.ReserveCompositeUaslResponse{
			ConflictType:          uaslConflict.ConflictType,
			ConflictedResourceIds: uaslConflict.ConflictedIDs,
			ParentUaslReservation: nil,
		}, nil
	}

	conflictResult, shouldProceed, err := uc.airspaceConflictChecker.CheckAirspaceConflictWithPolicy(
		ctx,
		internalUaslRequests,
		req.IgnoreFlightPlanConflict,
	)
	if err != nil {
		sagaTx.SagaOrchestrator.Rollback(ctx)
		return nil, err
	}

	if !shouldProceed {
		sagaTx.SagaOrchestrator.Rollback(ctx)
		conflicted := converter.ToConflictedFlightPlanIDs(conflictResult)
		return &model.ReserveCompositeUaslResponse{
			ConflictedFlightPlanIDs: conflicted,
			ConflictType:            "FLIGHT_PLAN",
			ConflictedResourceIds:   conflicted,
			ParentUaslReservation:   nil,
			ChildUaslReservations:   nil,
			Vehicles:                nil,
			Ports:                   nil,
		}, nil
	}

	resourceConflict, err := uc.resourceChecker.CheckCompositeResourcesAvailability(
		ctx,
		reqData.Vehicles,
		internalPorts,
	)
	if err != nil {
		logger.LogInfo("Failed to check composite resource availability, attempting compensation (Rollback)")
		sagaTx.SagaOrchestrator.Rollback(ctx)
		return nil, fmt.Errorf("composite availability check failed: %w", err)
	}
	if resourceConflict != nil {
		sagaTx.SagaOrchestrator.Rollback(ctx)
		return &model.ReserveCompositeUaslResponse{
			ConflictType:          resourceConflict.ConflictType,
			ConflictedResourceIds: resourceConflict.ConflictedIDs,
			ParentUaslReservation: nil,
		}, nil
	}

	if !isInterConnect && processedPortIDs != nil {

		for _, portReq := range internalPorts {
			processedPortIDs[fmt.Sprintf("%s_%d", portReq.PortID, portReq.UsageType)] = true
		}

		var unresolvedPortIDs []string
		for portKey := range requestedPortIDs {
			if !processedPortIDs[portKey] {
				unresolvedPortIDs = append(unresolvedPortIDs, portKey)
			}
		}

		if len(unresolvedPortIDs) > 0 {
			logger.LogError("Ports not found in any uasl system, initiating rollback",
				"unresolved_port_ids", unresolvedPortIDs)

			if rollbackErr := sagaTx.SagaOrchestrator.Rollback(ctx); rollbackErr != nil {
				logger.LogError("Failed to rollback after port validation failure",
					"error", rollbackErr,
					"unresolved_port_ids", unresolvedPortIDs)
			}

			return nil, fmt.Errorf("ports not found in any uasl system: %v", unresolvedPortIDs)
		}
	}

	gswPrices, err := uc.billingService.CalculateResourcePrices(ctx, reqData.Vehicles, internalPorts)
	if err != nil {
		sagaTx.SagaOrchestrator.Rollback(ctx)
		return nil, fmt.Errorf("fetch GSW prices failed: %w", err)
	}

	_, totalUaslAmount, err := uc.billingService.CalculateInternalSectionPrices(
		ctx,
		administratorGroups,
	)
	if err != nil {
		sagaTx.SagaOrchestrator.Rollback(ctx)
		return nil, fmt.Errorf("calculate internal section prices failed: %w", err)
	}

	if parentDomain != nil {
		for _, group := range administratorGroups {
			if group != nil && group.IsInternal && len(group.ChildDomains) > 0 {
				firstChild := group.ChildDomains[0]
				if firstChild.PricingRuleVersion != nil {
					parentDomain.PricingRuleVersion = firstChild.PricingRuleVersion
					logger.LogInfo("Set pricing rule version to parent domain from internal child",
						"pricing_rule_version", *firstChild.PricingRuleVersion)
					break
				}
			}
		}
	}

	baseAmount := totalUaslAmount + gswPrices.TotalAmount

	var externalReservationAmount int32 = 0
	if !isInterConnect {
		for _, extResult := range externalReservationResults {

			for _, child := range extResult.ChildDomains {
				if child.Amount != nil {
					externalReservationAmount += util.SafeIntToInt32(*child.Amount)
				}
			}
		}
	}

	totalAmount := baseAmount + externalReservationAmount

	logger.LogInfo("Calculated total reservation amount",
		"internal_uasl_amount", totalUaslAmount,
		"gsw_resource_amount", gswPrices.TotalAmount,
		"external_uasl_amount", externalReservationAmount,
		"total_amount", totalAmount,
		"is_interconnect", isInterConnect)

	totalAmountInt := int(totalAmount)
	parentDomain.Amount = &totalAmountInt
	now := time.Now()
	parentDomain.EstimatedAt = &now

	var vehicleElements []*model.VehicleElement
	var portElements []*model.PortElement
	var externalResourceMappings []*model.ExternalResourceReservation

	if !isInterConnect {
		allChildren := make([]*model.UaslReservation, 0)
		for _, adminID := range administratorOrder {
			group := administratorGroups[adminID]
			if group != nil {
				allChildren = append(allChildren, group.ChildDomains...)
			}
		}
		childDomains = allChildren
	}

	requestID := value.ModelID("")
	if parentDomain != nil {
		requestID = parentDomain.RequestID
	}
	if requestID == "" && len(childDomains) > 0 {
		requestID = childDomains[0].RequestID
	}

	externalResourceConv := converter.NewExternalResourceConverter()
	var externalResourceAmount int32
	vehicleNameMap := map[string]string{}
	portNameMap := map[string]string{}
	if resourceNameMap != nil {
		if resourceNameMap.VehicleNames != nil {
			vehicleNameMap = resourceNameMap.VehicleNames
		}
		if resourceNameMap.PortNames != nil {
			portNameMap = resourceNameMap.PortNames
		}
	}
	vehicleElements, portElements, externalResourceMappings, externalResourceAmount = externalResourceConv.ToExternalResourceElements(
		gswPrices,
		externalReservationResults,
		reqData.Vehicles,
		internalPorts,
		parentDomain,
		requestID,
		internalResourceAdministratorID,
		vehicleNameMap,
		portNameMap,
	)

	if !isInterConnect && externalResourceAmount > 0 {
		totalAmountWithExtResources := int(totalAmount) + int(externalResourceAmount)
		parentDomain.Amount = &totalAmountWithExtResources
	}

	var aircraftForAssessment model.VehicleDetailInfo
	if req != nil {
		if req.OperatingAircraft != nil {

			aircraftForAssessment = *req.OperatingAircraft
			logger.LogInfo("Using operatingAircraft for conformity assessment",
				"maker", aircraftForAssessment.Maker,
				"model_number", aircraftForAssessment.ModelNumber)
		} else if len(req.Vehicles) > 0 && req.Vehicles[0].AircraftInfo != nil {
			aircraftForAssessment = *req.Vehicles[0].AircraftInfo
			logger.LogInfo("Using vehicles[0] for conformity assessment",
				"maker", aircraftForAssessment.Maker,
				"model_number", aircraftForAssessment.ModelNumber)
		}
	}

	internalChildDomains := make([]*model.UaslReservation, 0)
	if len(administratorGroups) == 0 {

		internalChildDomains = childDomains
	} else {

		for _, child := range childDomains {
			if child.ExAdministratorID != nil && *child.ExAdministratorID != "" {
				adminID := *child.ExAdministratorID
				if group, exists := administratorGroups[adminID]; exists && group.IsInternal {
					internalChildDomains = append(internalChildDomains, child)
				}
			}
		}
	}

	assessments, err := uc.airspaceConflictChecker.CallConformityAssessmentForChildren(ctx, internalChildDomains, aircraftForAssessment)
	if err != nil {
		logger.LogError("Failed to call conformity assessment", "error", err)

		if parentDomain != nil && (aircraftForAssessment.Maker != "" || aircraftForAssessment.ModelNumber != "") {
			parentDomain.ConformityAssessment = model.ConformityAssessmentList{
				{
					AircraftInfo: aircraftForAssessment,
				},
			}
		}
	}

	if parentDomain != nil && len(assessments) > 0 {
		parentDomain.ConformityAssessment = assessments
	}

	if len(internalChildDomains) > 0 && parentDomain != nil {
		converter.ApplyConformityToChildren(parentDomain, internalChildDomains)
	}

	internalGroups := uc.interconnectReservationSvc.DetectInternalUaslGroups(childDomains, administratorGroups)

	includeInternalParents := len(internalGroups) > 0 && (isInterConnect || len(internalGroups) > 1)

	if includeInternalParents {

		uc.interconnectReservationSvc.BuildParentReservationsForGroups(internalGroups, parentDomain, isInterConnect)

		uc.interconnectReservationSvc.SetChildReservationParentsForGroups(internalGroups)

		if !isInterConnect {
			var primaryGroup *services.InternalUaslGroup
			var minSeq *int
			for _, g := range internalGroups {
				if g == nil || len(g.Children) == 0 {
					continue
				}
				for _, child := range g.Children {
					if child == nil || child.Sequence == nil {
						continue
					}
					if minSeq == nil || *child.Sequence < *minSeq {
						v := *child.Sequence
						minSeq = &v
						primaryGroup = g
					}
				}
			}
			if primaryGroup == nil && len(internalGroups) > 0 {
				primaryGroup = internalGroups[0]
			}

			if primaryGroup != nil && primaryGroup.ParentReservation != nil {
				primaryGroup.ParentReservation.Amount = &totalAmountInt
				primaryGroup.ParentReservation.EstimatedAt = &now
			}
		}

		if isInterConnect && len(internalGroups) > 0 && internalGroups[0].ParentReservation != nil {
			parentDomain = internalGroups[0].ParentReservation
		}
	} else {

		for _, child := range childDomains {
			if child == nil {
				continue
			}
			if child.ParentUaslReservationID == nil || child.ParentUaslReservationID.ToString() == "" {
				child.ParentUaslReservationID = &parentDomain.ID
			}
			if child.Status != value.RESERVATION_STATUS_INHERITED {
				child.Status = value.RESERVATION_STATUS_INHERITED
			}
		}
	}

	externalGroups := uc.interconnectReservationSvc.DetectExternalUaslGroups(
		childDomains,
		administratorGroups,
		externalReservationResults,
	)
	if len(externalGroups) > 0 {
		uc.interconnectReservationSvc.BuildParentReservationsForExternalGroups(externalGroups, parentDomain)
		uc.interconnectReservationSvc.SetChildReservationParentsForExternalGroups(externalGroups)

		logger.LogInfo("TryHoldCompositeUasl: external groups parent reservations created",
			"external_groups_count", len(externalGroups))
	} else {

		uc.interconnectReservationSvc.SetExternalChildReservationParents(externalReservationResults)
	}

	if len(externalReservationResults) > 0 {
		converter.ToExternalConformity(childDomains, externalReservationResults)
	}

	uaslConv := converter.NewUaslReservationDomain()

	convGroups := make([]converter.InternalUaslGroup, len(internalGroups))
	for i, g := range internalGroups {
		convGroups[i] = converter.InternalUaslGroup{
			ParentReservation: g.ParentReservation,
			Children:          g.Children,
		}
	}

	convExternalGroups := make([]converter.ExternalUaslGroup, len(externalGroups))
	for i, g := range externalGroups {
		convExternalGroups[i] = converter.ExternalUaslGroup{
			AdministratorID:       g.AdministratorID,
			ExUaslID:              g.ExUaslID,
			ExternalReservationID: g.ExternalReservationID,
			ParentReservation:     g.ParentReservation,
			Children:              g.Children,
		}
	}

	destReservations := uaslConv.ToDestinationReservationsFromGroups(
		isInterConnect,
		externalReservationResults,
		convExternalGroups,
		convGroups,
		childDomains,
	)

	if !isInterConnect && len(destReservations) > 0 {
		var primaryGroup *services.InternalUaslGroup
		var minSeq *int
		for _, g := range internalGroups {
			if g == nil || len(g.Children) == 0 {
				continue
			}
			for _, child := range g.Children {
				if child == nil || child.Sequence == nil {
					continue
				}
				if minSeq == nil || *child.Sequence < *minSeq {
					v := *child.Sequence
					minSeq = &v
					primaryGroup = g
				}
			}
		}
		if primaryGroup == nil && len(internalGroups) > 0 {
			primaryGroup = internalGroups[0]
		}
		if primaryGroup != nil && primaryGroup.ParentReservation != nil {
			primaryGroup.ParentReservation.DestinationReservations = destReservations
			parentDomain = primaryGroup.ParentReservation
		}

		logger.LogInfo("DestinationReservations set to sequence-1 parent",
			"parent_id", func() string {
				if primaryGroup != nil && primaryGroup.ParentReservation != nil {
					return primaryGroup.ParentReservation.ID.ToString()
				}
				return ""
			}(),
			"destination_count", len(destReservations))
	}

	if !isInterConnect && parentDomain != nil && len(internalGroups) > 0 {
		services.MergeConformityFromInternalGroups(parentDomain, internalGroups)
	}

	if parentDomain != nil && len(externalReservationResults) > 0 {
		existing := make(map[string]struct{})
		for _, ca := range parentDomain.ConformityAssessment {
			key := ca.UaslSectionID + "|" + ca.Type + "|" + ca.StartAt.Format(time.RFC3339) + "|" + ca.EndAt.Format(time.RFC3339)
			existing[key] = struct{}{}
		}

		for _, extResult := range externalReservationResults {
			if extResult == nil || extResult.ReservationData == nil {
				continue
			}
			for _, destRes := range extResult.ReservationData.DestinationReservations {
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

					parentDomain.ConformityAssessment = append(parentDomain.ConformityAssessment, model.ConformityAssessmentItem{
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
	}

	if len(externalGroups) > 0 && parentDomain != nil {
		services.ApplyConformityToExternalGroups(parentDomain, externalGroups)
	}

	allReservations := uaslConv.ToReservationsForBatchInsert(
		convGroups,
		convExternalGroups,
		externalReservationResults,
		parentDomain,
		includeInternalParents,
	)

	uaslBatch := &model.UaslReservationBatch{
		Parent:   parentDomain,
		Children: allReservations,
	}

	parentDomain, childDomains, err = sagaTx.ReserveUasl(ctx, uaslBatch)
	if err != nil {
		logger.LogError("CRITICAL: uasl reservation failed",
			"error", err,
			"parent_id", func() string {
				if parentDomain != nil {
					return parentDomain.ID.ToString()
				}
				return "not_yet_assigned"
			}(),
			"children_count", len(childDomains),
			"severity", "CRITICAL",
			"requires_manual_intervention", false,
		)

		if len(externalReservationResults) > 0 {
			sagaTx.SagaOrchestrator.Rollback(ctx)
		}

		return nil, fmt.Errorf("uasl reservation failed: %w", err)
	}

	if len(externalResourceMappings) > 0 {
		if uc.externalResourceRepoIF != nil {
			if _, err := uc.externalResourceRepoIF.InsertBatch(externalResourceMappings); err != nil {
				logger.LogError("CRITICAL: failed to insert external resource mappings, attempting compensation",
					"error", err,
					"parent_reservation_id", parentDomain.ID.ToString(),
					"vehicle_count", len(reqData.Vehicles),
					"port_count", len(reqData.PortReservations),
					"severity", "CRITICAL",
					"requires_manual_intervention", true,
				)

				sagaTx.SagaOrchestrator.Rollback(ctx)

				return nil, fmt.Errorf(
					"external resource mappings insert failed - external reservations may require manual cleanup: %w",
					err,
				)
			}
		}
	}

	if !isInterConnect {
		for _, child := range childDomains {
			if child.Sequence != nil && *child.Sequence == 1 && child.ExUaslSectionID != nil && *child.ExUaslSectionID != "" {
				if uc.externalUaslDefRepoIF != nil {
					if def, err := uc.externalUaslDefRepoIF.FindByExSectionID(ctx, *child.ExUaslSectionID); err != nil {
						logger.LogError("TryHoldCompositeUasl: failed to fetch flight purpose (non-fatal)",
							"error", err, "ex_uasl_section_id", *child.ExUaslSectionID)
					} else if def != nil && def.FlightPurpose.Valid && def.FlightPurpose.String != nil {
						parentDomain.FlightPurpose = *def.FlightPurpose.String
					}
				}
				break
			}
		}
	}

	{
		domainConv := converter.NewUaslReservationDomain()
		response := domainConv.ToCompositeUaslReservationResponse(
			parentDomain,
			childDomains,
			conflictResult,
			vehicleElements,
			portElements,
		)

		return response, nil
	}
}

func (uc *UaslReservationUsecase) SearchByCondition(
	ctx context.Context,
	req *converter.SearchByConditionInput,
) (*model.SearchByConditionResponse, error) {
	logger.LogInfo("SearchByCondition started",
		"requestIds_count", len(req.RequestIDs))

	if len(req.RequestIDs) == 0 {
		return &model.SearchByConditionResponse{
			Result: []*model.UaslReservationListItemMsg{},
		}, nil
	}

	requestIDSet := make(map[string]struct{}, len(req.RequestIDs))
	for _, rid := range req.RequestIDs {
		requestIDSet[rid] = struct{}{}
	}

	requestIDs := make([]value.ModelID, 0, len(requestIDSet))
	for rid := range requestIDSet {
		id, err := value.NewModelIDFromUUIDString(rid)
		if err != nil {
			logger.LogError("Invalid request ID format",
				"request_id", rid,
				"error", err)
			continue
		}
		requestIDs = append(requestIDs, id)
	}

	if len(requestIDs) == 0 {
		return &model.SearchByConditionResponse{
			Result: []*model.UaslReservationListItemMsg{},
		}, nil
	}

	items, err := uc.uaslReservationRepoIF.ListByRequestIDs(requestIDs)
	if err != nil {
		return nil, fmt.Errorf("ListByRequestIDs failed: %w", err)
	}

	domainConv := converter.NewUaslReservationDomain()
	var result []*model.UaslReservationListItemMsg
	for _, item := range items {
		converted, err := domainConv.ToUaslReservationListItem(item)
		if err != nil {
			logger.LogError("Failed to convert item to proto",
				"error", err)
			continue
		}
		if converted != nil {
			result = append(result, converted)
		}
	}

	logger.LogInfo("SearchByCondition completed",
		"result_count", len(result))

	return &model.SearchByConditionResponse{
		Result: result,
	}, nil
}

func (uc *UaslReservationUsecase) EstimateUaslReservation(ctx context.Context, req *converter.EstimateUaslReservationInput) (*model.EstimateUaslReservationResponse, error) {
	logger.LogInfo("EstimateUaslReservation started")

	estimateValidator := validator.NewUaslReservation()
	sectionIDs, vehicleIDs, portIDs, err := estimateValidator.EstimateRequest(ctx, req)
	if err != nil {
		return nil, myerror.Wrap(myerror.BadParams, err, err.Error())
	}

	ownSections, err := uc.externalUaslDefRepoIF.FindByExSectionIDs(ctx, sectionIDs)
	if err != nil {
		return nil, myerror.Wrap(myerror.Database, err, "failed to find external uasl definitions")
	}

	ownSectionMap := make(map[string]*model.ExternalUaslDefinition)
	for _, def := range ownSections {
		ownSectionMap[def.ExUaslSectionID] = def
	}

	logger.LogInfo("Own sections loaded",
		"requested_count", len(sectionIDs),
		"found_count", len(ownSections))

	ownResources := make([]*model.ExternalUaslResource, 0)
	if len(vehicleIDs) > 0 {
		vehicles, err := uc.externalUaslResourceRepoIF.FindByResourceIDsAndType(vehicleIDs, model.ExternalResourceTypeVehicle)
		if err != nil {
			return nil, myerror.Wrap(myerror.Database, err, "failed to find vehicle resources")
		}
		ownResources = append(ownResources, vehicles...)
	}
	if len(portIDs) > 0 {
		ports, err := uc.externalUaslResourceRepoIF.FindByResourceIDsAndType(portIDs, model.ExternalResourceTypePort)
		if err != nil {
			return nil, myerror.Wrap(myerror.Database, err, "failed to find port resources")
		}
		ownResources = append(ownResources, ports...)
	}

	ownResourceMap := make(map[string]*model.ExternalUaslResource)
	for _, res := range ownResources {
		ownResourceMap[res.ExResourceID] = res
	}

	logger.LogInfo("Own resources loaded",
		"vehicle_requested", len(vehicleIDs),
		"port_requested", len(portIDs),
		"found_count", len(ownResources))

	isInterConnect := req.IsInterConnect
	var externalAmount int32 = 0

	externalSectionIDs := make([]string, 0)
	for _, sID := range sectionIDs {
		if _, found := ownSectionMap[sID]; !found {
			externalSectionIDs = append(externalSectionIDs, sID)
		}
	}

	logger.LogInfo("External sections identified",
		"external_section_count", len(externalSectionIDs),
		"is_interconnect", isInterConnect)

	externalVehicleIDs := make([]string, 0)
	externalPortIDs := make([]string, 0)
	for _, vID := range vehicleIDs {
		if _, found := ownResourceMap[vID]; !found {
			externalVehicleIDs = append(externalVehicleIDs, vID)
		}
	}
	for _, pID := range portIDs {
		if _, found := ownResourceMap[pID]; !found {
			externalPortIDs = append(externalPortIDs, pID)
		}
	}

	hasExternalResources := len(externalVehicleIDs) > 0 || len(externalPortIDs) > 0
	hasExternalSections := len(externalSectionIDs) > 0

	logger.LogInfo("External resources identified",
		"external_vehicle_count", len(externalVehicleIDs),
		"external_port_count", len(externalPortIDs))

	if hasExternalSections || hasExternalResources {
		logger.LogInfo("External API call required",
			"external_sections", len(externalSectionIDs),
			"external_vehicles", len(externalVehicleIDs),
			"external_ports", len(externalPortIDs),
			"is_interconnect", isInterConnect)

		conv := converter.NewUaslReservationDomain()
		externalReq, err := conv.ToExternalEstimateRequest(req, externalSectionIDs, externalVehicleIDs, externalPortIDs)
		if err != nil {
			logger.LogError("Failed to build external estimate request",
				"error", err)
			return nil, myerror.Wrap(myerror.BadParams, err, "failed to build external estimate request")
		}

		uaslIDsForEstimate := make([]string, 0)
		for _, sec := range externalReq.UaslSections {
			if sec.UaslID != "" {
				uaslIDsForEstimate = append(uaslIDsForEstimate, sec.UaslID)
			}
		}

		internalUaslIDs, externalUaslIDs, err := uc.interconnectReservationSvc.ClassifyInternalExternalUaslIDs(ctx, uaslIDsForEstimate)
		if err != nil {
			logger.LogError("Estimate: failed to classify uasl IDs", "error", err)
			return nil, myerror.Wrap(myerror.Internal, err, "failed to classify uasl IDs")
		}

		if len(externalUaslIDs) > 0 {
			estimateAdminResolution := uc.interconnectReservationSvc.ResolveAdministratorsForAvailability(ctx, externalUaslIDs, internalUaslIDs)
			externalAmount, err = uc.interconnectReservationSvc.EstimateExternalUasl(ctx, externalReq, *estimateAdminResolution)
		}
		if err != nil {
			logger.LogError("Failed to estimate external uasl",
				"error", err)
			return nil, myerror.Wrap(myerror.Connection, err, "failed to estimate external uasl")
		}
	}

	var ownSectionAmount int32 = 0

	estimateAdminGroups := make(map[string]*services.AdministratorGroup)
	ownSectionCount := 0
	for _, s := range req.UaslSections {
		def, isOwn := ownSectionMap[s.ExUaslSectionID]
		if !isOwn || def.ExAdministratorID == "" {
			continue
		}

		startAt, err := time.Parse(time.RFC3339, s.StartAt)
		if err != nil {
			return nil, myerror.Wrap(myerror.BadParams, err, fmt.Sprintf("invalid startAt for section %s", s.ExUaslSectionID))
		}
		endAt, err := time.Parse(time.RFC3339, s.EndAt)
		if err != nil {
			return nil, myerror.Wrap(myerror.BadParams, err, fmt.Sprintf("invalid endAt for section %s", s.ExUaslSectionID))
		}

		sectionID := s.ExUaslSectionID
		adminID := def.ExAdministratorID
		child := &model.UaslReservation{
			ExUaslSectionID:   &sectionID,
			ExAdministratorID: &adminID,
			StartAt:           startAt,
			EndAt:             endAt,
		}

		if _, exists := estimateAdminGroups[adminID]; !exists {
			estimateAdminGroups[adminID] = &services.AdministratorGroup{
				AdministratorID: adminID,
				IsInternal:      true,
			}
		}
		estimateAdminGroups[adminID].ChildDomains = append(estimateAdminGroups[adminID].ChildDomains, child)
		ownSectionCount++
	}

	if ownSectionCount > 0 {
		_, ownSectionAmount, err = uc.billingService.CalculateInternalSectionPrices(ctx, estimateAdminGroups)
		if err != nil {
			logger.LogError("Failed to calculate own section prices", "error", err)
			return nil, myerror.Wrap(myerror.Internal, err, "failed to calculate own section prices")
		}

		logger.LogInfo("Own sections total calculated (time-segmented, merged)",
			"total_sections", ownSectionCount,
			"total_amount", ownSectionAmount)
	}

	var ownResourceAmount int32 = 0

	resourceTimeMap := make(map[string]struct {
		startAt time.Time
		endAt   time.Time
	})

	for _, v := range req.Vehicles {
		if v.StartAt != "" && v.EndAt != "" {
			startAt, err1 := time.Parse(time.RFC3339, v.StartAt)
			endAt, err2 := time.Parse(time.RFC3339, v.EndAt)
			if err1 == nil && err2 == nil {
				resourceTimeMap[v.VehicleID] = struct {
					startAt time.Time
					endAt   time.Time
				}{startAt, endAt}
			}
		}
	}

	for _, p := range req.Ports {
		if p.StartAt != "" && p.EndAt != "" {
			startAt, err1 := time.Parse(time.RFC3339, p.StartAt)
			endAt, err2 := time.Parse(time.RFC3339, p.EndAt)
			if err1 == nil && err2 == nil {
				resourceTimeMap[p.PortID] = struct {
					startAt time.Time
					endAt   time.Time
				}{startAt, endAt}
			}
		}
	}

	for _, res := range ownResources {

		resourceTime, ok := resourceTimeMap[res.ExResourceID]
		if !ok {
			logger.LogError("Time not specified for resource in request",
				"resource_id", res.ExResourceID)
			continue
		}

		priceInfos := converter.ToResourcePriceInfoList(res.EstimatedPricePerMinute)
		resourcePrice := services.CalculateResourcePriceFromRules(priceInfos, resourceTime.startAt, resourceTime.endAt)
		ownResourceAmount += resourcePrice

		logger.LogInfo("Own resource price calculated",
			"resource_id", res.ExResourceID,
			"start_at", resourceTime.startAt,
			"end_at", resourceTime.endAt,
			"total_price", resourcePrice)
	}

	logger.LogInfo("Own resources total calculated",
		"total_amount", ownResourceAmount)

	totalAmount := ownSectionAmount + ownResourceAmount + externalAmount
	now := time.Now().UTC().Format(time.RFC3339)

	logger.LogInfo("EstimateUaslReservation completed",
		"own_section_amount", ownSectionAmount,
		"own_resource_amount", ownResourceAmount,
		"external_amount", externalAmount,
		"total_amount", totalAmount)

	return &model.EstimateUaslReservationResponse{
		TotalAmount: totalAmount,
		EstimatedAt: now,
	}, nil
}

func (uc *UaslReservationUsecase) NotifyReservationCompleted(
	ctx context.Context,
	req *model.NotifyReservationCompletedRequest,
) (*model.NotifyReservationCompletedResponse, error) {
	if req == nil || len(req.Notifications) == 0 {
		return &model.NotifyReservationCompletedResponse{
			Sent:   []*model.NotificationResult{},
			Failed: []*model.NotificationResult{},
		}, nil
	}
	if uc.interconnectReservationSvc == nil || uc.ouranosProxyGW == nil {
		return nil, fmt.Errorf("notify reservation completed: required services are not configured")
	}

	uaslIDSet := make(map[string]struct{})
	for _, n := range req.Notifications {
		if n == nil {
			continue
		}
		if n.UaslId != "" {
			uaslIDSet[n.UaslId] = struct{}{}
		}
	}
	uaslIDs := make([]string, 0, len(uaslIDSet))
	for id := range uaslIDSet {
		uaslIDs = append(uaslIDs, id)
	}

	internalUaslIDs := []string{}
	if uc.uaslAdminRepoIF != nil {
		internalAdmins, err := uc.uaslAdminRepoIF.FindInternalAdministrators(ctx)
		if err != nil {
			logger.LogError("NotifyReservationCompleted: failed to find internal administrators", "error", err)
		} else {
			for _, admin := range internalAdmins {
				servicesList, svcErr := admin.GetExternalServicesList()
				if svcErr != nil || servicesList == nil {
					continue
				}
				for _, svc := range servicesList {
					if svc.ExUaslID != "" {
						internalUaslIDs = append(internalUaslIDs, svc.ExUaslID)
					}
				}
			}
		}
	}

	adminResolution := uc.interconnectReservationSvc.ResolveAdministratorsForAvailability(ctx, uaslIDs, internalUaslIDs)

	sent := make([]*model.NotificationResult, 0)
	failed := make([]*model.NotificationResult, 0)

	for _, n := range req.Notifications {
		if n == nil {
			continue
		}
		if n.UaslId == "" {
			failed = append(failed, &model.NotificationResult{
				ReservationID:   n.ReservationId,
				UaslID:          n.UaslId,
				AdministratorID: n.AdministratorId,
			})
			logger.LogError("NotifyReservationCompleted: missing uaslId",
				"request_id", n.RequestId,
				"reservation_id", n.ReservationId)
			continue
		}

		baseURL := ""
		if adminResolution != nil {
			if endpoints, ok := adminResolution.ExternalServices[n.UaslId]; ok {
				baseURL = endpoints.BaseURL
			}
		}
		if baseURL == "" {
			failed = append(failed, &model.NotificationResult{
				ReservationID:   n.ReservationId,
				UaslID:          n.UaslId,
				AdministratorID: n.AdministratorId,
			})
			logger.LogError("NotifyReservationCompleted: missing base URL",
				"request_id", n.RequestId,
				"reservation_id", n.ReservationId,
				"uasl_id", n.UaslId)
			continue
		}

		sendErr := retry.WithBackoff(ctx, func(retryCtx context.Context) error {
			return uc.ouranosProxyGW.SendReservationCompleted(retryCtx, baseURL, *n)
		}, retry.DefaultConfig())

		if sendErr != nil {
			failed = append(failed, &model.NotificationResult{
				ReservationID:   n.ReservationId,
				UaslID:          n.UaslId,
				AdministratorID: n.AdministratorId,
			})
			logger.LogError("NotifyReservationCompleted: failed to send webhook after retries",
				"request_id", n.RequestId,
				"reservation_id", n.ReservationId,
				"uasl_id", n.UaslId,
				"url", baseURL,
				"error", sendErr.Error())
			continue
		}
		sent = append(sent, &model.NotificationResult{
			ReservationID:   n.ReservationId,
			UaslID:          n.UaslId,
			AdministratorID: n.AdministratorId,
		})
	}

	return &model.NotifyReservationCompletedResponse{
		Sent:   sent,
		Failed: failed,
	}, nil
}
