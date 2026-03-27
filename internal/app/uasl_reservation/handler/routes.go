package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func NewHealthCheckRouter(e *echo.Echo) *echo.Echo {
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "tm service / path ok")
	})
	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	return e
}

func NewUaslReservationRouter(uaslReservationHH UaslReservationHandler, e *echo.Echo) *echo.Echo {
	admin := e.Group("/v1/admin/uaslReservations")
	uaslReservation := e.Group("/v1/uaslReservations")
	operator := e.Group("/v1/operator")

	admin.GET("", uaslReservationHH.ListAdmin)
	admin.PUT("/:id/rescind", uaslReservationHH.Rescind)

	operator.GET("/:operatorId/uaslReservations", uaslReservationHH.ListByOperator)

	uaslReservation.GET("/:id", uaslReservationHH.FindByRequestID)
	uaslReservation.POST("", uaslReservationHH.TryHoldCompositeUasl)
	uaslReservation.POST("/availability", uaslReservationHH.GetAvailability)
	uaslReservation.POST("/estimate", uaslReservationHH.EstimateUaslReservation)
	uaslReservation.POST("/completed", uaslReservationHH.NotifyReservationCompleted)
	uaslReservation.POST("/search", uaslReservationHH.SearchByCondition)
	uaslReservation.DELETE("/:id", uaslReservationHH.Delete)
	uaslReservation.PUT("/:id/cancel", uaslReservationHH.Cancel)
	uaslReservation.PUT("/:id/confirm", uaslReservationHH.Confirm)

	return e
}
