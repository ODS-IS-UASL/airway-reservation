package converter

import (
	"fmt"
	"time"

	gsw "uasl-reservation/external/uasl/port/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/pkg/util"
	"uasl-reservation/internal/pkg/value"
)

type PortReservationConverter struct{}

func NewPortReservationConverter() *PortReservationConverter { return &PortReservationConverter{} }
func (c *PortReservationConverter) ToListReservationsRequest(req model.PortFetchRequest) gsw.DronePortReserveInfoListRequestDto {
	dto := gsw.DronePortReserveInfoListRequestDto{DronePortID: req.PortID}
	if req.TimeFrom != nil {
		dto.TimeFrom = req.TimeFrom.Format(time.RFC3339)
	}
	if req.TimeTo != nil {
		dto.TimeTo = req.TimeTo.Format(time.RFC3339)
	}
	return dto
}
func (c *PortReservationConverter) ToCreateReservationRequest(req model.PortReservationRequest) gsw.DronePortReserveInfoRegisterListRequestDto {
	elem := gsw.DronePortReserveElement{GroupReservationID: req.GroupReservationID, DronePortID: req.PortID, UsageType: int(req.UsageType), ReservationTimeFrom: req.ReservationTimeFrom.Format(time.RFC3339), ReservationTimeTo: req.ReservationTimeTo.Format(time.RFC3339)}
	if req.AircraftID != nil {
		elem.AircraftID = *req.AircraftID
	}
	reserveProviderID := ""
	if req.OperatorID != nil {
		reserveProviderID = req.OperatorID.ToString()
	}
	return gsw.DronePortReserveInfoRegisterListRequestDto{Data: []gsw.DronePortReserveElement{elem}, ReserveProviderID: reserveProviderID}
}
func (c *PortReservationConverter) ToReservationHandle(resp gsw.DronePortReserveInfoRegisterListResponseDto, url string) (model.ReservationHandle, error) {
	if len(resp.DronePortReservationIDs) == 0 {
		return model.ReservationHandle{}, nil
	}
	h, err := model.NewReservationHandle(resp.DronePortReservationIDs[0], model.ResourceTypePort, url)
	if err != nil {
		return model.ReservationHandle{}, err
	}
	h.ExternalIDs = resp.DronePortReservationIDs
	return h, nil
}
func (c *PortReservationConverter) ToPortReservationDetail(dto gsw.DronePortReserveInfoDetailResponseDto) (*model.PortReservationDetail, error) {
	start, err := time.Parse(time.RFC3339, dto.ReservationTimeFrom)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse(time.RFC3339, dto.ReservationTimeTo)
	if err != nil {
		return nil, err
	}
	return &model.PortReservationDetail{ReservationID: dto.DronePortReservationID, GroupReservationID: dto.GroupReservationID, PortID: dto.DronePortID, PortName: dto.DronePortName, UsageType: int32(dto.UsageType), StartAt: start, EndAt: end}, nil
}

func (c *PortReservationConverter) ToPortReservationRequestFromDetail(detail *model.PortReservationDetail, operatorID string) model.PortReservationRequest {
	if detail == nil {
		return model.PortReservationRequest{}
	}
	req := model.PortReservationRequest{
		PortID:              detail.PortID,
		AircraftID:          detail.VehicleID,
		GroupReservationID:  detail.GroupReservationID,
		UsageType:           detail.UsageType,
		ReservationTimeFrom: detail.StartAt,
		ReservationTimeTo:   detail.EndAt,
	}
	if operatorID != "" {
		opID := value.ModelID(operatorID)
		req.OperatorID = &opID
	}
	return req
}

func (c *PortReservationConverter) ToPortReservationDetailFromExternal(res *model.ExternalResourceReservation) (*model.PortReservationDetail, error) {
	if res.ResourceType != model.ExternalResourceTypePort {
		return nil, fmt.Errorf("invalid resource type: expected port, got %s", res.ResourceType)
	}
	detail := &model.PortReservationDetail{
		ReservationID:     res.ExReservationID,
		PortID:            res.ExResourceID,
		PortName:          res.ResourceName,
		ExAdministratorID: res.ExAdministratorID,
	}
	if res.StartAt != nil {
		detail.StartAt = *res.StartAt
	}
	if res.EndAt != nil {
		detail.EndAt = *res.EndAt
	}
	if res.UsageType != nil {
		detail.UsageType = int32(*res.UsageType)
	}
	if res.Amount != nil {
		detail.Amount = int32(*res.Amount)
	}
	return detail, nil
}

func (c *PortReservationConverter) ToPortReservationDetailsFromExternalElements(externalElements []model.PortReservationElement) []*model.PortReservationDetail {
	if len(externalElements) == 0 {
		return []*model.PortReservationDetail{}
	}
	details := make([]*model.PortReservationDetail, 0, len(externalElements))
	for _, exPort := range externalElements {
		var startAt, endAt time.Time
		if exPort.StartAt != "" {
			startAt, _ = time.Parse(time.RFC3339, exPort.StartAt)
		}
		if exPort.EndAt != "" {
			endAt, _ = time.Parse(time.RFC3339, exPort.EndAt)
		}
		details = append(details, &model.PortReservationDetail{
			PortID:            exPort.PortID,
			PortName:          exPort.Name,
			ReservationID:     exPort.ReservationID,
			StartAt:           startAt,
			EndAt:             endAt,
			UsageType:         util.SafeIntToInt32(exPort.UsageType),
			Amount:            util.SafeIntToInt32(exPort.Amount),
			ExAdministratorID: exPort.ExAdministratorID,
		})
	}
	return details
}

func (c *PortReservationConverter) ToPortElements(portDetails []*model.PortReservationDetail) []*model.PortElement {
	if len(portDetails) == 0 {
		return []*model.PortElement{}
	}
	result := make([]*model.PortElement, 0, len(portDetails))
	for _, detail := range portDetails {
		if detail == nil {
			continue
		}
		result = append(result, &model.PortElement{
			PortID:            detail.PortID,
			UsageType:         int(detail.UsageType),
			StartAt:           detail.StartAt.Format(time.RFC3339),
			EndAt:             detail.EndAt.Format(time.RFC3339),
			ReservationID:     detail.ReservationID,
			Name:              detail.PortName,
			ExAdministratorID: detail.ExAdministratorID,
			Amount:            int(detail.Amount),
		})
	}
	return result
}

func (c *PortReservationConverter) ToProtoPortReservations(infos []model.PortReservationInfo) []model.PortAvailabilityItem {
	result := make([]model.PortAvailabilityItem, 0, len(infos))
	for _, info := range infos {
		item := model.PortAvailabilityItem{
			PortID:  info.PortID,
			Name:    info.PortName,
			StartAt: info.StartAt,
			EndAt:   info.EndAt,
		}
		if info.ReservationID != nil {
			item.ReservationID = *info.ReservationID
		}
		result = append(result, item)
	}
	return result
}
