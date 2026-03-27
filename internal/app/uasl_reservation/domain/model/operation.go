package model

import "uasl-reservation/internal/pkg/value"

type Operation struct {
	ID value.ModelID `json:"id"`
}

func (Operation) TableName() string {
	return "uasl_reservation.operations"
}
