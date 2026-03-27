package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	appconv "uasl-reservation/internal/app/uasl_reservation/application/converter"
	"uasl-reservation/internal/app/uasl_reservation/application/usecase"
	"uasl-reservation/internal/app/uasl_reservation/domain/model"
	"uasl-reservation/internal/pkg/myerror"
	"uasl-reservation/internal/pkg/myvalidator/baseValidator"

	"github.com/labstack/echo/v4"
)

type UaslReservationHandler interface {
	ListAdmin(c echo.Context) error
	ListByOperator(c echo.Context) error
	FindByRequestID(c echo.Context) error
	TryHoldCompositeUasl(c echo.Context) error
	GetAvailability(c echo.Context) error
	EstimateUaslReservation(c echo.Context) error
	NotifyReservationCompleted(c echo.Context) error
	SearchByCondition(c echo.Context) error
	Delete(c echo.Context) error
	Cancel(c echo.Context) error
	Confirm(c echo.Context) error
	Rescind(c echo.Context) error
}

type uaslReservationHandler struct {
	uaslReservationUC *usecase.UaslReservationUsecase
}

type statusRequest struct {
	IsInterConnect  bool   `json:"isInterConnect"`
	OperatorID      string `json:"operatorId"`
	AdministratorID string `json:"administratorId"`
}

func NewUaslReservationHandler(uc *usecase.UaslReservationUsecase) UaslReservationHandler {
	return &uaslReservationHandler{
		uaslReservationUC: uc,
	}
}

func (h *uaslReservationHandler) TryHoldCompositeUasl(c echo.Context) error {
	var input restTryHoldRequest
	if err := bindJSONBody(c, &input); err != nil {
		return respondError(c, err)
	}
	req, err := input.toInput()
	if err != nil {
		return respondError(c, echo.NewHTTPError(http.StatusBadRequest, err.Error()))
	}
	res, err := h.uaslReservationUC.TryHoldCompositeUasl(c.Request().Context(), req)
	if err != nil {
		return respondError(c, err)
	}
	if res.ParentUaslReservation == nil {
		requestID := ""
		administratorID := ""
		if req.ParentUaslReservation != nil {
			requestID = req.ParentUaslReservation.RequestID
			administratorID = req.ParentUaslReservation.ExAdministratorID
		}
		return c.JSON(http.StatusConflict, buildConflictResponse(
			requestID,
			res.ConflictType,
			res.ConflictedResourceIds,
			res.ConflictedFlightPlanIDs,
			administratorID,
			requestID,
			"",
		))
	}
	return c.JSON(http.StatusOK, buildReserveCompositeResponse(
		res.ParentUaslReservation,
		res.ChildUaslReservations,
		res.Vehicles,
		res.Ports,
		res.ConflictedFlightPlanIDs,
		req.IsInterConnect,
		true,
	))
}

func (h *uaslReservationHandler) EstimateUaslReservation(c echo.Context) error {
	var input restEstimateRequest
	if err := bindJSONBody(c, &input); err != nil {
		return respondError(c, err)
	}
	req, err := input.toInput()
	if err != nil {
		return respondError(c, echo.NewHTTPError(http.StatusBadRequest, err.Error()))
	}
	res, err := h.uaslReservationUC.EstimateUaslReservation(c.Request().Context(), req)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"totalAmount": res.TotalAmount,
		"estimatedAt": res.EstimatedAt,
	})
}

func (h *uaslReservationHandler) GetAvailability(c echo.Context) error {
	var input restAvailabilityRequest
	if err := bindJSONBody(c, &input); err != nil {
		return respondError(c, err)
	}

	if len(input.RequestIDs) > 0 {
		req, err := input.toSearchInput()
		if err != nil {
			return respondError(c, echo.NewHTTPError(http.StatusBadRequest, err.Error()))
		}
		res, err := h.uaslReservationUC.SearchByCondition(c.Request().Context(), req)
		if err != nil {
			return respondError(c, err)
		}
		return c.JSON(http.StatusOK, buildSearchResponse(res))
	}

	req, err := input.toGetAvailabilityInput()
	if err != nil {
		return respondError(c, echo.NewHTTPError(http.StatusBadRequest, err.Error()))
	}
	res, err := h.uaslReservationUC.GetAvailability(c.Request().Context(), req)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(http.StatusOK, buildAvailabilityResponse(res))
}

