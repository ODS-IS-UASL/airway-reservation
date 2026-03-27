package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/repositoryIF"
	"uasl-reservation/internal/pkg/logger"
	"uasl-reservation/internal/pkg/value"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type uaslReservationRepository struct {
	db *gorm.DB
}

func NewUaslReservationRepository(db *gorm.DB) repositoryIF.UaslReservationRepositoryIF {
	return &uaslReservationRepository{db}
}

func (uaslReservationRepo *uaslReservationRepository) FindReservedByMonth(yearMonth time.Time) ([]*model.UaslReservation, error) {
	startOfMonth := time.Date(yearMonth.Year(), yearMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)

	var reservations []*model.UaslReservation
	if err := uaslReservationRepo.db.
		Where(`(status = ? AND parent_uasl_reservation_id IS NULL AND start_at >= ? AND start_at < ?)
			OR (sequence = 1 AND parent_uasl_reservation_id IN (
				SELECT id FROM uasl_reservation.uasl_reservations
				WHERE status = ? AND parent_uasl_reservation_id IS NULL AND start_at >= ? AND start_at < ?
			))`,
			"RESERVED", startOfMonth, startOfNextMonth,
			"RESERVED", startOfMonth, startOfNextMonth).
		Find(&reservations).Error; err != nil {
		return nil, err
	}
	return reservations, nil
}

