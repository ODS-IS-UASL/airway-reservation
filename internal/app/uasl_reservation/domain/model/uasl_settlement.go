package model

import (
	"database/sql/driver"
	"fmt"
	"time"
	"uasl-reservation/internal/pkg/value"

	"github.com/lib/pq"
)

type UaslSettlement struct {
	ID                 value.ModelID `json:"id" gorm:"column:id;primaryKey;type:uuid;default:uuid_generate_v4()"`
	ExAdministratorID  string        `json:"ex_administrator_id" gorm:"column:ex_administrator_id;type:varchar(255);not null"`
	OperatorID         value.ModelID `json:"operator_id" gorm:"column:operator_id;type:uuid;not null"`
	TargetYearMonth    time.Time     `json:"target_year_month" gorm:"column:target_year_month;type:date;not null"`
	UaslReservationIDs UUIDArray     `json:"uasl_reservation_ids" gorm:"column:uasl_reservation_ids;type:uuid[]"`
	TotalAmount        int           `json:"total_amount" gorm:"column:total_amount;type:integer;not null"`
	TaxRate            float64       `json:"tax_rate" gorm:"column:tax_rate;type:numeric(5,4);not null"`
	PaymentConfirmedAt *time.Time    `json:"payment_confirmed_at" gorm:"column:payment_confirmed_at"`
	SubmittedAt        *time.Time    `json:"submitted_at" gorm:"column:submitted_at"`
	BilledAt           *time.Time    `json:"billed_at" gorm:"column:billed_at"`
	PaymentDueAt       *time.Time    `json:"payment_due_at" gorm:"column:payment_due_at"`
	PaidAt             *time.Time    `json:"paid_at" gorm:"column:paid_at"`
	CreatedAt          time.Time     `json:"created_at" gorm:"column:created_at;not null;default:now()"`
	UpdatedAt          time.Time     `json:"updated_at" gorm:"column:updated_at;not null;default:now()"`
}

func (s *UaslSettlement) TableName() string {
	return "uasl_reservation.uasl_settlements"
}

type UUIDArray []string

func (a *UUIDArray) Scan(src interface{}) error {
	if src == nil {
		*a = UUIDArray{}
		return nil
	}
	strArray := pq.StringArray{}
	if err := strArray.Scan(src); err != nil {
		return fmt.Errorf("failed to scan uuid array: %w", err)
	}
	*a = UUIDArray(strArray)
	return nil
}

func (a UUIDArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return nil, nil
	}
	return pq.StringArray(a).Value()
}
