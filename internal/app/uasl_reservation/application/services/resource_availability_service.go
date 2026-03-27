package services

import (
	"context"
	"errors"
	"fmt"
	"sync"

	gatewayIF "uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/repositoryIF"
	"uasl-reservation/internal/pkg/logger"
	"uasl-reservation/internal/pkg/myerror"
)

type ResourceAvailabilityService struct {
	vehicleGW            gatewayIF.VehicleOpenAPIGatewayIF
	portGW               gatewayIF.PortOpenAPIGatewayIF
	externalResourceRepo repositoryIF.ExternalUaslResourceRepositoryIF
}

type ResourceNameMap struct {
	VehicleNames map[string]string
	PortNames    map[string]string
}

func NewResourceAvailabilityService(
	vehicleGW gatewayIF.VehicleOpenAPIGatewayIF,
	portGW gatewayIF.PortOpenAPIGatewayIF,
	externalResourceRepo repositoryIF.ExternalUaslResourceRepositoryIF,
) *ResourceAvailabilityService {
	return &ResourceAvailabilityService{
		vehicleGW:            vehicleGW,
		portGW:               portGW,
		externalResourceRepo: externalResourceRepo,
	}
}

func (c *ResourceAvailabilityService) CheckVehicleAvailability(
	ctx context.Context,
	vehicleID string,
	timeRange model.TimeRange,
) (*model.ResourceConflictResult, error) {
	if c.vehicleGW == nil {
		return nil, nil
	}

	req := model.VehicleFetchRequest{
		VehicleID: vehicleID,
		StartAt:   &timeRange.Start,
		EndAt:     &timeRange.End,
	}

	if err := timeRange.Validate(); err != nil {
		return nil, fmt.Errorf("invalid time range: %w", err)
	}

	reservations, err := c.vehicleGW.ListReservations(ctx, "", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle reservations: %w", err)
	}

	if len(reservations) > 0 {
		return &model.ResourceConflictResult{
			ConflictType:  "VEHICLE",
			ConflictedIDs: []string{vehicleID},
		}, nil
	}

	logger.LogInfo("vehicle availability check passed - vehicle_id=%s", vehicleID)
	return nil, nil
}

func (c *ResourceAvailabilityService) CheckPortAvailability(
	ctx context.Context,
	portID string,
	timeRange model.TimeRange,
) (*model.ResourceConflictResult, error) {

	if c.portGW == nil {
		return nil, nil
	}

	req := model.PortFetchRequest{
		PortID:   portID,
		TimeFrom: &timeRange.Start,
		TimeTo:   &timeRange.End,
	}
	if err := timeRange.Validate(); err != nil {
		return nil, fmt.Errorf("invalid time range: %w", err)
	}

	reservations, err := c.portGW.ListReservations(ctx, "", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get port reservations: %w", err)
	}

	if len(reservations) > 0 {
		return &model.ResourceConflictResult{
			ConflictType:  "PORT",
			ConflictedIDs: []string{portID},
		}, nil
	}

	logger.LogInfo("port availability check passed - port_id=%s", portID)
	return nil, nil
}

func (c *ResourceAvailabilityService) CheckCompositeResourcesAvailability(
	ctx context.Context,
	vehicleReqs []model.VehicleReservationRequest,
	portRequests []model.PortReservationRequest,
) (*model.ResourceConflictResult, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	totalCount := len(vehicleReqs) + len(portRequests)
	conflictChan := make(chan *model.ResourceConflictResult, totalCount)
	errChan := make(chan error, totalCount)
	for _, vr := range vehicleReqs {
		vehicleReq := vr
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
			}
			tr := model.TimeRange{Start: vehicleReq.StartAt, End: vehicleReq.EndAt}
			conflict, err := c.CheckVehicleAvailability(ctx, vehicleReq.VehicleID, tr)
			if err != nil {
				errChan <- fmt.Errorf("vehicle %s availability check failed: %w", vehicleReq.VehicleID, err)
				cancel()
			} else if conflict != nil {
				conflictChan <- conflict
				cancel()
			}
		}()
	}

	for _, portReq := range portRequests {
		req := portReq
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
			}

			if c.portGW != nil {
				info, err := c.portGW.GetDronePortInfoDetail(ctx, req.PortID)
				if err != nil {
					if _, ok := err.(*myerror.HTTPError); ok {
						logger.LogInfo("Port info not found in this system (will aggregate results later)",
							"port_id", req.PortID)

						return
					}

					logger.LogError("CheckCompositeResourcesAvailability: failed to get drone port info detail",
						"port_id", req.PortID, "error", err)
					errChan <- fmt.Errorf("port %s drone port info check failed: %w", req.PortID, err)
					cancel()
					return
				}
				if info.ActiveStatus != model.DronePortActiveStatusAvailable {
					logger.LogInfo("CheckCompositeResourcesAvailability: drone port is not available",
						"port_id", req.PortID, "active_status", info.ActiveStatus)
					errChan <- fmt.Errorf("port %s is not available: activeStatus=%d", req.PortID, info.ActiveStatus)
					cancel()
					return
				}
			}
			conflict, err := c.CheckPortAvailability(ctx, req.PortID, req.ToTimeRange())
			if err != nil {
				if _, ok := err.(*myerror.HTTPError); ok {
					logger.LogInfo("Port availability not found in this system (will aggregate results later)",
						"port_id", req.PortID)

					return
				}

				errChan <- fmt.Errorf("port %s availability check failed: %w", req.PortID, err)
				cancel()
			} else if conflict != nil {
				conflictChan <- conflict
				cancel()
			}
		}()
	}

	wg.Wait()
	close(errChan)
	close(conflictChan)

	for conflict := range conflictChan {
		if conflict != nil {
			return conflict, nil
		}
	}

	var cancellationErr error
	for err := range errChan {
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				if cancellationErr == nil {
					cancellationErr = err
				}
				continue
			}
			return nil, fmt.Errorf("composite resources availability check failed: %w", err)
		}
	}
	if cancellationErr != nil {
		return nil, fmt.Errorf("composite resources availability check failed: %w", cancellationErr)
	}

	logger.LogInfo("composite resources availability check passed - vehicles=%d, ports=%d", len(vehicleReqs), len(portRequests))

	return nil, nil
}

