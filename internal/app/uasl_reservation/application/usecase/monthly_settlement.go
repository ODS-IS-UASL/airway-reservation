package usecase

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/app/uasl_reservation/domain/repositoryIF"
	"uasl-reservation/internal/pkg/logger"

	"google.golang.org/grpc/metadata"
)

type MonthlySettlementUsecase struct {
	uaslReservationRepoIF repositoryIF.UaslReservationRepositoryIF
	uaslSettlementRepoIF  repositoryIF.UaslSettlementRepositoryIF
	l3AuthGateway         gatewayIF.OuranosL3AuthGatewayIF
	paymentGateway        gatewayIF.PaymentGatewayIF
}

func NewMonthlySettlementUsecase(
	uaslReservationRepoIF repositoryIF.UaslReservationRepositoryIF,
	uaslSettlementRepoIF repositoryIF.UaslSettlementRepositoryIF,
	l3AuthGateway gatewayIF.OuranosL3AuthGatewayIF,
	paymentGateway gatewayIF.PaymentGatewayIF,
) *MonthlySettlementUsecase {
	return &MonthlySettlementUsecase{
		uaslReservationRepoIF: uaslReservationRepoIF,
		uaslSettlementRepoIF:  uaslSettlementRepoIF,
		l3AuthGateway:         l3AuthGateway,
		paymentGateway:        paymentGateway,
	}
}

type monthlySettlementGroupKey struct {
	ExAdministratorID string
	ReservedByID      string
}

func (u *MonthlySettlementUsecase) Run(ctx context.Context) error {
	logger.Infof("月次精算処理開始")

	targetYearMonth, err := u.aggregate(ctx)
	if err != nil {
		return err
	}

	unsubmitted, err := u.uaslSettlementRepoIF.FindUnsubmittedByTargetYearMonth(targetYearMonth)
	if err != nil {
		logger.Errorf("月次精算エラー: 未提出settlement取得失敗", err)
		return fmt.Errorf("failed to find unsubmitted settlements: %w", err)
	}

	if len(unsubmitted) == 0 {
		logger.Infof("月次精算処理完了: 処理対象なし")
		return nil
	}

	var failCount int
	for _, s := range unsubmitted {
		sid := s.ID.ToString()
		if err := u.processSettlement(ctx, sid); err != nil {
			logger.Errorf("月次精算エラー: settlement処理失敗", err, "settlement_id", sid)
			failCount++
			continue
		}
	}

	logger.Infof("月次精算処理完了: total=%d, success=%d, fail=%d",
		len(unsubmitted), len(unsubmitted)-failCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("monthly settlement completed with %d failures out of %d", failCount, len(unsubmitted))
	}

	return nil
}

func (u *MonthlySettlementUsecase) aggregate(ctx context.Context) (targetYearMonth time.Time, err error) {
	now := time.Now().UTC()
	targetYearMonth = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC)
	yearMonthStr := targetYearMonth.Format("2006-01")

	logger.Infof("月次精算集計開始: target_year_month=%s", yearMonthStr)

	reservations, err := u.uaslReservationRepoIF.FindReservedByMonth(targetYearMonth)
	if err != nil {
		logger.Errorf("月次精算エラー: 予約取得失敗", err)
		return targetYearMonth, fmt.Errorf("failed to fetch reservations: %w", err)
	}

	if len(reservations) == 0 {
		logger.Infof("月次精算集計完了: target_year_month=%s, groups=0 (対象予約なし)", yearMonthStr)
		return targetYearMonth, nil
	}

	groups := make(map[monthlySettlementGroupKey]*model.UaslSettlement)

	type reqGroup struct {
		parents  []*model.UaslReservation
		children []*model.UaslReservation
	}
	reqGroups := make(map[string]*reqGroup)
	for _, r := range reservations {
		reqID := r.RequestID.ToString()
		g, ok := reqGroups[reqID]
		if !ok {
			g = &reqGroup{}
			reqGroups[reqID] = g
		}
		if r.ParentUaslReservationID == nil {
			g.parents = append(g.parents, r)
		} else {
			g.children = append(g.children, r)
		}
	}

	originParentByRequest := make(map[string]*model.UaslReservation)
	for reqID, g := range reqGroups {
		if g == nil || len(g.parents) == 0 {
			continue
		}
		parentByID := make(map[string]*model.UaslReservation, len(g.parents))
		for _, p := range g.parents {
			parentByID[p.ID.ToString()] = p
		}

		var origin *model.UaslReservation
		for _, c := range g.children {
			if c.Sequence == nil || *c.Sequence != 1 || c.ParentUaslReservationID == nil {
				continue
			}
			if p := parentByID[c.ParentUaslReservationID.ToString()]; p != nil {
				origin = p
				break
			}
		}
		if origin == nil {
			origin = g.parents[0]
			for _, p := range g.parents[1:] {
				if p.StartAt.Before(origin.StartAt) {
					origin = p
				}
			}
		}
		originParentByRequest[reqID] = origin
	}

	for _, r := range reservations {
		if r.ExAdministratorID == nil || r.ExReservedBy == nil {
			continue
		}

		key := monthlySettlementGroupKey{
			ExAdministratorID: *r.ExAdministratorID,
			ReservedByID:      r.ExReservedBy.ToString(),
		}

		settlement, exists := groups[key]
		if !exists {
			settlement = &model.UaslSettlement{
				ExAdministratorID:  *r.ExAdministratorID,
				OperatorID:         *r.ExReservedBy,
				TargetYearMonth:    targetYearMonth,
				UaslReservationIDs: model.UUIDArray{},
				TotalAmount:        0,
				TaxRate:            0.1,
			}
			groups[key] = settlement
		}

		settlement.UaslReservationIDs = append(settlement.UaslReservationIDs, r.ID.ToString())
		if r.ParentUaslReservationID == nil && r.Amount != nil {
			origin := originParentByRequest[r.RequestID.ToString()]
			if origin != nil && origin.ID.ToString() == r.ID.ToString() {
				settlement.TotalAmount += *r.Amount
			}
		}
	}

	settlements := make([]*model.UaslSettlement, 0, len(groups))
	for _, s := range groups {
		settlements = append(settlements, s)
	}

	if err := u.uaslSettlementRepoIF.UpsertBatch(settlements); err != nil {
		logger.Errorf("月次精算エラー: 精算レコードUPSERT失敗", err)
		return targetYearMonth, fmt.Errorf("failed to upsert settlements: %w", err)
	}

	logger.Infof("月次精算集計完了: target_year_month=%s, groups=%d", yearMonthStr, len(groups))

	return targetYearMonth, nil
}

