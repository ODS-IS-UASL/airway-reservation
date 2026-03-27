package main

import (
	"context"
	"fmt"
	"os"

	"uasl-reservation/internal/app/uasl_reservation/batch"
	"uasl-reservation/internal/app/uasl_reservation/infrastructure/gateway"
	"uasl-reservation/internal/app/uasl_reservation/infrastructure/repository"
	"uasl-reservation/internal/pkg/config"
	database "uasl-reservation/internal/pkg/database/postgres"
	"uasl-reservation/internal/pkg/logger"
)

func main() {
	db := database.NewDB()
	defer database.CloseDB(db)
	db.Exec("SET search_path TO uasl_reservation")

	logger.InitLogger()
	config.Load()
	logger.SetLogLevelByConfig()

	ctx := context.Background()

	h := batch.NewMonthlySettlementJob(
		repository.NewUaslReservationRepository(db),
		repository.NewUaslSettlementRepository(db),
		gateway.NewOuranosL3AuthGateway(),
		gateway.NewPaymentGateway(),
	)

	if err := h.Run(ctx); err != nil {
		logger.Errorf("月次精算バッチエラー", err)
		fmt.Fprintf(os.Stderr, "monthly settlement batch failed: %v\n", err)
		os.Exit(1)
	}
}
