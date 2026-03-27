package converter

import (
	"fmt"
	"strings"

	gswmodel "uasl-reservation/external/uasl/resource_price/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/pkg/logger"
	"uasl-reservation/internal/pkg/value"
)

func ToDomainResourcePriceList(src *gswmodel.ResourcePriceListResponse) (model.ResourcePriceList, error) {
	if src == nil {
		return model.ResourcePriceList{}, nil
	}

	resources := make([]model.ResourcePrice, 0, len(src.Resources))
	for _, r := range src.Resources {
		rp := model.ResourcePrice{
			ResourceID:   string(r.ResourceID),
			ResourceType: r.ResourceType,
			PriceInfos:   make([]model.ResourcePriceInfo, 0, len(r.PriceInfos)),
		}

		for _, pi := range r.PriceInfos {
			start, err := value.NewNullTimeFromString(pi.EffectiveStartTime)
			if err != nil {
				logger.LogError("[converter/price] ToDomainResourcePriceList: invalid effectiveStartTime - price_id=%s, value=%s, error=%v",
					pi.PriceID, pi.EffectiveStartTime, err)
				return model.ResourcePriceList{}, fmt.Errorf("invalid effectiveStartTime for priceId=%s: %w", pi.PriceID, err)
			}
			end, err := value.NewNullTimeFromString(pi.EffectiveEndTime)
			if err != nil {
				logger.LogError("[converter/price] ToDomainResourcePriceList: invalid effectiveEndTime - price_id=%s, value=%s, error=%v",
					pi.PriceID, pi.EffectiveEndTime, err)
				return model.ResourcePriceList{}, fmt.Errorf("invalid effectiveEndTime for priceId=%s: %w", pi.PriceID, err)
			}

			info := model.ResourcePriceInfo{
				PriceID:            string(pi.PriceID),
				PriceType:          pi.PriceType,
				PricePerUnit:       pi.PricePerUnit,
				Price:              pi.Price,
				EffectiveStartTime: start,
				EffectiveEndTime:   end,
				Priority:           pi.Priority,
				OperatorID:         value.ModelID(pi.OperatorID),
			}

			rp.PriceInfos = append(rp.PriceInfos, info)
		}

		resources = append(resources, rp)
	}

	return model.ResourcePriceList{Resources: resources}, nil
}

func ToExternalResourcePriceRequest(req model.ResourcePriceListRequest) gswmodel.ResourcePriceListRequest {

	ext := gswmodel.ResourcePriceListRequest{}
	if len(req.ResourceIDs) > 0 {
		ids := make([]string, 0, len(req.ResourceIDs))
		for _, id := range req.ResourceIDs {
			ids = append(ids, string(id))
		}
		ext.ResourceID = strings.Join(ids, ",")
	}
	if req.ResourceType != nil {
		ext.ResourceType = req.ResourceType
	}
	if req.PriceType != nil {
		ext.PriceType = req.PriceType
	}
	ext.EffectiveStartTime = req.EffectiveStartTime.ToString()
	ext.EffectiveEndTime = req.EffectiveEndTime.ToString()

	return ext
}

func ToExternalResourcePriceInfoList(external []gswmodel.PriceInfoSearchListDetailElement) model.ExternalResourcePriceInfoList {
	if len(external) == 0 {
		return model.ExternalResourcePriceInfoList{}
	}

	priceInfos := make(model.ExternalResourcePriceInfoList, 0, len(external))
	for _, ext := range external {
		priceInfos = append(priceInfos, model.ExternalResourcePriceInfo{
			PriceType:          ext.PriceType,
			Price:              ext.Price,
			PricePerUnit:       ext.PricePerUnit,
			Priority:           ext.Priority,
			EffectiveStartTime: ext.EffectiveStartTime,
			EffectiveEndTime:   ext.EffectiveEndTime,
		})
	}

	return priceInfos
}

func ToResourcePriceInfoList(src model.ExternalResourcePriceInfoList) []model.ResourcePriceInfo {
	result := make([]model.ResourcePriceInfo, 0, len(src))
	for _, s := range src {
		start, err := value.NewNullTimeFromString(s.EffectiveStartTime)
		if err != nil {
			logger.LogError("[converter/price] ToResourcePriceInfoList: invalid effectiveStartTime - value=%s, error=%v", s.EffectiveStartTime, err)
			start = value.NewEmptyNullTime()
		}
		end, err := value.NewNullTimeFromString(s.EffectiveEndTime)
		if err != nil {
			logger.LogError("[converter/price] ToResourcePriceInfoList: invalid effectiveEndTime - value=%s, error=%v", s.EffectiveEndTime, err)
			end = value.NewEmptyNullTime()
		}
		result = append(result, model.ResourcePriceInfo{
			PriceType:          s.PriceType,
			Price:              s.Price,
			PricePerUnit:       s.PricePerUnit,
			Priority:           s.Priority,
			EffectiveStartTime: start,
			EffectiveEndTime:   end,
		})
	}
	return result
}