func (h *uaslReservationHandler) NotifyReservationCompleted(c echo.Context) error {
	var input restNotifyReservationCompletedRequest
	if err := bindJSONBody(c, &input); err != nil {
		return respondError(c, err)
	}
	req := input.toModelReq()
	res, err := h.uaslReservationUC.NotifyReservationCompleted(c.Request().Context(), req)
	if err != nil {
		return respondError(c, err)
	}
	_ = res
	return c.JSON(http.StatusOK, buildNotifyReservationCompletedResponse(&input))
}

func (h *uaslReservationHandler) SearchByCondition(c echo.Context) error {
	var input restSearchRequest
	if err := bindJSONBody(c, &input); err != nil {
		return respondError(c, err)
	}
	req, err := input.toInput()
	if err != nil {
		return respondError(c, echo.NewHTTPError(http.StatusBadRequest, err.Error()))
	}
	res, err := h.uaslReservationUC.SearchByCondition(c.Request().Context(), req)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(http.StatusOK, buildSearchResponse(res))
}

func (h *uaslReservationHandler) FindByRequestID(c echo.Context) error {
	if err := validatePathUUID(c, "id", c.Param("id")); err != nil {
		return respondError(c, err)
	}
	req := &appconv.FindUaslReservationInput{ID: c.Param("id")}
	res, err := h.uaslReservationUC.FindByRequestID(c.Request().Context(), req)
	if err != nil {
		return respondError(c, err)
	}
	if res.Parent == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "Reservation not found"})
	}
	return c.JSON(http.StatusOK, buildReserveCompositeResponse(
		res.Parent,
		res.Children,
		res.Vehicles,
		res.Ports,
		res.ConflictedFlightPlanIDs,
		false,
		false,
	))
}

func (h *uaslReservationHandler) Delete(c echo.Context) error {
	if err := validatePathUUID(c, "id", c.Param("id")); err != nil {
		return respondError(c, err)
	}
	req := &appconv.DeleteUaslReservationInput{ID: c.Param("id")}
	res, err := h.uaslReservationUC.Delete(c.Request().Context(), req)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(http.StatusOK, buildDeleteResponse(res, c.Param("id")))
}

func (h *uaslReservationHandler) Cancel(c echo.Context) error {
	if err := validatePathUUID(c, "id", c.Param("id")); err != nil {
		return respondError(c, err)
	}
	input, err := bindStatusRequest(c)
	if err != nil {
		return respondError(c, err)
	}
	req := &appconv.CancelUaslReservationInput{
		ID:             c.Param("id"),
		Status:         "CANCELED",
		IsInterConnect: input.IsInterConnect,
	}
	res, err := h.uaslReservationUC.Cancel(c.Request().Context(), req)
	if err != nil {
		return respondError(c, err)
	}
	response := buildReserveCompositeResponse(
		res.ParentUaslReservation,
		res.ChildUaslReservations,
		res.Vehicles,
		res.Ports,
		res.ConflictedFlightPlanIDs,
		input.IsInterConnect,
		false,
	)
	if !input.IsInterConnect {
		runPostReservationSideEffects(
			func(notifyReq *model.NotifyReservationCompletedRequest) (*model.NotifyReservationCompletedResponse, error) {
				return h.uaslReservationUC.NotifyReservationCompleted(c.Request().Context(), notifyReq)
			},
			response,
			res,
		)
	}
	return c.JSON(http.StatusOK, response)
}

func (h *uaslReservationHandler) Rescind(c echo.Context) error {
	if err := validatePathUUID(c, "id", c.Param("id")); err != nil {
		return respondError(c, err)
	}
	input, err := bindStatusRequest(c)
	if err != nil {
		return respondError(c, err)
	}
	req := &appconv.CancelUaslReservationInput{
		ID:             c.Param("id"),
		Status:         "RESCINDED",
		IsInterConnect: input.IsInterConnect,
	}
	res, err := h.uaslReservationUC.Cancel(c.Request().Context(), req)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(http.StatusOK, buildReserveCompositeResponse(
		res.ParentUaslReservation,
		res.ChildUaslReservations,
		res.Vehicles,
		res.Ports,
		res.ConflictedFlightPlanIDs,
		input.IsInterConnect,
		false,
	))
}