func (s *ResourceAvailabilityService) validateSynchronization(ctx context.Context, ids []string, resourceType model.ExternalResourceType, typeName string) (map[string]string, error) {
	nameMap := make(map[string]string)
	if len(ids) == 0 {
		return nameMap, nil
	}

	existingResources, err := s.externalResourceRepo.FindByResourceIDsAndType(ids, resourceType)
	if err != nil {
		logger.LogError("failed to validate "+typeName+" synchronization", "error", err)
		return nameMap, nil
	}

	existingIDMap := make(map[string]bool)
	for _, resource := range existingResources {
		existingIDMap[resource.ExResourceID] = true
		if resource.Name != "" {
			nameMap[resource.ExResourceID] = resource.Name
		}
	}

	var missingIDs []string
	for _, id := range ids {
		if !existingIDMap[id] {
			missingIDs = append(missingIDs, id)
		}
	}

	if len(missingIDs) > 0 {
		logger.LogError(typeName+"s not synchronized with external system", "missing_ids", missingIDs)
		return nameMap, nil
	}
	logger.LogInfo("ResourceSync: "+typeName+" synchronization confirmed - "+typeName+"s=%d", len(ids))
	return nameMap, nil
}

func (s *ResourceAvailabilityService) ValidateVehicleSynchronization(ctx context.Context, vehicleIDs []string) (map[string]string, error) {
	return s.validateSynchronization(ctx, vehicleIDs, model.ExternalResourceTypeVehicle, "vehicle")
}

func (s *ResourceAvailabilityService) ValidatePortSynchronization(ctx context.Context, portIDs []string) (map[string]string, error) {
	return s.validateSynchronization(ctx, portIDs, model.ExternalResourceTypePort, "port")
}

func (s *ResourceAvailabilityService) ValidateCompositeSynchronization(ctx context.Context, vehicleReqs []model.VehicleReservationRequest, portRequests []model.PortReservationRequest) (*ResourceNameMap, error) {
	vehicleIDs := make([]string, 0, len(vehicleReqs))
	for _, v := range vehicleReqs {
		vehicleIDs = append(vehicleIDs, v.VehicleID)
	}

	portIDs := make([]string, 0, len(portRequests))
	for _, p := range portRequests {
		portIDs = append(portIDs, p.PortID)
	}

	vehicleNames, err := s.ValidateVehicleSynchronization(ctx, vehicleIDs)
	if err != nil {
		return nil, err
	}
	portNames, err := s.ValidatePortSynchronization(ctx, portIDs)
	if err != nil {
		return nil, err
	}

	return &ResourceNameMap{
		VehicleNames: vehicleNames,
		PortNames:    portNames,
	}, nil
}

func (s *ResourceAvailabilityService) FetchVehicleReservations(
	ctx context.Context,
	vehicleIDs []string,
) ([]model.VehicleReservationInfo, error) {
	logger.LogInfo("FetchVehicleReservations started", "vehicle_ids_count", len(vehicleIDs))

	if s.vehicleGW == nil {
		logger.LogError("vehicleGW is not configured")
		return []model.VehicleReservationInfo{}, nil
	}

	allReservations := make([]model.VehicleReservationInfo, 0)

	for _, vehicleID := range vehicleIDs {
		req := model.VehicleFetchRequest{
			VehicleID: vehicleID,
			StartAt:   nil,
			EndAt:     nil,
		}

		reservations, err := s.vehicleGW.ListReservations(ctx, "", req)
		if err != nil {
			logger.LogError("Failed to fetch vehicle reservations from GSW API",
				"vehicle_id", vehicleID,
				"error", err)

			continue
		}

		allReservations = append(allReservations, reservations...)
	}

	logger.LogInfo("FetchVehicleReservations completed", "reservations_count", len(allReservations))
	return allReservations, nil
}

func (s *ResourceAvailabilityService) FetchPortReservations(
	ctx context.Context,
	portIDs []string,
) ([]model.PortReservationInfo, error) {
	logger.LogInfo("FetchPortReservations started", "port_ids_count", len(portIDs))

	if s.portGW == nil {
		logger.LogError("portGW is not configured")
		return []model.PortReservationInfo{}, nil
	}

	allReservations := make([]model.PortReservationInfo, 0)

	for _, portID := range portIDs {
		req := model.PortFetchRequest{
			PortID:   portID,
			TimeFrom: nil,
			TimeTo:   nil,
		}

		reservations, err := s.portGW.ListReservations(ctx, "", req)
		if err != nil {
			logger.LogError("Failed to fetch port reservations from GSW API",
				"port_id", portID,
				"error", err)

			continue
		}

		allReservations = append(allReservations, reservations...)
	}

	logger.LogInfo("FetchPortReservations completed", "reservations_count", len(allReservations))
	return allReservations, nil
}