func (uaslReservationRepo *uaslReservationRepository) FetchAll(uaslReservations *[]model.UaslReservation) error {
	if err := uaslReservationRepo.db.Find(&uaslReservations).Error; err != nil {
		return err
	}
	return nil
}
func (uaslReservationRepo *uaslReservationRepository) FindByID(uaslReservationID value.ModelID) (*model.UaslReservation, error) {
	var uaslReservation model.UaslReservation
	if err := uaslReservationRepo.db.First(&uaslReservation, "id = ?", uaslReservationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &uaslReservation, nil
}

func (uaslReservationRepo *uaslReservationRepository) FindByRequestID(requestID value.ModelID) (*model.UaslReservation, []*model.UaslReservation, error) {

	var reservations []model.UaslReservation
	if err := uaslReservationRepo.db.
		Where("request_id = ?", requestID).
		Order("sequence ASC NULLS FIRST, start_at ASC").
		Find(&reservations).Error; err != nil {
		return nil, nil, err
	}

	if len(reservations) == 0 {
		return nil, nil, nil
	}

	var parents []*model.UaslReservation
	var children []*model.UaslReservation

	for i := range reservations {
		if reservations[i].ParentUaslReservationID == nil {
			parents = append(parents, &reservations[i])
		} else {
			children = append(children, &reservations[i])
		}
	}

	if len(parents) == 0 {
		return nil, nil, fmt.Errorf("parent reservation not found for request_id: %s", requestID)
	}

	var parent *model.UaslReservation

	var minSeq *int
	var parentID *value.ModelID
	for _, child := range children {
		if child.ParentUaslReservationID == nil {
			continue
		}
		if child.Sequence != nil {
			if minSeq == nil || *child.Sequence < *minSeq {
				seq := *child.Sequence
				minSeq = &seq
				parentID = child.ParentUaslReservationID
			}
		} else if parentID == nil {
			parentID = child.ParentUaslReservationID
		}
	}
	if parentID != nil {
		for _, p := range parents {
			if p.ID.ToString() == parentID.ToString() {
				parent = p
				break
			}
		}
	}

	if parent == nil {
		parent = parents[0]
	}

	var preferredSectionID string
	var minSectionSeq *int
	sectionIDs := make([]string, 0, len(children)+1)
	sectionIDSet := make(map[string]struct{})
	for _, child := range children {
		if child.ExUaslSectionID == nil || *child.ExUaslSectionID == "" {
			continue
		}
		sid := *child.ExUaslSectionID
		if child.Sequence != nil {
			if minSectionSeq == nil || *child.Sequence < *minSectionSeq {
				seq := *child.Sequence
				minSectionSeq = &seq
				preferredSectionID = sid
			}
		} else if preferredSectionID == "" {
			preferredSectionID = sid
		}
		if _, exists := sectionIDSet[sid]; !exists {
			sectionIDSet[sid] = struct{}{}
			sectionIDs = append(sectionIDs, sid)
		}
	}
	if preferredSectionID == "" && parent.ExUaslSectionID != nil && *parent.ExUaslSectionID != "" {
		preferredSectionID = *parent.ExUaslSectionID
		if _, exists := sectionIDSet[preferredSectionID]; !exists {
			sectionIDSet[preferredSectionID] = struct{}{}
			sectionIDs = append(sectionIDs, preferredSectionID)
		}
	}

	flightPurpose := ""
	if len(sectionIDs) > 0 {
		var uaslDefs []model.ExternalUaslDefinition
		if err := uaslReservationRepo.db.
			Where("ex_uasl_section_id IN ?", sectionIDs).
			Find(&uaslDefs).Error; err != nil {

			logger.LogError("FindByRequestID: failed to fetch flight purpose", "error", err)
		} else if len(uaslDefs) == 0 {
			logger.LogInfo("FindByRequestID: external uasl definition not found",
				"request_id", requestID.ToString(),
				"ex_uasl_section_ids", sectionIDs)
		} else {

			selected := uaslDefs[0]
			if preferredSectionID != "" {
				for _, def := range uaslDefs {
					if def.ExUaslSectionID == preferredSectionID {
						selected = def
						break
					}
				}
			}

			if selected.FlightPurpose.Valid && selected.FlightPurpose.String != nil && *selected.FlightPurpose.String != "" {
				flightPurpose = *selected.FlightPurpose.String
			} else {

				for _, def := range uaslDefs {
					if def.FlightPurpose.Valid && def.FlightPurpose.String != nil && *def.FlightPurpose.String != "" {
						flightPurpose = *def.FlightPurpose.String
						break
					}
				}
			}
		}
	}

	parent.FlightPurpose = flightPurpose

	return parent, children, nil
}

func (uaslReservationRepo *uaslReservationRepository) CheckConflictReservations(
	queries []model.UaslCheckRequest,
) ([]model.UaslReservation, error) {
	if len(queries) == 0 {
		return []model.UaslReservation{}, nil
	}

	var reservations []model.UaslReservation

	statuses := []string{"RESERVED"}

	db := uaslReservationRepo.db.
		Model(&model.UaslReservation{}).
		Select("child.*").
		Table("? AS child", gorm.Expr((&model.UaslReservation{}).TableName())).
		Joins("LEFT JOIN ? AS parent ON child.parent_uasl_reservation_id = parent.id", gorm.Expr((&model.UaslReservation{}).TableName())).
		Joins("LEFT JOIN ? AS def ON child.ex_uasl_section_id = def.ex_uasl_section_id", gorm.Expr((&model.ExternalUaslDefinition{}).TableName()))

	parentStatusCondition := "((child.parent_uasl_reservation_id IS NULL AND child.status IN ?) OR (child.parent_uasl_reservation_id IS NOT NULL AND parent.status IN ?))"

	var conflictConditions []string
	var conflictArgs []interface{}

	for _, query := range queries {

		sectionCondition := "(child.ex_uasl_section_id = ? AND child.start_at < ? AND child.end_at > ?)"
		conflictConditions = append(conflictConditions, sectionCondition)
		conflictArgs = append(conflictArgs, query.ExUaslSectionID, query.TimeRange.End, query.TimeRange.Start)

		pointCondition := `(
			def.point_ids && (
				SELECT point_ids
				FROM ?
				WHERE ex_uasl_section_id = ?
			)
			AND child.start_at < ?
			AND child.end_at > ?
		)`
		conflictConditions = append(conflictConditions, pointCondition)
		conflictArgs = append(conflictArgs,
			gorm.Expr((&model.ExternalUaslDefinition{}).TableName()),
			query.ExUaslSectionID,
			query.TimeRange.End,
			query.TimeRange.Start,
		)
	}

	if len(conflictConditions) == 0 {
		return []model.UaslReservation{}, nil
	}

	conflictConditionStr := "(" + strings.Join(conflictConditions, " OR ") + ")"

	finalCondition := parentStatusCondition + " AND " + conflictConditionStr
	finalArgs := append([]interface{}{statuses, statuses}, conflictArgs...)

	db = db.Where(finalCondition, finalArgs...)

	err := db.Find(&reservations).Error
	if err != nil {
		return nil, err
	}

	return reservations, nil
}

func (uaslReservationRepo *uaslReservationRepository) FindByUaslSectionIDs(
	uaslSectionIDs []string,
	baseAt *value.NullTime,
) ([]model.UaslReservation, error) {
	if len(uaslSectionIDs) == 0 {
		return []model.UaslReservation{}, nil
	}

	var reservations []model.UaslReservation

	statuses := []string{"RESERVED"}

	db := uaslReservationRepo.db.
		Model(&model.UaslReservation{}).
		Select("DISTINCT child.*").
		Table("? AS child", gorm.Expr((&model.UaslReservation{}).TableName())).
		Joins("LEFT JOIN ? AS parent ON child.parent_uasl_reservation_id = parent.id", gorm.Expr((&model.UaslReservation{}).TableName())).
		Joins("LEFT JOIN ? AS def ON child.ex_uasl_section_id = def.ex_uasl_section_id", gorm.Expr((&model.ExternalUaslDefinition{}).TableName()))

	parentStatusCondition := "((child.parent_uasl_reservation_id IS NULL AND child.status IN ?) OR (child.parent_uasl_reservation_id IS NOT NULL AND parent.status IN ?))"

	pointOverlapCondition := `(
		child.ex_uasl_section_id IN ?
		OR def.point_ids && (
			SELECT COALESCE(array_agg(DISTINCT pt), '{}')
			FROM ?, unnest(point_ids) AS pt
			WHERE ex_uasl_section_id IN ?
		)
	)`

	db = db.Where(parentStatusCondition, statuses, statuses).
		Where(pointOverlapCondition,
			uaslSectionIDs,
			gorm.Expr((&model.ExternalUaslDefinition{}).TableName()),
			uaslSectionIDs,
		)

	if baseAt != nil && baseAt.Valid && baseAt.Time != nil {
		startRange := baseAt.Time.AddDate(0, 0, -30)
		endRange := baseAt.Time.AddDate(0, 0, 30)

		db = db.Where("child.start_at < ? AND child.end_at > ?", endRange, startRange)
	}

	err := db.Find(&reservations).Error
	if err != nil {
		return nil, err
	}

	requestGroups := make(map[string][]model.UaslReservation)
	for _, res := range reservations {
		rid := res.RequestID.ToString()
		requestGroups[rid] = append(requestGroups[rid], res)
	}

	seq1SectionIDs := make([]string, 0)
	requestToSectionMap := make(map[string]string)
	for rid, group := range requestGroups {
		for _, res := range group {
			if res.Sequence != nil && *res.Sequence == 1 && res.ExUaslSectionID != nil && *res.ExUaslSectionID != "" {
				seq1SectionIDs = append(seq1SectionIDs, *res.ExUaslSectionID)
				requestToSectionMap[rid] = *res.ExUaslSectionID
				break
			}
		}
	}

	flightPurposeMap := make(map[string]string)
	if len(seq1SectionIDs) > 0 {
		var uaslDefs []model.ExternalUaslDefinition
		if err := uaslReservationRepo.db.
			Where("ex_uasl_section_id IN ?", seq1SectionIDs).
			Find(&uaslDefs).Error; err != nil {

			logger.LogError("FindByUaslSectionIDs: failed to fetch flight purposes", "error", err)
		} else {
			for _, def := range uaslDefs {
				if def.FlightPurpose.Valid && def.FlightPurpose.String != nil {
					flightPurposeMap[def.ExUaslSectionID] = *def.FlightPurpose.String
				}
			}
		}
	}

	for i := range reservations {
		rid := reservations[i].RequestID.ToString()
		if sectionID, ok := requestToSectionMap[rid]; ok {
			if fp, ok := flightPurposeMap[sectionID]; ok {
				reservations[i].FlightPurpose = fp
			}
		}
	}

	return reservations, nil
}

func (uaslReservationRepo *uaslReservationRepository) InsertOne(uaslReservation *model.UaslReservation) (*model.UaslReservation, error) {
	uaslReservation.ParentUaslReservationID = toNullModelID(uaslReservation.ParentUaslReservationID)
	uaslReservation.ExReservedBy = toNullModelID(uaslReservation.ExReservedBy)
	uaslReservation.OrganizationID = toNullModelID(uaslReservation.OrganizationID)
	uaslReservation.ProjectID = toNullModelID(uaslReservation.ProjectID)
	uaslReservation.OperationID = toNullModelID(uaslReservation.OperationID)
	logger.LogInfo("InsertOne: inserting UaslReservation", "id", uaslReservation.ID)

	if err := uaslReservationRepo.db.Create(uaslReservation).Error; err != nil {
		logger.LogError("InsertOne: failed to insert UaslReservation", "error", err)

		if pqErr, ok := err.(*pq.Error); ok {
			logger.LogError("InsertOne: PostgreSQL error detail", "code", pqErr.Code, "detail", pqErr.Detail, "hint", pqErr.Hint)
		}

		return uaslReservation, err
	}

	return uaslReservation, nil
}

func (uaslReservationRepo *uaslReservationRepository) InsertBatch(uaslReservations []*model.UaslReservation) ([]*model.UaslReservation, error) {
	if len(uaslReservations) == 0 {
		return []*model.UaslReservation{}, nil
	}

	idMap := make(map[string]int)

	for i, ar := range uaslReservations {

		if ar.ID == "" || ar.ID.ToString() == "" {
			ar.ID = value.ModelID(uuid.New().String())
			logger.LogInfo("Generated UUID for reservation",
				"id", ar.ID.ToString(),
				"index", i,
				"is_parent", ar.Sequence == nil,
				"sequence", func() interface{} {
					if ar.Sequence != nil {
						return *ar.Sequence
					}
					return nil
				}())
		}

		idStr := ar.ID.ToString()
		if count, exists := idMap[idStr]; exists {
			logger.LogError("Duplicate ID detected in batch",
				"id", idStr,
				"first_index", count,
				"duplicate_index", i,
				"is_parent", ar.Sequence == nil)
			idMap[idStr]++
		} else {
			idMap[idStr] = i
		}

		if ar.Sequence == nil {
			ar.ParentUaslReservationID = nil
		} else {
			ar.ParentUaslReservationID = toNullModelID(ar.ParentUaslReservationID)
		}
		ar.ExReservedBy = toNullModelID(ar.ExReservedBy)
		ar.OrganizationID = toNullModelID(ar.OrganizationID)
		ar.ProjectID = toNullModelID(ar.ProjectID)
		ar.OperationID = toNullModelID(ar.OperationID)
	}

	logger.LogInfo("Batch insert summary",
		"total_count", len(uaslReservations),
		"unique_ids", len(idMap),
		"parents_count", func() int {
			count := 0
			for _, ar := range uaslReservations {
				if ar.Sequence == nil {
					count++
				}
			}
			return count
		}(),
		"children_count", func() int {
			count := 0
			for _, ar := range uaslReservations {
				if ar.Sequence != nil {
					count++
				}
			}
			return count
		}())

	if err := uaslReservationRepo.db.Create(&uaslReservations).Error; err != nil {
		logger.LogError("Batch insert failed", "error", err)
		return nil, err
	}
	return uaslReservations, nil
}

func (uaslReservationRepo *uaslReservationRepository) UpdateOne(uaslReservation *model.UaslReservation) (*model.UaslReservation, error) {
	updateData := make(map[string]interface{})

	if !uaslReservation.UpdatedAt.IsZero() {
		updateData["updated_at"] = uaslReservation.UpdatedAt
	}

	if uaslReservation.Status != "" {
		updateData["status"] = uaslReservation.Status
	}

	if uaslReservation.AcceptedAt != nil {
		updateData["accepted_at"] = uaslReservation.AcceptedAt
	}

	if uaslReservation.FixedAt != nil {
		updateData["fixed_at"] = uaslReservation.FixedAt
	}

	var updatedUaslReservation model.UaslReservation
	result := uaslReservationRepo.db.Model(&model.UaslReservation{}).
		Clauses(clause.Returning{}).
		Where("id = ?", uaslReservation.ID).
		Updates(updateData).
		Scan(&updatedUaslReservation)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected < 1 {
		return nil, fmt.Errorf("object does not exist")
	}
	return &updatedUaslReservation, nil
}

func (uaslReservationRepo *uaslReservationRepository) UpdateBatch(uaslReservations []*model.UaslReservation) ([]*model.UaslReservation, error) {
	if len(uaslReservations) == 0 {
		return []*model.UaslReservation{}, nil
	}

	tx := uaslReservationRepo.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	updatedReservations := make([]*model.UaslReservation, 0, len(uaslReservations))

	for _, ar := range uaslReservations {
		updateData := make(map[string]interface{})
		updateData["updated_at"] = ar.UpdatedAt

		if ar.Status != "" {
			updateData["status"] = ar.Status
		}

		updateData["accepted_at"] = ar.AcceptedAt
		updateData["fixed_at"] = ar.FixedAt

		var updated model.UaslReservation
		result := tx.Model(&model.UaslReservation{}).
			Where("id = ?", ar.ID).
			Clauses(clause.Returning{}).
			Updates(updateData).
			Scan(&updated)

		if result.Error != nil {
			tx.Rollback()
			return nil, result.Error
		}
		if result.RowsAffected < 1 {
			tx.Rollback()
			return nil, fmt.Errorf("reservation with id %s does not exist", ar.ID)
		}

		updatedReservations = append(updatedReservations, &updated)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return updatedReservations, nil
}

func (uaslReservationRepo *uaslReservationRepository) ExpirePendingParentsAndChildren(
	expiredBeforeUTC time.Time,
	limit int,
) ([]model.ExpiredUaslReservation, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
WITH expired_parents AS (
    SELECT id, request_id
    FROM uasl_reservation.uasl_reservations
    WHERE status = 'PENDING'
      AND parent_uasl_reservation_id IS NULL
      AND created_at < ?
    ORDER BY created_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT ?
),
deleted AS (
    DELETE FROM uasl_reservation.uasl_reservations ur
    FROM expired_parents ep
    WHERE ur.request_id = ep.request_id
      AND ur.status IN ('PENDING', 'INHERITED')
    RETURNING ur.request_id
)
SELECT DISTINCT ep.id::text AS parent_id, ep.request_id::text AS request_id
FROM expired_parents ep
JOIN deleted d ON d.request_id = ep.request_id
`

	type expiredRow struct {
		ParentID  string `gorm:"column:parent_id"`
		RequestID string `gorm:"column:request_id"`
	}
	rows := make([]expiredRow, 0)
	if err := uaslReservationRepo.db.Raw(query, expiredBeforeUTC, limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	results := make([]model.ExpiredUaslReservation, 0, len(rows))
	for _, row := range rows {
		results = append(results, model.ExpiredUaslReservation{
			ParentID:  value.ModelID(row.ParentID),
			RequestID: value.ModelID(row.RequestID),
		})
	}
	return results, nil
}

func (uaslReservationRepo *uaslReservationRepository) ExpirePendingByRequestID(
	requestID value.ModelID,
	expiredBeforeUTC time.Time,
) (*model.ExpiredUaslReservation, error) {
	query := `
WITH target_parent AS (
    SELECT id, request_id
    FROM uasl_reservation.uasl_reservations
    WHERE request_id = ?
      AND parent_uasl_reservation_id IS NULL
      AND status = 'PENDING'
      AND created_at < ?
    FOR UPDATE SKIP LOCKED
    LIMIT 1
),
deleted AS (
    DELETE FROM uasl_reservation.uasl_reservations ur
    USING target_parent tp
    WHERE ur.request_id = tp.request_id
      AND ur.status IN ('PENDING', 'INHERITED')
    RETURNING ur.request_id
)
SELECT tp.id::text AS parent_id, tp.request_id::text AS request_id
FROM target_parent tp
WHERE EXISTS (SELECT 1 FROM deleted d WHERE d.request_id = tp.request_id)
LIMIT 1
`

	type expiredRow struct {
		ParentID  string `gorm:"column:parent_id"`
		RequestID string `gorm:"column:request_id"`
	}
	var row expiredRow
	if err := uaslReservationRepo.db.Raw(query, requestID, expiredBeforeUTC).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ParentID == "" || row.RequestID == "" {
		return nil, nil
	}
	return &model.ExpiredUaslReservation{
		ParentID:  value.ModelID(row.ParentID),
		RequestID: value.ModelID(row.RequestID),
	}, nil
}

func (uaslReservationRepo *uaslReservationRepository) DeleteOne(uaslReservationID value.ModelID) (value.ModelID, error) {
	result := uaslReservationRepo.db.Where("id = ?", uaslReservationID).Delete(&model.UaslReservation{})
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected < 1 {
		return "", fmt.Errorf("object does not exist")
	}
	return uaslReservationID, nil
}

func (uaslReservationRepo *uaslReservationRepository) DeleteByRequestID(requestID value.ModelID) (int, error) {
	result := uaslReservationRepo.db.Where("request_id = ?", requestID.ToString()).Delete(&model.UaslReservation{})
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

func toNullModelID(id *value.ModelID) *value.ModelID {
	if id == nil || id.ToString() == "" {
		return nil
	}
	return id
}

func (uaslReservationRepo *uaslReservationRepository) ListByRequestIDs(requestIDs []value.ModelID) ([]*model.UaslReservationListItem, error) {
	if len(requestIDs) == 0 {
		return []*model.UaslReservationListItem{}, nil
	}

	var allReservations []model.UaslReservation
	if err := uaslReservationRepo.db.
		Where("request_id IN ?", requestIDs).
		Order("sequence ASC NULLS FIRST, start_at ASC").
		Find(&allReservations).Error; err != nil {
		return nil, err
	}

	type parentChildren struct {
		parent   *model.UaslReservation
		children []*model.UaslReservation
	}
	pcMap := make(map[string]*parentChildren, len(requestIDs))
	orderedRIDs := make([]string, 0, len(requestIDs))
	for i := range allReservations {
		rid := allReservations[i].RequestID.ToString()
		pc, ok := pcMap[rid]
		if !ok {
			pc = &parentChildren{}
			pcMap[rid] = pc
			orderedRIDs = append(orderedRIDs, rid)
		}
		if allReservations[i].ParentUaslReservationID == nil {
			pc.parent = &allReservations[i]
		} else {
			pc.children = append(pc.children, &allReservations[i])
		}
	}

	var externalResources []model.ExternalResourceReservation
	if err := uaslReservationRepo.db.
		Table("? AS err", gorm.Expr((&model.ExternalResourceReservation{}).TableName())).
		Select(`err.*,
			(SELECT name
			 FROM ?
			 WHERE ex_resource_id = err.ex_resource_id
			 AND resource_type::text = err.resource_type::text
			 LIMIT 1) as resource_name`, gorm.Expr((&model.ExternalUaslResource{}).TableName())).
		Where("err.request_id IN ?", requestIDs).
		Scan(&externalResources).Error; err != nil {

		logger.LogError("ListByRequestIDs: failed to fetch external resources", "error", err)
	}

	extResMap := make(map[string][]model.ExternalResourceReservation)
	for _, res := range externalResources {
		rid := res.RequestID.ToString()
		extResMap[rid] = append(extResMap[rid], res)
	}

	seq1SectionIDs := make([]string, 0, len(pcMap))
	sectionIDMap := make(map[string]string)
	for _, rid := range orderedRIDs {
		pc := pcMap[rid]
		if pc.parent == nil {
			continue
		}
		sectionID := ""
		for _, child := range pc.children {
			if child.Sequence != nil && *child.Sequence == 1 && child.ExUaslSectionID != nil && *child.ExUaslSectionID != "" {
				sectionID = *child.ExUaslSectionID
				break
			}
		}
		if sectionID == "" && pc.parent.ExUaslSectionID != nil && *pc.parent.ExUaslSectionID != "" {
			sectionID = *pc.parent.ExUaslSectionID
		}
		if sectionID != "" {
			seq1SectionIDs = append(seq1SectionIDs, sectionID)
			sectionIDMap[rid] = sectionID
		}
	}

	flightPurposeMap := make(map[string]string)
	if len(seq1SectionIDs) > 0 {
		var uaslDefs []model.ExternalUaslDefinition
		if err := uaslReservationRepo.db.
			Where("ex_uasl_section_id IN ?", seq1SectionIDs).
			Find(&uaslDefs).Error; err != nil {

			logger.LogError("ListByRequestIDs: failed to fetch flight purposes", "error", err)
		} else {
			for _, def := range uaslDefs {
				if def.FlightPurpose.Valid && def.FlightPurpose.String != nil {
					flightPurposeMap[def.ExUaslSectionID] = *def.FlightPurpose.String
				}
			}
		}
	}

	items := make([]*model.UaslReservationListItem, 0, len(pcMap))
	for _, rid := range orderedRIDs {
		pc := pcMap[rid]
		if pc.parent == nil {
			continue
		}
		items = append(items, &model.UaslReservationListItem{
			Parent:            pc.parent,
			Children:          pc.children,
			ExternalResources: extResMap[rid],
			FlightPurpose:     flightPurposeMap[sectionIDMap[rid]],
		})
	}

	return items, nil
}

func (uaslReservationRepo *uaslReservationRepository) ListByOperator(
	ctx context.Context,
	operatorID string,
	exAdministratorIDs []string,
	page int32,
	perPage int32,
) ([]*model.UaslReservationListItem, int64, error) {
	var total int64
	excludedStatuses := []value.ReservationStatus{
		value.RESERVATION_STATUS_PENDING,
	}

	p := int(page)
	pp := int(perPage)

	if p <= 0 {
		p = 1
	}
	if pp <= 0 {
		pp = 20
	}
	offset := (p - 1) * pp

	if err := uaslReservationRepo.db.
		Model(&model.UaslReservation{}).
		Select("request_id").
		Where("ex_reserved_by = ? AND parent_uasl_reservation_id IS NULL AND ex_administrator_id IN ? AND status NOT IN ?", operatorID, exAdministratorIDs, excludedStatuses).
		Group("request_id").
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	type requestRow struct {
		RequestID value.ModelID `gorm:"column:request_id"`
		UpdatedAt time.Time     `gorm:"column:updated_at"`
	}
	var requestRows []requestRow
	if err := uaslReservationRepo.db.
		Model(&model.UaslReservation{}).
		Select("request_id, MAX(updated_at) AS updated_at").
		Where("ex_reserved_by = ? AND parent_uasl_reservation_id IS NULL AND ex_administrator_id IN ? AND status NOT IN ?", operatorID, exAdministratorIDs, excludedStatuses).
		Group("request_id").
		Order("MAX(updated_at) DESC").
		Limit(pp).
		Offset(offset).
		Scan(&requestRows).Error; err != nil {
		return nil, 0, err
	}

	if len(requestRows) == 0 {
		return []*model.UaslReservationListItem{}, total, nil
	}

	requestIDs := make([]value.ModelID, 0, len(requestRows))
	for _, row := range requestRows {
		requestIDs = append(requestIDs, row.RequestID)
	}

	var parents []*model.UaslReservation
	if err := uaslReservationRepo.db.
		Where("parent_uasl_reservation_id IS NULL AND request_id IN ? AND ex_administrator_id IN ?", requestIDs, exAdministratorIDs).
		Order("updated_at DESC").
		Find(&parents).Error; err != nil {
		return nil, 0, err
	}

	var children []model.UaslReservation
	if err := uaslReservationRepo.db.
		Where("request_id IN ? AND parent_uasl_reservation_id IS NOT NULL", requestIDs).
		Order("start_at ASC").
		Find(&children).Error; err != nil {
		return nil, 0, err
	}

	var externalResources []model.ExternalResourceReservation

	query := uaslReservationRepo.db.
		Table("? AS err", gorm.Expr((&model.ExternalResourceReservation{}).TableName())).
		Select(`err.*,
			(SELECT name
			 FROM ?
			 WHERE ex_resource_id = err.ex_resource_id
			 AND resource_type::text = err.resource_type::text
			 LIMIT 1) as resource_name`, gorm.Expr((&model.ExternalUaslResource{}).TableName())).
		Where("err.request_id IN ?", requestIDs)

	err := query.Scan(&externalResources).Error
	if err != nil {
		logger.LogError("Failed to fetch external resources", "error", err)
		return nil, 0, err
	}

	childrenMap := make(map[string][]*model.UaslReservation)
	for i := range children {
		rid := children[i].RequestID.ToString()
		childrenMap[rid] = append(childrenMap[rid], &children[i])
	}

	parentsMap := make(map[string][]*model.UaslReservation)
	for _, p := range parents {
		parentsMap[p.RequestID.ToString()] = append(parentsMap[p.RequestID.ToString()], p)
	}

	externalResourcesMap := make(map[string][]model.ExternalResourceReservation)
	for _, res := range externalResources {
		requestIDStr := res.RequestID.ToString()
		externalResourcesMap[requestIDStr] = append(externalResourcesMap[requestIDStr], res)
	}

	sectionIDMap := make(map[string]string)
	sectionIDs := make([]string, 0)
	sectionIDSet := make(map[string]struct{})
	for rid, childList := range childrenMap {
		var selectedID string
		var minSeq *int
		for _, child := range childList {
			if child.ExUaslSectionID == nil || *child.ExUaslSectionID == "" {
				continue
			}
			if child.Sequence != nil {
				if minSeq == nil || *child.Sequence < *minSeq {
					seq := *child.Sequence
					minSeq = &seq
					selectedID = *child.ExUaslSectionID
				}
			} else if selectedID == "" {
				selectedID = *child.ExUaslSectionID
			}
		}
		if selectedID != "" {
			sectionIDMap[rid] = selectedID
			if _, exists := sectionIDSet[selectedID]; !exists {
				sectionIDSet[selectedID] = struct{}{}
				sectionIDs = append(sectionIDs, selectedID)
			}
		}
	}

	var uaslDefs []model.ExternalUaslDefinition
	flightPurposeMap := make(map[string]string)
	if len(sectionIDs) > 0 {
		if err := uaslReservationRepo.db.
			Where("ex_uasl_section_id IN ?", sectionIDs).
			Find(&uaslDefs).Error; err != nil {

			logger.LogError("Failed to fetch flight purposes", "error", err)
		} else {
			for _, def := range uaslDefs {
				if def.FlightPurpose.Valid && def.FlightPurpose.String != nil {
					flightPurposeMap[def.ExUaslSectionID] = *def.FlightPurpose.String
				}
			}
		}
	}

	items := make([]*model.UaslReservationListItem, 0, len(requestRows))
	for _, row := range requestRows {
		rid := row.RequestID.ToString()
		parentList := parentsMap[rid]
		if len(parentList) == 0 {
			continue
		}

		var parent *model.UaslReservation
		childList := childrenMap[rid]
		var minSeq *int
		var parentID *value.ModelID
		for _, child := range childList {
			if child.ParentUaslReservationID == nil {
				continue
			}
			if child.Sequence != nil {
				if minSeq == nil || *child.Sequence < *minSeq {
					seq := *child.Sequence
					minSeq = &seq
					parentID = child.ParentUaslReservationID
				}
			} else if parentID == nil {
				parentID = child.ParentUaslReservationID
			}
		}
		if parentID != nil {
			for _, p := range parentList {
				if p.ID.ToString() == parentID.ToString() {
					parent = p
					break
				}
			}
		}

		if parent == nil {
			if len(parentList) == 1 {

				parent = parentList[0]
			} else {

				parentIDSet := make(map[string]*model.UaslReservation, len(parentList))
				for _, p := range parentList {
					parentIDSet[p.ID.ToString()] = p
				}
				for _, child := range childList {
					if child.ParentUaslReservationID == nil {
						continue
					}
					if p, ok := parentIDSet[child.ParentUaslReservationID.ToString()]; ok {
						parent = p
						break
					}
				}

				if parent == nil {
					parent = parentList[0]
					for _, p := range parentList[1:] {
						if p.StartAt.Before(parent.StartAt) {
							parent = p
						}
					}
				}
			}
		}

		flightPurpose := ""
		if sid, ok := sectionIDMap[rid]; ok {
			if fp, ok := flightPurposeMap[sid]; ok {
				flightPurpose = fp
			}
		}

		item := &model.UaslReservationListItem{
			Parent:            parent,
			Children:          childList,
			ExternalResources: externalResourcesMap[rid],
			FlightPurpose:     flightPurpose,
		}
		items = append(items, item)
	}

	return items, total, nil
}

func (uaslReservationRepo *uaslReservationRepository) ListAdmin(
	ctx context.Context,
	page int32,
	perPage int32,
) ([]*model.UaslReservationListItem, int64, error) {

	var total int64
	excludedStatuses := []value.ReservationStatus{
		value.RESERVATION_STATUS_PENDING,
	}

	p := int(page)
	pp := int(perPage)

	if p <= 0 {
		p = 1
	}
	if pp <= 0 {
		pp = 20
	}
	offset := (p - 1) * pp

	if err := uaslReservationRepo.db.
		Model(&model.UaslReservation{}).
		Select("request_id").
		Where("parent_uasl_reservation_id IS NULL AND status NOT IN ?", excludedStatuses).
		Group("request_id").
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	type requestRow struct {
		RequestID value.ModelID `gorm:"column:request_id"`
		UpdatedAt time.Time     `gorm:"column:updated_at"`
	}
	var requestRows []requestRow
	if err := uaslReservationRepo.db.
		Model(&model.UaslReservation{}).
		Select("request_id, MAX(updated_at) AS updated_at").
		Where("parent_uasl_reservation_id IS NULL AND status NOT IN ?", excludedStatuses).
		Group("request_id").
		Order("MAX(updated_at) DESC").
		Limit(pp).
		Offset(offset).
		Scan(&requestRows).Error; err != nil {
		return nil, 0, err
	}

	if len(requestRows) == 0 {
		return []*model.UaslReservationListItem{}, total, nil
	}

	requestIDs := make([]value.ModelID, 0, len(requestRows))
	for _, row := range requestRows {
		requestIDs = append(requestIDs, row.RequestID)
	}

	var parents []*model.UaslReservation
	if err := uaslReservationRepo.db.
		Where("parent_uasl_reservation_id IS NULL AND request_id IN ?", requestIDs).
		Order("updated_at DESC").
		Find(&parents).Error; err != nil {
		return nil, 0, err
	}

	var children []model.UaslReservation
	if err := uaslReservationRepo.db.
		Where("request_id IN ? AND parent_uasl_reservation_id IS NOT NULL", requestIDs).
		Order("start_at ASC").
		Find(&children).Error; err != nil {
		return nil, 0, err
	}

	var externalResources []model.ExternalResourceReservation

	query := uaslReservationRepo.db.
		Table("? AS err", gorm.Expr((&model.ExternalResourceReservation{}).TableName())).
		Select(`err.*,
			(SELECT name
			 FROM ?
			 WHERE ex_resource_id = err.ex_resource_id
			 AND resource_type::text = err.resource_type::text
			 LIMIT 1) as resource_name`, gorm.Expr((&model.ExternalUaslResource{}).TableName())).
		Where("err.request_id IN ?", requestIDs)

	if err := query.Scan(&externalResources).Error; err != nil {
		logger.LogError("Failed to fetch external resources (admin)", "error", err)
		return nil, 0, err
	}

	childrenMap := make(map[string][]*model.UaslReservation)
	for i := range children {
		rid := children[i].RequestID.ToString()
		childrenMap[rid] = append(childrenMap[rid], &children[i])
	}

	externalResourcesMap := make(map[string][]model.ExternalResourceReservation)
	for _, res := range externalResources {
		requestIDStr := res.RequestID.ToString()
		externalResourcesMap[requestIDStr] = append(externalResourcesMap[requestIDStr], res)
	}

	sectionIDMap := make(map[string]string)
	sectionIDs := make([]string, 0)
	sectionIDSet := make(map[string]struct{})
	for rid, childList := range childrenMap {
		var selectedID string
		var minSeq *int
		for _, child := range childList {
			if child.ExUaslSectionID == nil || *child.ExUaslSectionID == "" {
				continue
			}
			if child.Sequence != nil {
				if minSeq == nil || *child.Sequence < *minSeq {
					seq := *child.Sequence
					minSeq = &seq
					selectedID = *child.ExUaslSectionID
				}
			} else if selectedID == "" {
				selectedID = *child.ExUaslSectionID
			}
		}
		if selectedID != "" {
			sectionIDMap[rid] = selectedID
			if _, exists := sectionIDSet[selectedID]; !exists {
				sectionIDSet[selectedID] = struct{}{}
				sectionIDs = append(sectionIDs, selectedID)
			}
		}
	}

	var uaslDefs []model.ExternalUaslDefinition
	flightPurposeMap := make(map[string]string)
	if len(sectionIDs) > 0 {
		if err := uaslReservationRepo.db.
			Where("ex_uasl_section_id IN ?", sectionIDs).
			Find(&uaslDefs).Error; err != nil {

			logger.LogError("Failed to fetch flight purposes", "error", err)
		} else {
			for _, def := range uaslDefs {
				if def.FlightPurpose.Valid && def.FlightPurpose.String != nil {
					flightPurposeMap[def.ExUaslSectionID] = *def.FlightPurpose.String
				}
			}
		}
	}

	parentsMap := make(map[string][]*model.UaslReservation)
	for _, p := range parents {
		parentsMap[p.RequestID.ToString()] = append(parentsMap[p.RequestID.ToString()], p)
	}

	items := make([]*model.UaslReservationListItem, 0, len(requestRows))
	for _, row := range requestRows {
		rid := row.RequestID.ToString()
		parentList := parentsMap[rid]
		if len(parentList) == 0 {
			continue
		}

		var parent *model.UaslReservation
		childList := childrenMap[rid]
		var minSeq *int
		var parentID *value.ModelID
		for _, child := range childList {
			if child.ParentUaslReservationID == nil {
				continue
			}
			if child.Sequence != nil {
				if minSeq == nil || *child.Sequence < *minSeq {
					seq := *child.Sequence
					minSeq = &seq
					parentID = child.ParentUaslReservationID
				}
			} else if parentID == nil {
				parentID = child.ParentUaslReservationID
			}
		}
		if parentID != nil {
			for _, p := range parentList {
				if p.ID.ToString() == parentID.ToString() {
					parent = p
					break
				}
			}
		}

		if parent == nil {
			if len(parentList) == 1 {

				parent = parentList[0]
			} else {

				parentIDSet := make(map[string]*model.UaslReservation, len(parentList))
				for _, p := range parentList {
					parentIDSet[p.ID.ToString()] = p
				}
				for _, child := range childList {
					if child.ParentUaslReservationID == nil {
						continue
					}
					if p, ok := parentIDSet[child.ParentUaslReservationID.ToString()]; ok {
						parent = p
						break
					}
				}

				if parent == nil {
					parent = parentList[0]
					for _, p := range parentList[1:] {
						if p.StartAt.Before(parent.StartAt) {
							parent = p
						}
					}
				}
			}
		}
		flightPurpose := ""
		if sid, ok := sectionIDMap[rid]; ok {
			if fp, ok := flightPurposeMap[sid]; ok {
				flightPurpose = fp
			}
		}

		item := &model.UaslReservationListItem{
			Parent:            parent,
			Children:          childList,
			ExternalResources: externalResourcesMap[rid],
			FlightPurpose:     flightPurpose,
		}
		items = append(items, item)
	}

	return items, total, nil
}

func (uaslReservationRepo *uaslReservationRepository) FindChildrenByParentID(parentID value.ModelID) ([]*model.UaslReservation, error) {
	var children []model.UaslReservation
	if err := uaslReservationRepo.db.
		Where("parent_uasl_reservation_id = ?", parentID).
		Order("sequence ASC").
		Find(&children).Error; err != nil {
		return nil, err
	}

	result := make([]*model.UaslReservation, len(children))
	for i := range children {
		result[i] = &children[i]
	}
	return result, nil
}

func (uaslReservationRepo *uaslReservationRepository) FindParentsByRequestID(requestID value.ModelID) ([]*model.UaslReservation, error) {
	var parents []model.UaslReservation
	if err := uaslReservationRepo.db.
		Where("request_id = ? AND parent_uasl_reservation_id IS NULL", requestID).
		Order("start_at ASC").
		Find(&parents).Error; err != nil {
		return nil, err
	}

	result := make([]*model.UaslReservation, len(parents))
	for i := range parents {
		result[i] = &parents[i]
	}
	return result, nil
}
