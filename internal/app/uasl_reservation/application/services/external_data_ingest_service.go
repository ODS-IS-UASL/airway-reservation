package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/repositoryIF"
	"uasl-reservation/internal/pkg/logger"
	"uasl-reservation/internal/pkg/retry"
	"uasl-reservation/internal/pkg/value"
)

type IngestTargets struct {
	UaslIDs    []string
	VehicleIDs []string
	PortIDs    []string
}

type ExternalDataIngestService struct {
	ouranosDiscoveryGW   gatewayIF.OuranosDiscoveryGatewayIF
	uaslDesignGateway    gatewayIF.UaslDesignGatewayIF
	vehicleGateway       gatewayIF.VehicleOpenAPIGatewayIF
	portGateway          gatewayIF.PortOpenAPIGatewayIF
	uaslAdminRepo        repositoryIF.UaslAdministratorRepositoryIF
	externalUaslDefRepo  repositoryIF.ExternalUaslDefinitionRepositoryIF
	externalResourceRepo repositoryIF.ExternalUaslResourceRepositoryIF
}

func NewExternalDataIngestService(
	ouranosDiscoveryGW gatewayIF.OuranosDiscoveryGatewayIF,
	uaslDesignGateway gatewayIF.UaslDesignGatewayIF,
	vehicleGateway gatewayIF.VehicleOpenAPIGatewayIF,
	portGateway gatewayIF.PortOpenAPIGatewayIF,
	uaslAdminRepo repositoryIF.UaslAdministratorRepositoryIF,
	externalUaslDefRepo repositoryIF.ExternalUaslDefinitionRepositoryIF,
	externalResourceRepo repositoryIF.ExternalUaslResourceRepositoryIF,
) *ExternalDataIngestService {
	return &ExternalDataIngestService{
		ouranosDiscoveryGW:   ouranosDiscoveryGW,
		uaslDesignGateway:    uaslDesignGateway,
		vehicleGateway:       vehicleGateway,
		portGateway:          portGateway,
		uaslAdminRepo:        uaslAdminRepo,
		externalUaslDefRepo:  externalUaslDefRepo,
		externalResourceRepo: externalResourceRepo,
	}
}

