package converter

import (
	"fmt"
	"strconv"
	"time"

	gsw "uasl-reservation/external/uasl/vehicle/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/pkg/logger"
	"uasl-reservation/internal/pkg/util"
	"uasl-reservation/internal/pkg/value"
)

type VehicleReservationConverter struct{}

func NewVehicleReservationConverter() *VehicleReservationConverter {
	return &VehicleReservationConverter{}
}
func (c *VehicleReservationConverter) ToListReservationsRequest(req model.VehicleFetchRequest) gsw.AircraftReserveInfoListRequestDto {
	dto := gsw.AircraftReserveInfoListRequestDto{AircraftID: req.VehicleID}
	if req.StartAt != nil {
		dto.TimeFrom = req.StartAt.Format(time.RFC3339)
	}
	if req.EndAt != nil {
		dto.TimeTo = req.EndAt.Format(time.RFC3339)
	}
	return dto
}
func (c *VehicleReservationConverter) ToCreateReservationRequest(req model.VehicleReservationRequest) gsw.AircraftReserveInfoRequestDto {
	return gsw.AircraftReserveInfoRequestDto{GroupReservationID: req.GroupReservationID, AircraftID: req.VehicleID, ReservationTimeFrom: req.StartAt.Format(time.RFC3339), ReservationTimeTo: req.EndAt.Format(time.RFC3339)}
}
func (c *VehicleReservationConverter) ToReservationHandle(resp gsw.AircraftReserveInfoResponseDto, url string) (model.ReservationHandle, error) {
	h, err := model.NewReservationHandle(resp.AircraftReservationID, model.ResourceTypeVehicle, url)
	if err != nil {
		return model.ReservationHandle{}, err
	}
	h.ExternalIDs = []string{resp.AircraftReservationID}
	return h, nil
}
func (c *VehicleReservationConverter) ToVehicleReservationDetail(dto gsw.AircraftReserveInfoDetailResponseDto) (*model.VehicleReservationDetail, error) {
	start, err := time.Parse(time.RFC3339, dto.ReservationTimeFrom)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse(time.RFC3339, dto.ReservationTimeTo)
	if err != nil {
		return nil, err
	}
	return &model.VehicleReservationDetail{ReservationID: dto.AircraftReservationID, GroupReservationID: dto.GroupReservationID, VehicleID: dto.AircraftID, StartAt: start, EndAt: end, VehicleName: dto.AircraftName, OperatorID: dto.OperatorID}, nil
}
func (c *VehicleReservationConverter) ToAircraftInfoDetail(dto gsw.AircraftInfoDetailResponseDto) *model.AircraftInfoDetail {
	fileInfos := make([]model.AircraftFileInfo, len(dto.FileInfos))
	for i, f := range dto.FileInfos {
		fileInfos[i] = model.AircraftFileInfo{FileLogicalName: f.FileLogicalName, FilePhysicalName: f.FilePhysicalName, FileID: f.FileID}
	}
	payloadInfos := make([]model.AircraftPayloadInfo, len(dto.PayloadInfos))
	for i, p := range dto.PayloadInfos {
		payloadInfos[i] = model.AircraftPayloadInfo{PayloadID: p.PayloadID, PayloadName: p.PayloadName, PayloadDetailText: p.PayloadDetailText, ImageData: p.ImageData, FilePhysicalName: p.FilePhysicalName, OperatorID: p.OperatorID}
	}
	priceInfos := make([]model.ResourcePriceInfo, len(dto.PriceInfos))
	for i, pr := range dto.PriceInfos {
		start, _ := value.NewNullTimeFromString(pr.EffectiveStartTime)
		end, _ := value.NewNullTimeFromString(pr.EffectiveEndTime)
		priceInfos[i] = model.ResourcePriceInfo{
			PriceID:            pr.PriceID,
			PriceType:          pr.PriceType,
			PricePerUnit:       pr.PricePerUnit,
			Price:              pr.Price,
			EffectiveStartTime: start,
			EffectiveEndTime:   end,
			Priority:           pr.Priority,
			OperatorID:         value.ModelID(pr.OperatorID),
		}
	}
	return &model.AircraftInfoDetail{AircraftID: dto.AircraftID, AircraftName: dto.AircraftName, Manufacturer: dto.Manufacturer, ModelNumber: dto.ModelNumber, ModelName: dto.ModelName, ManufacturingNumber: dto.ManufacturingNumber, AircraftType: dto.AircraftType, MaxTakeoffWeight: dto.MaxTakeoffWeight, BodyWeight: dto.BodyWeight, MaxFlightSpeed: dto.MaxFlightSpeed, MaxFlightTime: dto.MaxFlightTime, Certification: dto.Certification, Lat: dto.Lat, Lon: dto.Lon, DipsRegistrationCode: dto.DipsRegistrationCode, OwnerType: dto.OwnerType, OwnerID: dto.OwnerID, OperatorID: dto.OperatorID, ImageData: dto.ImageData, FileInfos: fileInfos, PayloadInfos: payloadInfos, PriceInfos: priceInfos}
}
func (c *VehicleReservationConverter) ToVehicleReservationResponse(resp gsw.AircraftReserveInfoDetailResponseDto) (model.VehicleReservationResponse, error) {
	id, err := value.NewModelIDFromUUIDString(resp.AircraftReservationID)
	if err != nil {
		return model.VehicleReservationResponse{}, err
	}
	start, err := time.Parse(time.RFC3339, resp.ReservationTimeFrom)
	if err != nil {
		return model.VehicleReservationResponse{}, err
	}
	end, err := time.Parse(time.RFC3339, resp.ReservationTimeTo)
	if err != nil {
		return model.VehicleReservationResponse{}, err
	}
	return model.VehicleReservationResponse{ID: id, VehicleID: resp.AircraftID, StartAt: start, EndAt: end}, nil
}

