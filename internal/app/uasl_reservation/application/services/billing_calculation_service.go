package services

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	gatewayIF "uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/repositoryIF"
	"uasl-reservation/internal/pkg/logger"
	"uasl-reservation/internal/pkg/retry"
	"uasl-reservation/internal/pkg/util"
	"uasl-reservation/internal/pkg/value"
)

type TimeSegment struct {
	StartTime time.Time
	EndTime   time.Time
	Rule      *model.PriceInfo
}

type BillingCalculationService struct {
	gswPriceGW          gatewayIF.ResourcePriceGatewayIF
	orchestrator        *CompositeReservationOrchestrator
	externalUaslDefRepo repositoryIF.ExternalUaslDefinitionRepositoryIF
}

func NewBillingCalculationService(
	gswPriceGW gatewayIF.ResourcePriceGatewayIF,
	orchestrator *CompositeReservationOrchestrator,
	externalUaslDefRepo repositoryIF.ExternalUaslDefinitionRepositoryIF,
) *BillingCalculationService {
	return &BillingCalculationService{
		gswPriceGW:          gswPriceGW,
		orchestrator:        orchestrator,
		externalUaslDefRepo: externalUaslDefRepo,
	}
}

func (s *BillingCalculationService) CalculateResourcePrices(
	ctx context.Context,
	vehicles []model.VehicleReservationRequest,
	ports []model.PortReservationRequest,
) (*model.ResourcePriceResult, error) {
	result := &model.ResourcePriceResult{
		VehiclePrices: make(map[string]int32),
		PortPrices:    make(map[string]int32),
		TotalAmount:   0,
	}

	if s.gswPriceGW == nil {
		return result, nil
	}

	if len(vehicles) == 0 && len(ports) == 0 {
		return result, nil
	}

	g, gctx := errgroup.WithContext(ctx)

	if len(vehicles) > 0 {
		g.Go(func() error {
			if err := s.fetchVehiclePrices(gctx, vehicles, result); err != nil {
				logger.LogError("Failed to fetch vehicle prices", "error", err)

				return fmt.Errorf("fetch vehicle prices failed: %w", err)
			}
			return nil
		})
	}

	if len(ports) > 0 {
		g.Go(func() error {
			if err := s.fetchPortPrices(gctx, ports, result); err != nil {
				logger.LogError("Failed to fetch port prices", "error", err)

				return fmt.Errorf("fetch port prices failed: %w", err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return result, err
	}

	for _, amount := range result.VehiclePrices {
		result.TotalAmount += amount
	}
	for _, amount := range result.PortPrices {
		result.TotalAmount += amount
	}

	return result, nil
}

func (s *BillingCalculationService) fetchVehiclePrices(
	ctx context.Context,
	vehicles []model.VehicleReservationRequest,
	result *model.ResourcePriceResult,
) error {
	if s.orchestrator == nil {
		for _, vehicle := range vehicles {
			result.VehiclePrices[vehicle.VehicleID] = 0
		}
		return nil
	}

	g, gctx := errgroup.WithContext(ctx)
	var mu sync.Mutex

	for _, vehicle := range vehicles {
		vehicle := vehicle
		g.Go(func() error {
			amount, err := s.fetchSingleResourcePrice(
				gctx,
				vehicle.VehicleID,
				vehicle.StartAt,
				vehicle.EndAt,
				30,
				"vehicle",
			)
			if err != nil {
				logger.LogError("Failed to fetch vehicle price from GSW", "vehicle_id", vehicle.VehicleID, "error", err)

				return fmt.Errorf("vehicle_id=%s: %w", vehicle.VehicleID, err)
			}

			mu.Lock()
			result.VehiclePrices[vehicle.VehicleID] = amount
			mu.Unlock()

			logger.LogInfo("Successfully fetched vehicle price from GSW", "vehicle_id", vehicle.VehicleID, "amount", amount)
			return nil
		})
	}

	return g.Wait()
}

type portDedupeKey struct {
	portID string
	from   time.Time
	to     time.Time
}

func (s *BillingCalculationService) fetchPortPrices(
	ctx context.Context,
	ports []model.PortReservationRequest,
	result *model.ResourcePriceResult,
) error {
	if s.orchestrator == nil {
		for _, port := range ports {
			result.PortPrices[port.PortID] = 0
		}
		return nil
	}

	seen := make(map[portDedupeKey]bool)
	dedupedPorts := make([]model.PortReservationRequest, 0, len(ports))
	for _, port := range ports {
		key := portDedupeKey{
			portID: port.PortID,
			from:   port.ReservationTimeFrom,
			to:     port.ReservationTimeTo,
		}
		if seen[key] {
			logger.LogInfo("Skipping duplicate port GSW query (same portID and time range)",
				"port_id", port.PortID,
				"usage_type", port.UsageType,
				"from", port.ReservationTimeFrom,
				"to", port.ReservationTimeTo)
			continue
		}
		seen[key] = true
		dedupedPorts = append(dedupedPorts, port)
	}

	g, gctx := errgroup.WithContext(ctx)
	var mu sync.Mutex

	for _, port := range dedupedPorts {
		port := port
		g.Go(func() error {
			amount, err := s.fetchSingleResourcePrice(
				gctx,
				port.PortID,
				port.ReservationTimeFrom,
				port.ReservationTimeTo,
				20,
				"port",
			)
			if err != nil {
				logger.LogError("Failed to fetch port price from GSW", "port_id", port.PortID, "error", err)

				return fmt.Errorf("port_id=%s: %w", port.PortID, err)
			}

			mu.Lock()
			result.PortPrices[port.PortID] = amount
			mu.Unlock()

			logger.LogInfo("Successfully fetched port price from GSW", "port_id", port.PortID, "amount", amount)
			return nil
		})
	}

	return g.Wait()
}

func (s *BillingCalculationService) fetchSingleResourcePrice(
	ctx context.Context,
	resourceID string,
	startAt time.Time,
	endAt time.Time,
	resourceType int,
	resourceName string,
) (int32, error) {
	var req model.ResourcePriceListRequest
	if !startAt.IsZero() {
		req.EffectiveStartTime = value.NewNullTimeFromTime(startAt)
	} else {
		req.EffectiveStartTime = value.NewEmptyNullTime()
	}
	if !endAt.IsZero() {
		req.EffectiveEndTime = value.NewNullTimeFromTime(endAt)
	} else {
		req.EffectiveEndTime = value.NewEmptyNullTime()
	}
	req.ResourceIDs = []string{resourceID}
	req.ResourceType = &resourceType

	priceReq := req

	var priceResp *model.ResourcePriceList
	op := func(rc context.Context) error {
		var err error

		priceResp, err = s.gswPriceGW.GetResourcePriceList(rc, "", priceReq)
		return err
	}

	if err := retry.WithBackoff(ctx, op, retry.DefaultConfig()); err != nil {
		return 0, fmt.Errorf("failed to fetch %s price: %w", resourceName, err)
	}

	amount, err := s.calculatePriceFromResponse(priceResp, startAt, endAt)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate %s price: %w", resourceName, err)
	}

	return amount, nil
}

func (s *BillingCalculationService) calculatePriceFromResponse(resp *model.ResourcePriceList, startAt time.Time, endAt time.Time) (int32, error) {
	if resp == nil || len(resp.Resources) == 0 {
		return 0, nil
	}

	totalAmount := int32(0)

	for _, resource := range resp.Resources {
		totalAmount += CalculateResourcePriceFromRules(resource.PriceInfos, startAt, endAt)
	}

	return totalAmount, nil
}

func (s *BillingCalculationService) calculateDurationMinutes(startAt time.Time, endAt time.Time) (int32, error) {
	minutes := util.SafeIntToInt32(int(endAt.Sub(startAt).Minutes()))

	if minutes < 0 {
		return 0, fmt.Errorf("invalid time range: endAt (%s) is before startAt (%s)",
			endAt.Format(time.RFC3339), startAt.Format(time.RFC3339))
	}

	if minutes == 0 {
		return 1, nil
	}

	return minutes, nil
}

func collectPriceBoundaries(
	priceInfos []model.ResourcePriceInfo,
	reservationStart, reservationEnd time.Time,
) []time.Time {
	boundarySet := map[int64]time.Time{
		reservationStart.Unix(): reservationStart,
		reservationEnd.Unix():   reservationEnd,
	}

	for _, pi := range priceInfos {
		if pi.EffectiveStartTime.Valid && pi.EffectiveStartTime.Time != nil {
			t := *pi.EffectiveStartTime.Time

			if t.After(reservationStart) && t.Before(reservationEnd) {
				boundarySet[t.Unix()] = t
			}
		}
		if pi.EffectiveEndTime.Valid && pi.EffectiveEndTime.Time != nil {
			t := *pi.EffectiveEndTime.Time
			if t.After(reservationStart) && t.Before(reservationEnd) {
				boundarySet[t.Unix()] = t
			}
		}
	}

	boundaries := make([]time.Time, 0, len(boundarySet))
	for _, t := range boundarySet {
		boundaries = append(boundaries, t)
	}
	sort.Slice(boundaries, func(i, j int) bool {
		return boundaries[i].Before(boundaries[j])
	})

	return boundaries
}

func selectBestPriceRule(priceInfos []model.ResourcePriceInfo, at time.Time) *model.ResourcePriceInfo {
	var best *model.ResourcePriceInfo

	for i := range priceInfos {
		pi := &priceInfos[i]

		if pi.EffectiveStartTime.Valid && pi.EffectiveStartTime.Time != nil {
			if at.Before(*pi.EffectiveStartTime.Time) {
				continue
			}
		}
		if pi.EffectiveEndTime.Valid && pi.EffectiveEndTime.Time != nil {

			if !at.Before(*pi.EffectiveEndTime.Time) {
				continue
			}
		}

		if best == nil {
			best = pi
			continue
		}

		if pi.Priority < best.Priority {

			best = pi
		} else if pi.Priority == best.Priority {

			logger.LogError("Multiple price rules with same priority found (data inconsistency)",
				"at", at, "price_type", pi.PriceType, "priority", pi.Priority,
				"current_price_id", pi.PriceID, "best_price_id", best.PriceID)

		}
	}

	return best
}

func priceTypeToUnitSeconds(priceType int) int64 {
	switch priceType {
	case 1:
		return 1
	case 2:
		return 60
	case 3:
		return 3600
	case 4:
		return 86400
	case 5:
		return 604800
	case 6:
		return 2592000
	case 7:
		return 31536000
	default:
		return 60
	}
}

func calculateSegmentAmount(pi *model.ResourcePriceInfo, segmentSeconds int64) int32 {
	if pi.PricePerUnit <= 0 {
		return util.SafeIntToInt32(pi.Price)
	}

	unitSecs := priceTypeToUnitSeconds(pi.PriceType)
	totalUnitSecs := int64(pi.PricePerUnit) * unitSecs
	if totalUnitSecs <= 0 {
		return util.SafeIntToInt32(pi.Price)
	}

	units := (segmentSeconds + totalUnitSecs - 1) / totalUnitSecs
	amount := int64(pi.Price) * units

	return util.SafeIntToInt32(int(amount))
}

func CalculateResourcePriceFromRules(
	priceInfos []model.ResourcePriceInfo,
	reservationStart, reservationEnd time.Time,
) int32 {
	if len(priceInfos) == 0 {
		return 0
	}

	boundaries := collectPriceBoundaries(priceInfos, reservationStart, reservationEnd)
	totalAmount := int32(0)

	for i := 0; i < len(boundaries)-1; i++ {
		segStart := boundaries[i]
		segEnd := boundaries[i+1]

		segSeconds := int64(segEnd.Sub(segStart).Seconds())
		if segSeconds <= 0 {
			continue
		}

		rule := selectBestPriceRule(priceInfos, segStart)
		if rule == nil {
			logger.LogInfo("No applicable price rule for segment",
				"segment_start", segStart, "segment_end", segEnd)
			continue
		}

		amount := calculateSegmentAmount(rule, segSeconds)
		totalAmount += amount

		logger.LogInfo("Resource price segment calculated",
			"segment_start", segStart,
			"segment_end", segEnd,
			"segment_seconds", segSeconds,
			"price_type", rule.PriceType,
			"priority", rule.Priority,
			"price", rule.Price,
			"price_per_unit", rule.PricePerUnit,
			"amount", amount)
	}

	return totalAmount
}

func (s *BillingCalculationService) calculateSectionPrice(section *model.ExternalUaslDefinition, startAt time.Time, endAt time.Time, durationMinutes int32) (int32, error) {

	if len(section.PriceInfo) == 0 {
		return 0, fmt.Errorf("no price rules available for section: %s", section.ExUaslSectionID)
	}

	timeSegments := s.splitTimeByPriceRules(section.PriceInfo, startAt, endAt)

	if len(timeSegments) == 0 {

		logger.LogInfo("No time segments found, using default rule",
			"section_id", section.ExUaslSectionID,
			"start_at", startAt,
			"end_at", endAt)
		defaultRule := section.PriceInfo[0]
		amount := s.calculateAmountFromRule(&defaultRule, durationMinutes)
		return amount, nil
	}

	totalAmount := int32(0)
	for _, segment := range timeSegments {
		segmentMinutes := util.SafeIntToInt32(int(segment.EndTime.Sub(segment.StartTime).Minutes()))
		if segmentMinutes <= 0 {

			segmentMinutes = 1
		}

		amount := s.calculateAmountFromRule(segment.Rule, segmentMinutes)
		totalAmount += amount

		logger.LogInfo("Segment price calculated",
			"section_id", section.ExUaslSectionID,
			"segment_start", segment.StartTime,
			"segment_end", segment.EndTime,
			"segment_minutes", segmentMinutes,
			"rule_start_time", segment.Rule.EffectiveStartTime,
			"rule_end_time", segment.Rule.EffectiveEndTime,
			"price_type", segment.Rule.PriceType,
			"price_per_minute", segment.Rule.Price,
			"segment_amount", amount)
	}

	logger.LogInfo("Total section price calculated",
		"section_id", section.ExUaslSectionID,
		"total_segments", len(timeSegments),
		"total_amount", totalAmount)

	return totalAmount, nil
}

func (s *BillingCalculationService) findMatchingPriceRule(rules model.PriceInfoList, startAt time.Time) *model.PriceInfo {

	layoutDate := "2006-01-02"
	layoutTime := "15:04"

	for i := range rules {
		rule := &rules[i]

		if rule.EffectiveStartDate != "" {
			sd, err := time.ParseInLocation(layoutDate, rule.EffectiveStartDate, startAt.Location())
			if err != nil || startAt.Before(sd) {
				continue
			}
		}
		if rule.EffectiveEndDate != "" {
			ed, err := time.ParseInLocation(layoutDate, rule.EffectiveEndDate, startAt.Location())
			if err != nil {
				continue
			}

			if startAt.After(ed.Add(24 * time.Hour)) {
				continue
			}
		}

		sTimeStr := rule.EffectiveStartTime
		eTimeStr := rule.EffectiveEndTime
		if sTimeStr == "" {
			sTimeStr = "00:00"
		}
		if eTimeStr == "" {
			eTimeStr = "24:00"
		}

		parseHM := func(v string) (t time.Time, is24 bool, err error) {
			if v == "24:00" {
				tt, err := time.ParseInLocation(layoutTime, "00:00", startAt.Location())
				return tt, true, err
			}
			tt, err := time.ParseInLocation(layoutTime, v, startAt.Location())
			return tt, false, err
		}

		sT, s24, err1 := parseHM(sTimeStr)
		eT, e24, err2 := parseHM(eTimeStr)
		if err1 != nil || err2 != nil {
			continue
		}

		startOfDay := time.Date(startAt.Year(), startAt.Month(), startAt.Day(), 0, 0, 0, 0, startAt.Location())
		sDateTime := startOfDay.Add(time.Duration(sT.Hour())*time.Hour + time.Duration(sT.Minute())*time.Minute)
		eDateTime := startOfDay.Add(time.Duration(eT.Hour())*time.Hour + time.Duration(eT.Minute())*time.Minute)
		if e24 {
			eDateTime = eDateTime.Add(24 * time.Hour)
		}
		if s24 {
			sDateTime = sDateTime.Add(24 * time.Hour)
		}

		crossesMidnight := eDateTime.Before(sDateTime) || eDateTime.Equal(sDateTime)

		if crossesMidnight {
			if !(startAt.Equal(sDateTime) || startAt.After(sDateTime) || startAt.Before(eDateTime)) {
				continue
			}
		} else {
			if startAt.Before(sDateTime) || startAt.Equal(eDateTime) || startAt.After(eDateTime) {
				continue
			}
		}

		return rule
	}

	return nil
}

func (s *BillingCalculationService) splitTimeByPriceRules(
	rules model.PriceInfoList,
	reservationStart time.Time,
	reservationEnd time.Time,
) []TimeSegment {
	if len(rules) == 0 {
		return nil
	}

	boundaries := s.extractRuleBoundaries(rules, reservationStart, reservationEnd)

	if len(boundaries) == 0 {

		matchedRule := s.findMatchingPriceRule(rules, reservationStart)
		if matchedRule == nil {
			matchedRule = &rules[0]
		}
		return []TimeSegment{{
			StartTime: reservationStart,
			EndTime:   reservationEnd,
			Rule:      matchedRule,
		}}
	}

	var segments []TimeSegment
	for i := 0; i < len(boundaries)-1; i++ {
		segmentStart := boundaries[i]
		segmentEnd := boundaries[i+1]

		matchedRule := s.findMatchingPriceRule(rules, segmentStart)
		if matchedRule == nil {
			matchedRule = &rules[0]
		}

		segments = append(segments, TimeSegment{
			StartTime: segmentStart,
			EndTime:   segmentEnd,
			Rule:      matchedRule,
		})
	}

	return segments
}

func (s *BillingCalculationService) extractRuleBoundaries(
	rules model.PriceInfoList,
	reservationStart time.Time,
	reservationEnd time.Time,
) []time.Time {
	boundaryMap := make(map[int64]bool)
	loc := reservationStart.Location()

	boundaryMap[reservationStart.Unix()] = true
	boundaryMap[reservationEnd.Unix()] = true

	currentDay := time.Date(
		reservationStart.Year(),
		reservationStart.Month(),
		reservationStart.Day(),
		0, 0, 0, 0, loc,
	)
	endDay := time.Date(
		reservationEnd.Year(),
		reservationEnd.Month(),
		reservationEnd.Day(),
		23, 59, 59, 0, loc,
	).Add(24 * time.Hour)

	for currentDay.Before(endDay) {
		for _, rule := range rules {

			if !s.isRuleApplicableOnDate(&rule, currentDay) {
				continue
			}

			boundaries := s.extractTimeBoundariesForRule(&rule, currentDay, reservationStart, reservationEnd)
			for _, boundary := range boundaries {
				if !boundary.Before(reservationStart) && !boundary.After(reservationEnd) {
					boundaryMap[boundary.Unix()] = true
				}
			}
		}
		currentDay = currentDay.Add(24 * time.Hour)
	}

	var boundaries []time.Time
	for ts := range boundaryMap {
		boundaries = append(boundaries, time.Unix(ts, 0).In(loc))
	}

	for i := 0; i < len(boundaries)-1; i++ {
		for j := i + 1; j < len(boundaries); j++ {
			if boundaries[i].After(boundaries[j]) {
				boundaries[i], boundaries[j] = boundaries[j], boundaries[i]
			}
		}
	}

	return boundaries
}

func (s *BillingCalculationService) isRuleApplicableOnDate(rule *model.PriceInfo, targetDate time.Time) bool {
	layoutDate := "2006-01-02"
	loc := targetDate.Location()

	if rule.EffectiveStartDate != "" {
		sd, err := time.ParseInLocation(layoutDate, rule.EffectiveStartDate, loc)
		if err != nil || targetDate.Before(sd) {
			return false
		}
	}

	if rule.EffectiveEndDate != "" {
		ed, err := time.ParseInLocation(layoutDate, rule.EffectiveEndDate, loc)
		if err != nil {
			return false
		}

		if targetDate.After(ed.Add(24 * time.Hour)) {
			return false
		}
	}

	return true
}

func (s *BillingCalculationService) extractTimeBoundariesForRule(
	rule *model.PriceInfo,
	date time.Time,
	reservationStart time.Time,
	reservationEnd time.Time,
) []time.Time {
	layoutTime := "15:04"
	loc := date.Location()

	var boundaries []time.Time

	sTimeStr := rule.EffectiveStartTime
	if sTimeStr == "" {
		sTimeStr = "00:00"
	}

	eTimeStr := rule.EffectiveEndTime
	if eTimeStr == "" {
		eTimeStr = "24:00"
	}

	var startHour, startMinute int
	if sTimeStr == "24:00" {
		startHour, startMinute = 0, 0
	} else {
		t, err := time.ParseInLocation(layoutTime, sTimeStr, loc)
		if err == nil {
			startHour, startMinute = t.Hour(), t.Minute()
		}
	}

	var endHour, endMinute int
	if eTimeStr == "24:00" {
		endHour, endMinute = 0, 0
	} else {
		t, err := time.ParseInLocation(layoutTime, eTimeStr, loc)
		if err == nil {
			endHour, endMinute = t.Hour(), t.Minute()
		}
	}

	ruleStart := time.Date(date.Year(), date.Month(), date.Day(), startHour, startMinute, 0, 0, loc)
	if sTimeStr == "24:00" {
		ruleStart = ruleStart.Add(24 * time.Hour)
	}

	ruleEnd := time.Date(date.Year(), date.Month(), date.Day(), endHour, endMinute, 0, 0, loc)
	if eTimeStr == "24:00" {
		ruleEnd = ruleEnd.Add(24 * time.Hour)
	}

	if !ruleStart.Before(reservationStart) && !ruleStart.After(reservationEnd) {
		boundaries = append(boundaries, ruleStart)
	}
	if !ruleEnd.Before(reservationStart) && !ruleEnd.After(reservationEnd) {
		boundaries = append(boundaries, ruleEnd)
	}

	return boundaries
}

func (s *BillingCalculationService) calculateAmountFromRule(rule *model.PriceInfo, durationMinutes int32) int32 {
	switch rule.PriceType {
	case "TIME_MINUTE":
		unit := rule.PricePerUnit
		if unit <= 0 {
			unit = 1
		}

		units := (int64(durationMinutes) + int64(unit) - 1) / int64(unit)
		return int32(rule.Price * float64(units))
	default:
		return 0
	}
}

type internalSection struct {
	Child     *model.UaslReservation
	SectionID string
}

type sectionTimeRange struct {
	Start time.Time
	End   time.Time
}

func mergeSectionTimeRanges(ranges []sectionTimeRange) []sectionTimeRange {
	if len(ranges) == 0 {
		return nil
	}
	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Start.Before(ranges[j].Start)
	})

	merged := make([]sectionTimeRange, 0, len(ranges))
	current := ranges[0]
	for i := 1; i < len(ranges); i++ {
		r := ranges[i]
		if r.Start.After(current.End) {
			merged = append(merged, current)
			current = r
			continue
		}
		if r.End.After(current.End) {
			current.End = r.End
		}
	}
	merged = append(merged, current)
	return merged
}

func (s *BillingCalculationService) CalculateInternalSectionPrices(
	ctx context.Context,
	administratorGroups map[string]*AdministratorGroup,
) (map[string]int32, int32, error) {
	internalSections := make([]*internalSection, 0)
	internalSectionIDs := make([]string, 0)

	for _, group := range administratorGroups {
		if group == nil || !group.IsInternal {
			continue
		}

		for _, child := range group.ChildDomains {
			if child.ExUaslSectionID == nil || *child.ExUaslSectionID == "" {
				continue
			}

			sec := &internalSection{
				Child:     child,
				SectionID: *child.ExUaslSectionID,
			}
			internalSections = append(internalSections, sec)
			internalSectionIDs = append(internalSectionIDs, sec.SectionID)
		}
	}

	var internalDefinitions []*model.ExternalUaslDefinition
	var err error
	if len(internalSectionIDs) > 0 {
		internalDefinitions, err = s.externalUaslDefRepo.FindByExSectionIDs(ctx, internalSectionIDs)
		if err != nil {
			logger.LogError("Failed to find internal section definitions", "error", err)
			return nil, 0, fmt.Errorf("find internal section definitions failed: %w", err)
		}
	}

	sectionIDToDefinition := make(map[string]*model.ExternalUaslDefinition, len(internalDefinitions))
	for _, def := range internalDefinitions {
		sectionIDToDefinition[def.ExUaslSectionID] = def
	}

	sectionPrices := make(map[string]int32)
	sectionRanges := make(map[string][]sectionTimeRange)
	now := time.Now()

	for _, sec := range internalSections {
		if sec == nil || sec.Child == nil || sec.SectionID == "" {
			continue
		}
		def, ok := sectionIDToDefinition[sec.SectionID]
		if !ok {
			logger.LogError("Section definition not found", "section_id", sec.SectionID)
			continue
		}

		durationMinutes, err := s.calculateDurationMinutes(sec.Child.StartAt, sec.Child.EndAt)
		if err != nil {
			logger.LogError("Invalid duration",
				"section_id", sec.SectionID,
				"start", sec.Child.StartAt,
				"end", sec.Child.EndAt,
				"error", err)
			return nil, 0, fmt.Errorf("invalid duration for section %s: %w", sec.SectionID, err)
		}
		amount, err := s.calculateSectionPrice(def, sec.Child.StartAt, sec.Child.EndAt, durationMinutes)
		if err != nil {
			logger.LogError("Failed to calculate section price",
				"section_id", sec.SectionID,
				"start_at", sec.Child.StartAt,
				"end_at", sec.Child.EndAt,
				"duration_minutes", durationMinutes,
				"error", err)
			zeroAmt := 0
			sec.Child.Amount = &zeroAmt
			sectionPrices[sec.SectionID] = 0
		} else {
			amountInt := int(amount)
			sec.Child.Amount = &amountInt
			sec.Child.EstimatedAt = &now
			priceVersion := def.PriceVersion
			sec.Child.PricingRuleVersion = &priceVersion
			sectionPrices[sec.SectionID] = amount

			logger.LogInfo("Section price calculated",
				"section_id", sec.SectionID,
				"start_at", sec.Child.StartAt,
				"end_at", sec.Child.EndAt,
				"duration_minutes", durationMinutes,
				"amount", amount)
		}

		sectionRanges[sec.SectionID] = append(sectionRanges[sec.SectionID], sectionTimeRange{
			Start: sec.Child.StartAt,
			End:   sec.Child.EndAt,
		})
	}

	mergedTotal := int32(0)

	for sectionID, ranges := range sectionRanges {
		def, ok := sectionIDToDefinition[sectionID]
		if !ok || def == nil {
			continue
		}
		mergedRanges := mergeSectionTimeRanges(ranges)
		for _, r := range mergedRanges {
			durationMinutes, err := s.calculateDurationMinutes(r.Start, r.End)
			if err != nil {
				logger.LogError("Invalid merged duration",
					"section_id", sectionID,
					"start_at", r.Start,
					"end_at", r.End,
					"error", err)
				continue
			}
			amount, err := s.calculateSectionPrice(def, r.Start, r.End, durationMinutes)
			if err != nil {
				logger.LogError("Failed to calculate merged section price",
					"section_id", sectionID,
					"start_at", r.Start,
					"end_at", r.End,
					"duration_minutes", durationMinutes,
					"error", err)
				continue
			}
			mergedTotal += amount
		}
	}

	logger.LogInfo("Internal UASL sections price calculated",
		"total_sections", len(internalDefinitions),
		"merged_total_amount", mergedTotal)

	return sectionPrices, mergedTotal, nil
}
