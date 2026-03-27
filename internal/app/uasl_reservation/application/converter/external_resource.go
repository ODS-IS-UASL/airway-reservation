package converter

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/datatypes"

	portmodel "uasl-reservation/external/uasl/port/model"
	vehiclemodel "uasl-reservation/external/uasl/vehicle/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/pkg/logger"
	"uasl-reservation/internal/pkg/util"
	"uasl-reservation/internal/pkg/value"
)

type ExternalResourceConverter struct {
	vehicleConv *VehicleReservationConverter
	portConv    *PortReservationConverter
}

func NewExternalResourceConverter() *ExternalResourceConverter {
	return &ExternalResourceConverter{
		vehicleConv: NewVehicleReservationConverter(),
		portConv:    NewPortReservationConverter(),
	}
}

func (c *ExternalResourceConverter) ToExternalResourceElements(
	gswPrices *model.ResourcePriceResult,
	externalReservationResults []*model.ExternalReservationResult,
	vehicles []model.VehicleReservationRequest,
	portReservations []model.PortReservationRequest,
	parentDomain *model.UaslReservation,
	requestID value.ModelID,
	internalResourceAdministratorID string,
	vehicleNames map[string]string,
	portNames map[string]string,
) (
	vehicleElements []*model.VehicleElement,
	portElements []*model.PortElement,
	externalResourceMappings []*model.ExternalResourceReservation,
	externalResourceTotalAmount int32,
) {
	if requestID == "" {
		logger.LogError("Cannot build external resource mappings: empty request ID")
		return []*model.VehicleElement{}, []*model.PortElement{}, []*model.ExternalResourceReservation{}, 0
	}

	vehicleElements = make([]*model.VehicleElement, 0)
	portElements = make([]*model.PortElement, 0)
	externalResourceMappings = make([]*model.ExternalResourceReservation, 0)
	externalResourceTotalAmount = 0

	for vehicleID, price := range gswPrices.VehiclePrices {

		startAt, endAt := c.findVehicleTimeRange(vehicles, vehicleID, parentDomain)
		vehicleElements = append(vehicleElements, &model.VehicleElement{
			VehicleID: vehicleID,
			Name:      vehicleNames[vehicleID],
			Amount:    int(price),
			StartAt:   startAt.Format(time.RFC3339),
			EndAt:     endAt.Format(time.RFC3339),
		})

		resRow, err := model.NewExternalResourceReservation(requestID, "", model.ExternalResourceTypeVehicle)
		if err != nil {
			logger.LogError("Failed to build external resource reservation for vehicle",
				"error", err, "vehicle_id", vehicleID)
			continue
		}
		amt := int(price)
		resRow.ExResourceID = vehicleID
		resRow.Amount = &amt
		resRow.ExAdministratorID = internalResourceAdministratorID
		if !startAt.IsZero() {
			resRow.StartAt = &startAt
		}
		if !endAt.IsZero() {
			resRow.EndAt = &endAt
		}
		externalResourceMappings = append(externalResourceMappings, resRow)
	}

	for portID, price := range gswPrices.PortPrices {

		portInfos := c.findAllPortReservationInfos(portReservations, portID)

		if len(portInfos) == 0 {

			portElements = append(portElements, &model.PortElement{
				PortID:            portID,
				Name:              portNames[portID],
				UsageType:         0,
				Amount:            int(price),
				ExAdministratorID: internalResourceAdministratorID,
			})

			resRow, err := model.NewExternalResourceReservation(requestID, "", model.ExternalResourceTypePort)
			if err != nil {
				logger.LogError("Failed to build external resource reservation for port",
					"error", err, "port_id", portID)
				continue
			}
			amt := int(price)
			resRow.ExResourceID = portID
			resRow.Amount = &amt
			resRow.ExAdministratorID = internalResourceAdministratorID
			externalResourceMappings = append(externalResourceMappings, resRow)
			continue
		}

		for _, portInfo := range portInfos {
			portElements = append(portElements, &model.PortElement{
				PortID:            portID,
				Name:              portNames[portID],
				UsageType:         int(portInfo.UsageType),
				Amount:            int(price),
				StartAt:           portInfo.ReservationTimeFrom.Format(time.RFC3339),
				EndAt:             portInfo.ReservationTimeTo.Format(time.RFC3339),
				ExAdministratorID: internalResourceAdministratorID,
			})

			resRow, err := model.NewExternalResourceReservation(requestID, "", model.ExternalResourceTypePort)
			if err != nil {
				logger.LogError("Failed to build external resource reservation for port",
					"error", err, "port_id", portID)
				continue
			}
			amt := int(price)
			resRow.ExResourceID = portID
			resRow.Amount = &amt
			resRow.ExAdministratorID = internalResourceAdministratorID
			u := int(portInfo.UsageType)
			resRow.UsageType = &u
			startAt := portInfo.ReservationTimeFrom
			endAt := portInfo.ReservationTimeTo
			if !startAt.IsZero() {
				resRow.StartAt = &startAt
			}
			if !endAt.IsZero() {
				resRow.EndAt = &endAt
			}
			externalResourceMappings = append(externalResourceMappings, resRow)
		}
	}

	for _, extResult := range externalReservationResults {
		if len(extResult.ChildDomains) == 0 {
			continue
		}

		vehiclePriceMap := make(map[string]int)

		if extResult.ReservationData != nil {

			for _, destRes := range extResult.ReservationData.DestinationReservations {
				for _, vehicle := range destRes.Vehicles {
					vehiclePriceMap[vehicle.VehicleID] = vehicle.Amount
				}
			}
		}

		for _, vehicleReq := range vehicles {

			if price, exists := vehiclePriceMap[vehicleReq.VehicleID]; exists {
				resRow, err := model.NewExternalResourceReservation(requestID, "", model.ExternalResourceTypeVehicle)
				if err != nil {
					logger.LogError("Failed to build external resource reservation (external vehicle)",
						"error", err, "vehicle_id", vehicleReq.VehicleID)
					continue
				}

				resRow.ExResourceID = vehicleReq.VehicleID
				resRow.ExAdministratorID = extResult.AdministratorID
				amt := price
				resRow.Amount = &amt
				startAt := vehicleReq.StartAt
				endAt := vehicleReq.EndAt
				if !startAt.IsZero() {
					resRow.StartAt = &startAt
				}
				if !endAt.IsZero() {
					resRow.EndAt = &endAt
				}

				vehAmount := util.SafeIntToInt32(price)
				vehicleElements = append(vehicleElements, &model.VehicleElement{
					VehicleID: vehicleReq.VehicleID,
					Name:      vehicleNames[vehicleReq.VehicleID],
					Amount:    int(vehAmount),
					StartAt:   vehicleReq.StartAt.Format(time.RFC3339),
					EndAt:     vehicleReq.EndAt.Format(time.RFC3339),
				})

				externalResourceMappings = append(externalResourceMappings, resRow)

				if extResult.AdministratorID != internalResourceAdministratorID {
					externalResourceTotalAmount += vehAmount
				}

				logger.LogInfo("Created external vehicle mapping from API response",
					"vehicle_id", vehicleReq.VehicleID,
					"amount", price,
					"administrator_id", extResult.AdministratorID)
			}
		}

		portReqMap := make(map[string]model.PortReservationRequest, len(extResult.PortRequests))
		for _, pr := range extResult.PortRequests {
			key := fmt.Sprintf("%s:%d", pr.PortID, pr.UsageType)
			portReqMap[key] = pr
		}

		if extResult.ReservationData != nil {
			for _, destRes := range extResult.ReservationData.DestinationReservations {
				for _, port := range destRes.Ports {
					resRow, err := model.NewExternalResourceReservation(requestID, "", model.ExternalResourceTypePort)
					if err != nil {
						logger.LogError("Failed to build external resource reservation (external port)",
							"error", err, "port_id", port.PortID)
						continue
					}

					resRow.ExResourceID = port.PortID
					resRow.ExAdministratorID = extResult.AdministratorID

					portReqKey := fmt.Sprintf("%s:%d", port.PortID, port.UsageType)
					u := port.UsageType
					resRow.UsageType = &u
					if portReq, ok := portReqMap[portReqKey]; ok {
						startAt := portReq.ReservationTimeFrom
						endAt := portReq.ReservationTimeTo
						if !startAt.IsZero() {
							resRow.StartAt = &startAt
						}
						if !endAt.IsZero() {
							resRow.EndAt = &endAt
						}
					}

					portAmount := util.SafeIntToInt32(port.Amount)
					amt := int(portAmount)
					resRow.Amount = &amt

					portName := portNames[port.PortID]
					if portName == "" {
						portName = port.Name
					}

					usageType := int32(0)
					if resRow.UsageType != nil {
						usageType = util.SafeIntToInt32(*resRow.UsageType)
					}
					startAtStr := ""
					endAtStr := ""
					if resRow.StartAt != nil {
						startAtStr = resRow.StartAt.Format(time.RFC3339)
					}
					if resRow.EndAt != nil {
						endAtStr = resRow.EndAt.Format(time.RFC3339)
					}

					portElements = append(portElements, &model.PortElement{
						PortID:            port.PortID,
						Name:              portName,
						UsageType:         int(usageType),
						Amount:            int(portAmount),
						StartAt:           startAtStr,
						EndAt:             endAtStr,
						ExAdministratorID: extResult.AdministratorID,
					})

					externalResourceMappings = append(externalResourceMappings, resRow)

					if extResult.AdministratorID != internalResourceAdministratorID {
						externalResourceTotalAmount += portAmount
					}

					logger.LogInfo("Created external port mapping from API response",
						"port_id", port.PortID,
						"usage_type", usageType,
						"amount", portAmount,
						"administrator_id", extResult.AdministratorID)
				}
			}
		}
	}

	logger.LogInfo("External resource elements built",
		"vehicle_elements", len(vehicleElements),
		"port_elements", len(portElements),
		"external_resource_mappings", len(externalResourceMappings),
		"external_resource_total_amount", externalResourceTotalAmount)

	return vehicleElements, portElements, externalResourceMappings, externalResourceTotalAmount
}