func (c *VehicleReservationConverter) ToVehicleReservationRequestFromDetail(detail *model.VehicleReservationDetail, operatorID string) model.VehicleReservationRequest {
	if detail == nil {
		return model.VehicleReservationRequest{}
	}
	req := model.VehicleReservationRequest{
		VehicleID:          detail.VehicleID,
		GroupReservationID: detail.GroupReservationID,
		StartAt:            detail.StartAt,
		EndAt:              detail.EndAt,
	}
	if operatorID != "" {
		opID := value.ModelID(operatorID)
		req.OperatorID = &opID
	}
	return req
}

func (c *VehicleReservationConverter) ToVehicleReservationDetailFromExternal(res *model.ExternalResourceReservation) (*model.VehicleReservationDetail, error) {
	if res.ResourceType != model.ExternalResourceTypeVehicle {
		return nil, fmt.Errorf("invalid resource type: expected vehicle, got %s", res.ResourceType)
	}
	detail := &model.VehicleReservationDetail{
		ReservationID: res.ExReservationID,
		VehicleID:     res.ExResourceID,
		VehicleName:   res.ResourceName,
	}
	if res.StartAt != nil {
		detail.StartAt = *res.StartAt
	}
	if res.EndAt != nil {
		detail.EndAt = *res.EndAt
	}
	if res.Amount != nil {
		detail.Amount = int32(*res.Amount)
	}
	return detail, nil
}

func (c *VehicleReservationConverter) ToVehicleReservationDetailsFromExternalElements(externalElements []model.VehicleReservationElement) []*model.VehicleReservationDetail {
	if len(externalElements) == 0 {
		return []*model.VehicleReservationDetail{}
	}
	details := make([]*model.VehicleReservationDetail, 0, len(externalElements))
	for _, exVehicle := range externalElements {
		var startAt, endAt time.Time
		if exVehicle.StartAt != "" {
			startAt, _ = time.Parse(time.RFC3339, exVehicle.StartAt)
		}
		if exVehicle.EndAt != "" {
			endAt, _ = time.Parse(time.RFC3339, exVehicle.EndAt)
		}
		amountVal := int32(0)
		if exVehicle.Amount > 0 {
			val, err := util.SafeIntToInt32WithError(exVehicle.Amount)
			if err != nil {
				logger.LogInfo("Vehicle amount overflow, capping to max int32", "vehicle_id", exVehicle.VehicleID, "amount", exVehicle.Amount)
				amountVal = 1<<31 - 1
			} else {
				amountVal = val
			}
		}
		details = append(details, &model.VehicleReservationDetail{
			VehicleID:     exVehicle.VehicleID,
			VehicleName:   exVehicle.Name,
			ReservationID: exVehicle.ReservationID,
			StartAt:       startAt,
			EndAt:         endAt,
			Amount:        amountVal,
		})
	}
	return details
}

func (c *VehicleReservationConverter) ToVehicleElements(vehicleDetails []*model.VehicleReservationDetail) []*model.VehicleElement {
	if len(vehicleDetails) == 0 {
		return []*model.VehicleElement{}
	}
	result := make([]*model.VehicleElement, 0, len(vehicleDetails))
	for _, detail := range vehicleDetails {
		if detail == nil {
			continue
		}
		elem := &model.VehicleElement{
			VehicleID:     detail.VehicleID,
			ReservationID: detail.ReservationID,
			Name:          detail.VehicleName,
			StartAt:       detail.StartAt.Format(time.RFC3339),
			EndAt:         detail.EndAt.Format(time.RFC3339),
			Amount:        int(detail.Amount),
		}
		if detail.AircraftInfo != nil {
			length, _ := strconv.ParseFloat(detail.AircraftInfo.Length, 64)
			elem.AircraftInfo = &model.ReservationAircraftInfo{
				AircraftInfoID: detail.AircraftInfo.AircraftInfoID,
				RegistrationID: detail.AircraftInfo.RegistrationID,
				Maker:          detail.AircraftInfo.Maker,
				ModelNumber:    detail.AircraftInfo.ModelNumber,
				Name:           detail.AircraftInfo.Name,
				Type:           detail.AircraftInfo.Type,
				Length:         length,
			}
		}
		result = append(result, elem)
	}
	return result
}

func (c *VehicleReservationConverter) ToProtoVehicleReservations(infos []model.VehicleReservationInfo) []model.VehicleAvailabilityItem {
	result := make([]model.VehicleAvailabilityItem, 0, len(infos))
	for _, info := range infos {
		item := model.VehicleAvailabilityItem{
			VehicleID: info.VehicleID,
			Name:      info.VehicleName,
			StartAt:   info.StartAt,
			EndAt:     info.EndAt,
		}
		if info.ReservationID != nil {
			item.ReservationID = *info.ReservationID
		}
		result = append(result, item)
	}
	return result
}

func ToVehicleDetail(info *model.VehicleDetailInfo) (*model.VehicleDetailInfo, error) {
	if info == nil {
		return nil, nil
	}
	return &model.VehicleDetailInfo{
		AircraftInfoID: info.AircraftInfoID,
		RegistrationID: info.RegistrationID,
		Maker:          info.Maker,
		ModelNumber:    info.ModelNumber,
		Name:           info.Name,
		Type:           info.Type,
		Length:         info.Length,
	}, nil
}