func (u *MonthlySettlementUsecase) processSettlement(ctx context.Context, settlementID string) error {
	logger.Infof("月次精算ワーカー開始: settlement_id=%s", settlementID)

	settlement, err := u.uaslSettlementRepoIF.FindByID(settlementID)
	if err != nil {
		logger.Errorf("月次精算エラー: settlement取得失敗", err)
		return fmt.Errorf("failed to find settlement %s: %w", settlementID, err)
	}
	if settlement == nil {
		logger.Infof("月次精算エラー: settlement not found: %s", settlementID)
		return fmt.Errorf("settlement not found: %s", settlementID)
	}

	if settlement.SubmittedAt != nil {
		logger.Infof("月次精算ワーカー完了: settlement_id=%s, status=already_submitted", settlementID)
		return nil
	}

	if settlement.PaymentConfirmedAt != nil {
		logger.Infof("月次精算ワーカー: 決済済み・submitted_at未更新を検出, settlement_id=%s", settlementID)
		if err := u.uaslSettlementRepoIF.UpdateSubmittedAt(settlementID, time.Now().UTC()); err != nil {
			logger.Errorf("月次精算エラー: submitted_at更新失敗", err)
			return fmt.Errorf("failed to update submitted_at: %w", err)
		}
		logger.Infof("月次精算ワーカー完了: settlement_id=%s, status=submitted", settlementID)
		return nil
	}

	accessToken, err := u.l3AuthGateway.GetAccessToken(ctx)
	if err != nil {
		logger.Errorf("月次精算エラー: L3認証失敗", err)
		return fmt.Errorf("failed to get L3 access token: %w", err)
	}
	if err := u.l3AuthGateway.IntrospectToken(ctx, accessToken); err != nil {
		logger.Errorf("月次精算エラー: L3トークンイントロスペクション失敗", err)
		return fmt.Errorf("failed to introspect L3 access token: %w", err)
	}

	md := metadata.Pairs("authorization", "Bearer "+accessToken)
	ctx = metadata.NewIncomingContext(ctx, md)

	paymentServiceID := os.Getenv("PAYMENT_SERVICE_ID")
	if paymentServiceID == "" {
		paymentServiceID = "169f5226-4644-4d1e-ac36-14999e73601f"
	}

	confirmReq := model.TransactionConfirmRequest{
		ProviderID:        settlement.ExAdministratorID,
		ConsumerID:        settlement.OperatorID.ToString(),
		PaymentServiceID:  paymentServiceID,
		TaxClassification: "taxable",
		TaxRate:           settlement.TaxRate,
		CompletedAt:       time.Now(),
		Amount:            settlement.TotalAmount,
	}

	logger.Infof("金額確定API呼び出し: settlement_id=%s", settlementID)

	confirmResp, err := u.paymentGateway.ConfirmTransaction(ctx, "", confirmReq)
	if err != nil {
		logger.Errorf("月次精算エラー: 金額確定失敗", err)
		return fmt.Errorf("failed to confirm transaction: %w", err)
	}
	if confirmResp == nil {
		return fmt.Errorf("payment confirm response is nil")
	}
	if strings.ToLower(confirmResp.Status) != "success" {
		return fmt.Errorf("payment confirm not completed: status=%s, detail=%s", confirmResp.Status, confirmResp.Detail)
	}

	now := time.Now().UTC()
	if err := u.uaslSettlementRepoIF.UpdatePaymentConfirmedAt(settlementID, now); err != nil {
		logger.Errorf("月次精算エラー: payment_confirmed_at更新失敗", err)
		return fmt.Errorf("failed to update payment_confirmed_at: %w", err)
	}

	if err := u.uaslSettlementRepoIF.UpdateSubmittedAt(settlementID, now); err != nil {
		logger.Errorf("月次精算エラー: submitted_at更新失敗", err)
		return fmt.Errorf("failed to update submitted_at: %w", err)
	}

	logger.Infof("月次精算ワーカー完了: settlement_id=%s, status=submitted", settlementID)
	return nil
}