func (c *ExternalResourceConverter) findVehicleTimeRange(
	vehicles []model.VehicleReservationRequest,
	vehicleID string,
	parentDomain *model.UaslReservation,
) (startAt, endAt time.Time) {
	for _, v := range vehicles {
		if v.VehicleID == vehicleID {
			startAt = v.StartAt
			endAt = v.EndAt
			break
		}
	}

	if startAt.IsZero() && parentDomain != nil {
		startAt = parentDomain.StartAt
	}
	if endAt.IsZero() && parentDomain != nil {
		endAt = parentDomain.EndAt
	}

	return startAt, endAt
}

func (c *ExternalResourceConverter) findAllPortReservationInfos(
	portReservations []model.PortReservationRequest,
	portID string,
) []model.PortReservationRequest {
	var result []model.PortReservationRequest
	for i := range portReservations {
		if portReservations[i].PortID == portID {
			result = append(result, portReservations[i])
		}
	}
	return result
}

func ToExternalUaslResourceFromAircraft(external *vehiclemodel.AircraftInfoSearchListResponseDto) []*model.ExternalUaslResource {
	if external == nil || len(external.Data) == 0 {
		return []*model.ExternalUaslResource{}
	}

	resources := make([]*model.ExternalUaslResource, 0, len(external.Data))
	for _, ext := range external.Data {

		orgID, err := value.NewModelIDFromUUIDString(ext.OperatorID)
		if err != nil {
			logger.LogError("failed to parse organizationID", "operatorID", ext.OperatorID, "error", err)
			continue
		}

		priceInfos := ToExternalResourcePriceInfoList(ext.PriceInfos)

		aircraftJSON, err := json.Marshal(ext)
		if err != nil {
			logger.LogError("failed to marshal aircraft info", "aircraftID", ext.AircraftID, "error", err)
			aircraftJSON = nil
		}

		resource := &model.ExternalUaslResource{
			ID:                      value.NewEmptyModelID(),
			Name:                    ext.AircraftName,
			ExResourceID:            ext.AircraftID,
			ResourceType:            model.ResourceTypeVehicle,
			OrganizationID:          orgID,
			EstimatedPricePerMinute: priceInfos,
		}

		if aircraftJSON != nil {
			resource.AircraftInfo = &datatypes.JSON{}
			_ = resource.AircraftInfo.Scan(aircraftJSON)
		}

		resources = append(resources, resource)
	}

	return resources
}

