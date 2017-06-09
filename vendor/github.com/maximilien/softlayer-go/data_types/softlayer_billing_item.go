package data_types

import (
	"time"
)

type SoftLayer_Billing_Item struct {
	Id                    int        `json:"id"`
	AllowCancellationFlag int        `json:"allowCancellationFlag,omitempty"`
	CancellationDate      *time.Time `json:"cancellationDate,omitempty"`
	CategoryCode          string     `json:"categoryCode,omitempty"`
	CycleStartDate        *time.Time `json:"cycleStartDate,omitempty"`
	CreateDate            *time.Time `json:"createDate,omitempty"`
	Description           string     `json:"description,omitempty"`
	LaborFee              string     `json:"laborFee,omitempty"`
	LaborFeeTaxRate       string     `json:"laborFeeTaxRate,omitempty"`
	LastBillDate          *time.Time `json:"lastBillDate,omitempty"`
	ModifyDate            *time.Time `json:"modifyDate,omitempty"`
	NextBillDate          *time.Time `json:"nextBillDate,omitempty"`
	OneTimeFee            string     `json:"oneTimeFee,omitempty"`
	OneTimeFeeTaxRate     string     `json:"oneTimeFeeTaxRate,omitempty"`
	OrderItemId           int        `json:"orderItemId,omitempty"`
	ParentId              int        `json:"parentId,omitempty"`
	RecurringFee          string     `json:"recurringFee,omitempty"`
	RecurringFeeTaxRate   string     `json:"recurringFeeTaxRate,omitempty"`
	RecurringMonths       int        `json:"recurringMonths,omitempty"`
	ServiceProviderId     int        `json:"serviceProviderId,omitempty"`
	SetupFee              string     `json:"setupFee,omitempty"`
	SetupFeeTaxRate       string     `json:"setupFeeTaxRate,omitempty"`
}
