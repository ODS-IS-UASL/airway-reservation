package batch

import (
	"context"

	"uasl-reservation/internal/app/uasl_reservation/application/usecase"
	"uasl-reservation/internal/app/uasl_reservation/domain/gatewayIF"
	"uasl-reservation/internal/app/uasl_reservation/domain/repositoryIF"
)

type MonthlySettlementJob struct {
	usecase *usecase.MonthlySettlementUsecase
}

func NewMonthlySettlementJob(
	uaslReservationRepoIF repositoryIF.UaslReservationRepositoryIF,
	uaslSettlementRepoIF repositoryIF.UaslSettlementRepositoryIF,
	l3AuthGateway gatewayIF.OuranosL3AuthGatewayIF,
	paymentGateway gatewayIF.PaymentGatewayIF,
) *MonthlySettlementJob {
	return &MonthlySettlementJob{
		usecase: usecase.NewMonthlySettlementUsecase(
			uaslReservationRepoIF,
			uaslSettlementRepoIF,
			l3AuthGateway,
			paymentGateway,
		),
	}
}

func (j *MonthlySettlementJob) Run(ctx context.Context) error {
	return j.usecase.Run(ctx)
}
