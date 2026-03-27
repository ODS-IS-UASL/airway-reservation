package model

import (
	"fmt"
	"time"
)

type ResourceType string

const (
	ResourceTypeVehicle          ResourceType = "VEHICLE"
	ResourceTypePort             ResourceType = "PORT"
	ResourceTypePayload          ResourceType = "PAYLOAD"
	ResourceTypeUasl             ResourceType = "UASL"
	ResourceTypeInterConnectUasl ResourceType = "INTER_CONNECT_UASL"
)

func (rk ResourceType) Validate() error {
	switch rk {
	case ResourceTypeVehicle, ResourceTypePort, ResourceTypeUasl, ResourceTypeInterConnectUasl:
		return nil
	default:
		return fmt.Errorf("invalid resource resourceType: %s", rk)
	}
}

func (rk ResourceType) String() string {
	return string(rk)
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}

func NewTimeRange(start, end time.Time) (TimeRange, error) {
	tr := TimeRange{Start: start, End: end}
	if err := tr.Validate(); err != nil {
		return TimeRange{}, err
	}
	return tr, nil
}

func (tr TimeRange) Validate() error {
	if tr.End.Before(tr.Start) || tr.End.Equal(tr.Start) {
		return fmt.Errorf("end time must be after start time: start=%v, end=%v", tr.Start, tr.End)
	}
	if tr.Start.IsZero() || tr.End.IsZero() {
		return fmt.Errorf("start and end time must not be zero")
	}
	return nil
}

func (tr TimeRange) Duration() time.Duration {
	return tr.End.Sub(tr.Start)
}

func (tr TimeRange) Overlaps(other TimeRange) bool {
	return tr.Start.Before(other.End) && other.Start.Before(tr.End)
}

func (tr TimeRange) Contains(t time.Time) bool {
	return !t.Before(tr.Start) && t.Before(tr.End)
}

type UaslCheckRequest struct {
	ExUaslSectionID string
	TimeRange       TimeRange
}

func NewUaslCheckRequest(exUaslSectionID string, timeRange TimeRange) (UaslCheckRequest, error) {
	request := UaslCheckRequest{
		ExUaslSectionID: exUaslSectionID,
		TimeRange:       timeRange,
	}
	if err := request.Validate(); err != nil {
		return UaslCheckRequest{}, err
	}
	return request, nil
}

func (a UaslCheckRequest) Validate() error {
	if a.ExUaslSectionID == "" {
		return fmt.Errorf("ex_uasl_section_id must not be empty")
	}
	if err := a.TimeRange.Validate(); err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}
	return nil
}

type ReservationHandle struct {
	ID          string
	Type        ResourceType
	ExternalIDs []string
	Reserved    time.Time
	URL         string
}

func NewReservationHandle(id string, resourceType ResourceType, url string) (ReservationHandle, error) {
	if id == "" {
		return ReservationHandle{}, fmt.Errorf("reservation handle ID must not be empty")
	}
	if err := resourceType.Validate(); err != nil {
		return ReservationHandle{}, err
	}
	return ReservationHandle{
		ID:          id,
		Type:        resourceType,
		ExternalIDs: []string{id},
		Reserved:    time.Now(),
		URL:         url,
	}, nil
}

func (rh ReservationHandle) String() string {
	return fmt.Sprintf("%s:%s", rh.Type, rh.ID)
}

type ExternalReservationResult struct {
	AdministratorID string
	URL             string
	Reservations    map[string]string
	ChildDomains    []*UaslReservation
	PortRequests    []PortReservationRequest
	ReservationData *UaslReservationData
	Error           error
}

type AvailabilitySection struct {
	UaslID        string
	UaslSectionID string
}

type AvailabilityItem struct {
	RequestID     string
	OperatorID    string
	FlightPurpose string
	StartAt       time.Time
	EndAt         time.Time
}

type VehicleAvailabilityItem struct {
	RequestID     string
	ReservationID string
	Name          string
	VehicleID     string
	StartAt       time.Time
	EndAt         time.Time
}

type PortAvailabilityItem struct {
	RequestID     string
	ReservationID string
	Name          string
	PortID        string
	StartAt       time.Time
	EndAt         time.Time
}

type ResourceConflictResult struct {
	ConflictType  string
	ConflictedIDs []string
}