func (h *uaslReservationHandler) Confirm(c echo.Context) error {
	if err := validatePathUUID(c, "id", c.Param("id")); err != nil {
		return respondError(c, err)
	}
	input, err := bindStatusRequest(c)
	if err != nil {
		return respondError(c, err)
	}
	conflictAdministratorID := input.OperatorID
	if conflictAdministratorID == "" {
		conflictAdministratorID = input.AdministratorID
	}
	req := &appconv.ConfirmUaslReservationInput{
		ID:             c.Param("id"),
		Status:         "RESERVED",
		IsInterConnect: input.IsInterConnect,
	}
	res, err := h.uaslReservationUC.ConfirmCompositeUasl(c.Request().Context(), req)
	if err != nil {
		return respondError(c, err)
	}
	if res.ParentUaslReservation == nil {
		return c.JSON(http.StatusConflict, buildConflictResponse(
			c.Param("id"),
			res.ConflictType,
			res.ConflictedResourceIds,
			res.ConflictedFlightPlanIDs,
			conflictAdministratorID,
			c.Param("id"),
			"",
		))
	}
	response := buildReserveCompositeResponse(
		res.ParentUaslReservation,
		res.ChildUaslReservations,
		res.Vehicles,
		res.Ports,
		res.ConflictedFlightPlanIDs,
		input.IsInterConnect,
		false,
	)
	if !input.IsInterConnect {
		runPostReservationSideEffects(
			func(notifyReq *model.NotifyReservationCompletedRequest) (*model.NotifyReservationCompletedResponse, error) {
				return h.uaslReservationUC.NotifyReservationCompleted(c.Request().Context(), notifyReq)
			},
			response,
			res,
		)
	}
	return c.JSON(http.StatusOK, response)
}

func (h *uaslReservationHandler) ListAdmin(c echo.Context) error {
	req := &appconv.ListAdminInput{
		Page: parsePage(c.QueryParam("page")),
	}
	res, err := h.uaslReservationUC.ListAdmin(c.Request().Context(), req)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(http.StatusOK, buildListResponse(res))
}

func (h *uaslReservationHandler) ListByOperator(c echo.Context) error {
	if err := validatePathUUID(c, "operatorId", c.Param("operatorId")); err != nil {
		return respondError(c, err)
	}
	req := &appconv.ListByOperatorInput{
		OperatorID: c.Param("operatorId"),
		Page:       parsePage(c.QueryParam("page")),
	}
	res, err := h.uaslReservationUC.ListByOperator(c.Request().Context(), req)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(http.StatusOK, buildListResponse(res))
}

func bindStatusRequest(c echo.Context) (*statusRequest, error) {
	req := &statusRequest{}
	if err := bindJSONBody(c, req); err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) && httpErr.Code == http.StatusBadRequest {
			return nil, err
		}
		return nil, err
	}
	return req, nil
}

func bindJSONBody(c echo.Context, dest any) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func respondError(c echo.Context, err error) error {
	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		return c.JSON(httpErr.Code, map[string]string{"message": fmt.Sprint(httpErr.Message)})
	}
	statusCode, message := myerror.ConvertToHTTPError(err)
	return c.JSON(statusCode, map[string]string{"message": message})
}

func parsePage(page string) int32 {
	if page == "" {
		return 1
	}
	pageNum, err := strconv.Atoi(page)
	if err != nil || pageNum <= 0 {
		return 1
	}
	return int32(pageNum)
}

func validatePathUUID(c echo.Context, paramName string, id string) error {
	type validStruct struct {
		ID string `validate:"required,model-id"`
	}
	validate, err := baseValidator.New()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create validator")
	}
	if err := validate.Struct(validStruct{ID: id}); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid %s format: %s", paramName, id))
	}
	return nil
}
