package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"uasl-reservation/internal/app/uasl_reservation/application/usecase"
	"uasl-reservation/internal/app/uasl_reservation/handler"
	"uasl-reservation/internal/app/uasl_reservation/infrastructure/gateway"
	"uasl-reservation/internal/app/uasl_reservation/infrastructure/repository"
	"uasl-reservation/internal/pkg/config"
	database "uasl-reservation/internal/pkg/database/postgres"
	"uasl-reservation/internal/pkg/logger"

	"github.com/labstack/echo/v4"
)

type contextKey string

const authorizationKey contextKey = "authorization"

func main() {
	db := database.NewDB()
	db.Exec("SET search_path TO uasl_reservation")

	enableSeed := os.Getenv("ENABLE_SEED") == "true"
	if err := database.RunMigrations(db, enableSeed); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	logger.InitLogger()
	config.Load()
	logger.SetLogLevelByConfig()
	conf := config.GetConfig()

	uaslReservationUsecase := usecase.NewUaslReservationUsecase(
		context.Background(),
		nil,
		repository.NewUaslReservationRepository(db),
		repository.NewExternalResourceReservationRepository(db),
		repository.NewOperationRepository(db),
		gateway.NewVehicleOpenAPIGateway(),
		gateway.NewPortOpenAPIGateway(),
		gateway.NewFlightPlanInfoOpenAPIGateway(),
		repository.NewExternalUaslResourceRepository(db),
		gateway.NewOuranosDiscoveryGateway(),
		gateway.NewResourcePriceGateway(),
		gateway.NewOuranosProxyGateway(),
		gateway.NewConformityAssessmentGateway(),
		repository.NewExternalUaslDefinitionRepository(db),
		repository.NewUaslAdministratorRepository(db),
		gateway.NewUaslDesignOpenAPIGateway(),
		gateway.NewPaymentGateway(),
	)
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader != "" {
				token := authHeader
				if strings.HasPrefix(strings.ToLower(token), "bearer ") {
					token = token[7:]
				}
				ctx := context.WithValue(c.Request().Context(), authorizationKey, token)
				c.SetRequest(c.Request().WithContext(ctx))
			}
			return next(c)
		}
	})
	uaslReservationHandler := handler.NewUaslReservationHandler(uaslReservationUsecase)
	handler.NewHealthCheckRouter(e)
	handler.NewUaslReservationRouter(uaslReservationHandler, e)

	httpAddress := fmt.Sprintf("0.0.0.0:%d", conf.ServerConfig.RESTPort)
	httpServer := &http.Server{
		Addr:    httpAddress,
		Handler: e,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Infof("Start REST server at %s", httpAddress)
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("Received signal, exiting gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.LogError("Failed to shutdown REST server", "error", err.Error())
		}
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Cannot start server", err)
		}
	}
}