func (s *ExternalDataIngestService) IngestForGetAvailability(ctx context.Context, targets IngestTargets) error {
	logger.LogInfo("ExternalDataIngestService.IngestForGetAvailability started",
		"uasl_ids_count", len(targets.UaslIDs),
		"vehicle_ids_count", len(targets.VehicleIDs),
		"port_ids_count", len(targets.PortIDs))

	overallCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	eg, egCtx := errgroup.WithContext(overallCtx)
	sem := semaphore.NewWeighted(8)

	forceFullResourceIngest := false
	if s.externalResourceRepo != nil {
		hasAnyResources, err := s.externalResourceRepo.HasAnyExternalResources(egCtx)
		if err != nil {
			logger.LogError("HasAnyExternalResources failed", "error", err)
		} else {
			forceFullResourceIngest = !hasAnyResources
			if forceFullResourceIngest {
				logger.LogInfo("external_uasl_resources is empty: ingest both vehicles and ports")
			}
		}
	}

	var mu sync.Mutex
	var administratorsBuffer []model.UaslAdministrator
	var definitionsBuffer []model.ExternalUaslDefinition
	var resourcesBuffer []*model.ExternalUaslResource

	if len(targets.UaslIDs) > 0 {
		eg.Go(func() error {
			if err := sem.Acquire(egCtx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

			admins, defs, err := s.ingestUaslData(egCtx, targets.UaslIDs)
			if err != nil {
				logger.LogError("Failed to ingest uasl data", "error", err)

				return nil
			}

			mu.Lock()
			administratorsBuffer = append(administratorsBuffer, admins...)
			definitionsBuffer = append(definitionsBuffer, defs...)
			mu.Unlock()

			return nil
		})
	}

	if forceFullResourceIngest || len(targets.VehicleIDs) > 0 {
		eg.Go(func() error {
			if err := sem.Acquire(egCtx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

			var (
				resources []*model.ExternalUaslResource
				err       error
			)
			if forceFullResourceIngest {
				resources, err = s.fetchAllVehicles(egCtx)
			} else {
				resources, err = s.ingestVehicleData(egCtx, targets.VehicleIDs)
			}
			if err != nil {
				logger.LogError("Failed to ingest vehicle data", "error", err)

				return nil
			}

			mu.Lock()
			resourcesBuffer = append(resourcesBuffer, resources...)
			mu.Unlock()

			return nil
		})
	}

	if forceFullResourceIngest || len(targets.PortIDs) > 0 {
		eg.Go(func() error {
			if err := sem.Acquire(egCtx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

			var (
				resources []*model.ExternalUaslResource
				err       error
			)
			if forceFullResourceIngest {
				resources, err = s.fetchAllPorts(egCtx)
			} else {
				resources, err = s.ingestPortData(egCtx, targets.PortIDs)
			}
			if err != nil {
				logger.LogError("Failed to ingest port data", "error", err)

				return nil
			}

			mu.Lock()
			resourcesBuffer = append(resourcesBuffer, resources...)
			mu.Unlock()

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		logger.LogError("errgroup.Wait failed", "error", err)

	}

	if err := s.upsertCollectedData(ctx, administratorsBuffer, definitionsBuffer, resourcesBuffer); err != nil {
		logger.LogError("Failed to upsert collected data", "error", err)

	}

	logger.LogInfo("ExternalDataIngestService.IngestForGetAvailability completed",
		"administrators_count", len(administratorsBuffer),
		"definitions_count", len(definitionsBuffer),
		"resources_count", len(resourcesBuffer))

	return nil
}

func (s *ExternalDataIngestService) ingestUaslData(ctx context.Context, uaslIDs []string) ([]model.UaslAdministrator, []model.ExternalUaslDefinition, error) {
	logger.LogInfo("ingestUaslData started", "uasl_ids_count", len(uaslIDs))

	var admins []model.UaslAdministrator
	var defs []model.ExternalUaslDefinition

	hasAny, err := s.externalUaslDefRepo.HasAnyExternalDefinitions(ctx)
	if err != nil {
		logger.LogError("HasAnyExternalDefinitions failed, fallback to full fetch", "error", err)
		hasAny = false
	}

	if !hasAny {
		logger.LogInfo("ingestUaslData: DB is empty, fetching all uasl definitions")
		var bulkData *model.UaslBulkData
		op := func(rc context.Context) error {
			var e error
			bulkData, e = s.uaslDesignGateway.FetchUaslList(rc, "", true)
			return e
		}
		if err := retry.WithBackoff(ctx, op, retry.DefaultConfig()); err != nil {
			return nil, nil, fmt.Errorf("FetchUaslList failed after retries: %w", err)
		}
		logger.LogInfo("ingestUaslData: full fetch done",
			"administrators_count", len(bulkData.Administrators),
			"definitions_count", len(bulkData.Definitions))
		applyTemporaryPriceSettings(bulkData.Definitions)
		return bulkData.Administrators, bulkData.Definitions, nil
	}

	existingIDs, err := s.externalUaslDefRepo.FindExistingExUaslIDs(ctx, uaslIDs)
	if err != nil {
		logger.LogError("FindExistingExUaslIDs failed, fallback to full fetch", "uasl_ids", uaslIDs, "error", err)

		var bulkData *model.UaslBulkData
		op := func(rc context.Context) error {
			var e error
			bulkData, e = s.uaslDesignGateway.FetchUaslList(rc, "", true)
			return e
		}
		if err := retry.WithBackoff(ctx, op, retry.DefaultConfig()); err != nil {
			return nil, nil, fmt.Errorf("FetchUaslList failed after retries: %w", err)
		}
		applyTemporaryPriceSettings(bulkData.Definitions)
		return bulkData.Administrators, bulkData.Definitions, nil
	}

	existingSet := make(map[string]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existingSet[id] = struct{}{}
	}

	seenNew := make(map[string]struct{}, len(uaslIDs))
	newIDs := make([]string, 0, len(uaslIDs))
	for _, id := range uaslIDs {
		if _, found := existingSet[id]; found {
			continue
		}
		if _, alreadyQueued := seenNew[id]; alreadyQueued {
			continue
		}
		seenNew[id] = struct{}{}
		newIDs = append(newIDs, id)
	}

	if len(newIDs) == 0 {
		logger.LogInfo("ingestUaslData: all uasl_ids already exist in DB, skipping",
			"uasl_ids_count", len(uaslIDs))
		return admins, defs, nil
	}

	logger.LogInfo("ingestUaslData: new uasl_ids detected, fetching individually",
		"total", len(uaslIDs),
		"existing", len(existingIDs),
		"new_count", len(newIDs),
		"new_ids", newIDs)

	for _, uaslID := range newIDs {
		id := uaslID
		var bulkData *model.UaslBulkData
		op := func(rc context.Context) error {
			var e error
			bulkData, e = s.uaslDesignGateway.FetchUaslByID(rc, "", id, true)
			return e
		}
		if err := retry.WithBackoff(ctx, op, retry.DefaultConfig()); err != nil {

			logger.LogInfo("ingestUaslData: uasl_id not found in internal API, may belong to external system",
				"uasl_id", id,
				"error", err)
			continue
		}
		if bulkData == nil || (len(bulkData.Administrators) == 0 && len(bulkData.Definitions) == 0) {
			logger.LogInfo("ingestUaslData: empty response for uasl_id, may belong to external system",
				"uasl_id", id)
			continue
		}
		applyTemporaryPriceSettings(bulkData.Definitions)
		admins = append(admins, bulkData.Administrators...)
		defs = append(defs, bulkData.Definitions...)
		logger.LogInfo("ingestUaslData: fetched uasl_id",
			"uasl_id", id,
			"administrators", len(bulkData.Administrators),
			"definitions", len(bulkData.Definitions))
	}

	return admins, defs, nil
}

func (s *ExternalDataIngestService) ingestVehicleData(ctx context.Context, vehicleIDs []string) ([]*model.ExternalUaslResource, error) {
	logger.LogInfo("ingestVehicleData started", "vehicle_ids_count", len(vehicleIDs))

	hasAny, err := s.externalResourceRepo.HasAnyExternalResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("HasAnyExternalResources failed: %w", err)
	}

	if !hasAny {

		return s.fetchAllVehicles(ctx)
	}

	return s.fetchMissingVehicles(ctx, vehicleIDs)
}

func (s *ExternalDataIngestService) fetchAllVehicles(ctx context.Context) ([]*model.ExternalUaslResource, error) {
	var resources []*model.ExternalUaslResource
	op := func(rc context.Context) error {
		var err error

		resources, err = s.vehicleGateway.FetchAircraftList(rc, "", true)
		return err
	}
	if err := retry.WithBackoff(ctx, op, retry.DefaultConfig()); err != nil {
		return nil, fmt.Errorf("FetchAircraftList failed after retries: %w", err)
	}

	logger.LogInfo("fetchAllVehicles fetched", "count", len(resources))

	return resources, nil
}

func (s *ExternalDataIngestService) fetchMissingVehicles(ctx context.Context, vehicleIDs []string) ([]*model.ExternalUaslResource, error) {
	logger.LogInfo("fetchMissingVehicles started", "vehicle_ids_count", len(vehicleIDs))

	existingIDs, err := s.externalResourceRepo.FindExistingExResourceIDs(ctx, vehicleIDs)
	if err != nil {
		logger.LogError("FindExistingExResourceIDs failed for vehicles, fallback to full fetch", "error", err)
		return s.fetchAllVehicles(ctx)
	}

	existingSet := make(map[string]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existingSet[id] = struct{}{}
	}
	newIDs := make([]string, 0, len(vehicleIDs))
	for _, id := range vehicleIDs {
		if _, found := existingSet[id]; !found {
			newIDs = append(newIDs, id)
		}
	}

	if len(newIDs) == 0 {
		logger.LogInfo("fetchMissingVehicles: all vehicle_ids already exist in DB",
			"vehicle_ids_count", len(vehicleIDs))
		return nil, nil
	}

	logger.LogInfo("fetchMissingVehicles: new vehicle_ids detected",
		"total", len(vehicleIDs),
		"existing", len(existingIDs),
		"new_count", len(newIDs),
		"new_ids", newIDs)

	all, err := s.fetchAllVehicles(ctx)
	if err != nil {
		return nil, err
	}

	foundSet := make(map[string]struct{}, len(all))
	for _, r := range all {
		foundSet[r.ExResourceID] = struct{}{}
	}

	newIDSet := make(map[string]struct{}, len(newIDs))
	for _, id := range newIDs {
		newIDSet[id] = struct{}{}
	}

	var result []*model.ExternalUaslResource
	for _, r := range all {
		if _, ok := newIDSet[r.ExResourceID]; ok {
			result = append(result, r)
		}
	}

	for _, id := range newIDs {
		if _, found := foundSet[id]; !found {
			logger.LogInfo("fetchMissingVehicles: vehicle_id not found in internal API, may belong to external system",
				"vehicle_id", id)
		}
	}

	logger.LogInfo("fetchMissingVehicles completed", "new_records", len(result))
	return result, nil
}

func (s *ExternalDataIngestService) ingestPortData(ctx context.Context, portIDs []string) ([]*model.ExternalUaslResource, error) {
	logger.LogInfo("ingestPortData started", "port_ids_count", len(portIDs))

	hasAny, err := s.externalResourceRepo.HasAnyExternalResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("HasAnyExternalResources failed: %w", err)
	}

	if !hasAny {

		return s.fetchAllPorts(ctx)
	}

	return s.fetchMissingPorts(ctx, portIDs)
}

func (s *ExternalDataIngestService) fetchAllPorts(ctx context.Context) ([]*model.ExternalUaslResource, error) {
	var resources []*model.ExternalUaslResource
	op := func(rc context.Context) error {
		var err error

		resources, err = s.portGateway.FetchDronePortList(rc, "", true)
		return err
	}
	if err := retry.WithBackoff(ctx, op, retry.DefaultConfig()); err != nil {
		return nil, fmt.Errorf("FetchDronePortList failed after retries: %w", err)
	}

	logger.LogInfo("fetchAllPorts fetched", "count", len(resources))

	return resources, nil
}

func (s *ExternalDataIngestService) fetchMissingPorts(ctx context.Context, portIDs []string) ([]*model.ExternalUaslResource, error) {
	logger.LogInfo("fetchMissingPorts started", "port_ids_count", len(portIDs))

	existingIDs, err := s.externalResourceRepo.FindExistingExResourceIDs(ctx, portIDs)
	if err != nil {
		logger.LogError("FindExistingExResourceIDs failed for ports, fallback to full fetch", "error", err)
		return s.fetchAllPorts(ctx)
	}

	existingSet := make(map[string]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existingSet[id] = struct{}{}
	}
	newIDs := make([]string, 0, len(portIDs))
	for _, id := range portIDs {
		if _, found := existingSet[id]; !found {
			newIDs = append(newIDs, id)
		}
	}

	if len(newIDs) == 0 {
		logger.LogInfo("fetchMissingPorts: all port_ids already exist in DB",
			"port_ids_count", len(portIDs))
		return nil, nil
	}

	logger.LogInfo("fetchMissingPorts: new port_ids detected",
		"total", len(portIDs),
		"existing", len(existingIDs),
		"new_count", len(newIDs),
		"new_ids", newIDs)

	all, err := s.fetchAllPorts(ctx)
	if err != nil {
		return nil, err
	}

	foundSet := make(map[string]struct{}, len(all))
	for _, r := range all {
		foundSet[r.ExResourceID] = struct{}{}
	}

	newIDSet := make(map[string]struct{}, len(newIDs))
	for _, id := range newIDs {
		newIDSet[id] = struct{}{}
	}

	var result []*model.ExternalUaslResource
	for _, r := range all {
		if _, ok := newIDSet[r.ExResourceID]; ok {
			result = append(result, r)
		}
	}

	for _, id := range newIDs {
		if _, found := foundSet[id]; !found {
			logger.LogInfo("fetchMissingPorts: port_id not found in internal API, may belong to external system",
				"port_id", id)
		}
	}

	logger.LogInfo("fetchMissingPorts completed", "new_records", len(result))
	return result, nil
}

func (s *ExternalDataIngestService) upsertCollectedData(
	ctx context.Context,
	admins []model.UaslAdministrator,
	defs []model.ExternalUaslDefinition,
	resources []*model.ExternalUaslResource,
) error {
	logger.LogInfo("upsertCollectedData started",
		"administrators_count", len(admins),
		"definitions_count", len(defs),
		"resources_count", len(resources))

	if len(admins) > 0 {

		adminMap := make(map[string]*model.UaslAdministrator)
		mergeServices := func(base *model.UaslAdministrator, incoming *model.UaslAdministrator) {
			if base == nil || incoming == nil {
				return
			}

			if incoming.IsInternal && !base.IsInternal {
				base.IsInternal = true
			}

			if !base.BusinessNumber.Valid && incoming.BusinessNumber.Valid {
				base.BusinessNumber = incoming.BusinessNumber
			}
			if base.Name == "" && incoming.Name != "" {
				base.Name = incoming.Name
			}

			var merged model.ExternalServicesList
			seen := make(map[string]model.ExternalService)

			endpointScore := func(svc model.ExternalService) int {
				score := 0
				if svc.Services.BaseURL != "" {
					score++
				}
				return score
			}

			addList := func(src *model.UaslAdministrator) {
				if src == nil {
					return
				}
				list, err := src.GetExternalServicesList()
				if err != nil || len(list) == 0 {
					return
				}
				for _, svc := range list {
					if svc.ExUaslID == "" {
						continue
					}
					if existing, ok := seen[svc.ExUaslID]; ok {
						if endpointScore(svc) > endpointScore(existing) {
							seen[svc.ExUaslID] = svc
						}
					} else {
						seen[svc.ExUaslID] = svc
					}
				}
			}

			addList(base)
			addList(incoming)

			if len(seen) > 0 {
				merged = make(model.ExternalServicesList, 0, len(seen))
				for _, v := range seen {
					merged = append(merged, v)
				}
				if b, err := json.Marshal(merged); err == nil {
					raw := json.RawMessage(b)
					base.ExternalServices = value.NullJSON{Valid: true, JSON: &raw}
				}
			}
		}

		for i := range admins {
			key := admins[i].ExAdministratorID
			existing, ok := adminMap[key]
			if !ok {
				adminCopy := admins[i]
				adminMap[key] = &adminCopy
				continue
			}
			mergeServices(existing, &admins[i])
		}

		if s.uaslAdminRepo != nil {
			exAdminIDs := make([]string, 0, len(adminMap))
			for id := range adminMap {
				exAdminIDs = append(exAdminIDs, id)
			}
			if len(exAdminIDs) > 0 {
				if existingAdmins, err := s.uaslAdminRepo.FindByExAdministratorIDs(ctx, exAdminIDs); err == nil {
					for _, existing := range existingAdmins {
						if existing == nil {
							continue
						}
						if current, ok := adminMap[existing.ExAdministratorID]; ok {
							mergeServices(current, existing)
						}
					}
				} else {
					logger.LogError("FindByExAdministratorIDs failed for merge", "error", err)
				}
			}
		}

		adminPtrs := make([]*model.UaslAdministrator, 0, len(adminMap))
		for _, admin := range adminMap {
			adminPtrs = append(adminPtrs, admin)
		}
		if err := s.uaslAdminRepo.UpsertBatch(ctx, adminPtrs); err != nil {
			logger.LogError("UpsertBatch failed", "error", err)

		} else {
			logger.LogInfo("UpsertBatch succeeded", "count", len(adminPtrs))
		}
	}

	if len(defs) > 0 {

		defMap := make(map[string]model.ExternalUaslDefinition, len(defs))
		defOrder := make([]string, 0, len(defs))
		for _, d := range defs {
			if _, exists := defMap[d.ExUaslSectionID]; !exists {
				defMap[d.ExUaslSectionID] = d
				defOrder = append(defOrder, d.ExUaslSectionID)
			}
		}
		dedupedDefs := make([]model.ExternalUaslDefinition, 0, len(defMap))
		for _, sectionID := range defOrder {
			dedupedDefs = append(dedupedDefs, defMap[sectionID])
		}

		chunkSize := 500
		for i := 0; i < len(dedupedDefs); i += chunkSize {
			end := i + chunkSize
			if end > len(dedupedDefs) {
				end = len(dedupedDefs)
			}
			chunk := dedupedDefs[i:end]
			chunkPtrs := make([]*model.ExternalUaslDefinition, len(chunk))
			for j := range chunk {
				chunkPtrs[j] = &chunk[j]
			}
			if err := s.externalUaslDefRepo.UpsertBatch(ctx, chunkPtrs); err != nil {
				logger.LogError("UpsertBatch failed", "error", err, "chunk_start", i, "chunk_end", end)

			} else {
				logger.LogInfo("UpsertBatch succeeded", "chunk_start", i, "chunk_end", end)
			}
		}
	}

	if len(resources) > 0 {

		chunkSize := 500
		for i := 0; i < len(resources); i += chunkSize {
			end := i + chunkSize
			if end > len(resources) {
				end = len(resources)
			}
			chunk := resources[i:end]
			if err := s.externalResourceRepo.UpsertBatch(ctx, chunk); err != nil {
				logger.LogError("UpsertBatch failed", "error", err, "chunk_start", i, "chunk_end", end)

			} else {
				logger.LogInfo("UpsertBatch succeeded", "chunk_start", i, "chunk_end", end)
			}
		}
	}

	logger.LogInfo("upsertCollectedData completed")
	return nil
}

func applyTemporaryPriceSettings(definitions []model.ExternalUaslDefinition) {

	temporaryPrices := model.PriceInfoList{
		{
			PriceType:          "TIME_MINUTE",
			Price:              1000,
			PricePerUnit:       10,
			EffectiveStartDate: "2026-01-01",
			EffectiveEndDate:   "2026-12-31",
			EffectiveStartTime: "00:00",
			EffectiveEndTime:   "08:00",
		},
		{
			PriceType:          "TIME_MINUTE",
			Price:              2000,
			PricePerUnit:       10,
			EffectiveStartDate: "2026-01-01",
			EffectiveEndDate:   "2026-12-31",
			EffectiveStartTime: "08:00",
			EffectiveEndTime:   "17:00",
		},
		{
			PriceType:          "TIME_MINUTE",
			Price:              3000,
			PricePerUnit:       10,
			EffectiveStartDate: "2026-01-01",
			EffectiveEndDate:   "2026-12-31",
			EffectiveStartTime: "17:00",
			EffectiveEndTime:   "24:00",
		},
	}

	for i := range definitions {
		definitions[i].PriceInfo = temporaryPrices
	}
}