func ToExternalUaslResourceFromPort(external *portmodel.DronePortInfoListResponseDto) []*model.ExternalUaslResource {
	if external == nil || len(external.Data) == 0 {
		return []*model.ExternalUaslResource{}
	}

	resources := make([]*model.ExternalUaslResource, 0, len(external.Data))
	for _, ext := range external.Data {

		orgID, err := value.NewModelIDFromUUIDString(ext.OperatorID)
		if err != nil {
			logger.LogError("failed to parse organizationID", "operatorID", ext.OperatorID, "error", err)
			continue
		}

		priceInfos := ToExternalResourcePriceInfoList(ext.PriceInfos)

		portJSON, err := json.Marshal(ext)
		if err != nil {
			logger.LogError("failed to marshal port info", "dronePortID", ext.DronePortID, "error", err)
			portJSON = nil
		}

		resource := &model.ExternalUaslResource{
			ID:                      value.NewEmptyModelID(),
			Name:                    ext.DronePortName,
			ExResourceID:            ext.DronePortID,
			ResourceType:            model.ResourceTypePort,
			OrganizationID:          orgID,
			EstimatedPricePerMinute: priceInfos,
		}

		if portJSON != nil {
			resource.AircraftInfo = &datatypes.JSON{}
			_ = resource.AircraftInfo.Scan(portJSON)
		}

		resources = append(resources, resource)
	}

	return resources
}
