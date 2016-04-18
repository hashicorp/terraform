package fastly

import (
	"fmt"
	"time"
)

// Billing is the top-level representation of a billing response from the Fastly
// API.
type Billing struct {
	InvoiceID string         `mapstructure:"invoice_id"`
	StartTime *time.Time     `mapstructure:"start_time"`
	EndTime   *time.Time     `mapstructure:"end_time"`
	Status    *BillingStatus `mapstructure:"status"`
	Total     *BillingTotal  `mapstructure:"total"`
}

// BillingStatus is a representation of the status of the bill from the Fastly
// API.
type BillingStatus struct {
	InvoiceID string     `mapstructure:"invoice_id"`
	Status    string     `mapstructure:"status"`
	SentAt    *time.Time `mapstructure:"sent_at"`
}

// BillingTotal is a repsentation of the status of the usage for this bill from
// the Fastly API.
type BillingTotal struct {
	PlanName           string          `mapstructure:"plan_name"`
	PlanCode           string          `mapstructure:"plan_code"`
	PlanMinimum        string          `mapstructure:"plan_minimum"`
	Bandwidth          float64         `mapstructure:"bandwidth"`
	BandwidthCost      float64         `mapstructure:"bandwidth_cost"`
	Requests           uint64          `mapstructure:"requests"`
	RequestsCost       float64         `mapstructure:"requests_cost"`
	IncurredCost       float64         `mapstructure:"incurred_cost"`
	Overage            float64         `mapstructure:"overage"`
	Extras             []*BillingExtra `mapstructure:"extras"`
	ExtrasCost         float64         `mapstructure:"extras_cost"`
	CostBeforeDiscount float64         `mapstructure:"cost_before_discount"`
	Discount           float64         `mapstructure:"discount"`
	Cost               float64         `mapstructure:"cost"`
	Terms              string          `mapstructure:"terms"`
}

// BillingExtra is a representation of extras (such as SSL addons) from the
// Fastly API.
type BillingExtra struct {
	Name      string  `mapstructure:"name"`
	Setup     float64 `mapstructure:"setup"`
	Recurring float64 `mapstructure:"recurring"`
}

// GetBillingInput is used as input to the GetBilling function.
type GetBillingInput struct {
	Year  uint16
	Month uint8
}

// GetBilling returns the billing information for the current account.
func (c *Client) GetBilling(i *GetBillingInput) (*Billing, error) {
	if i.Year == 0 {
		return nil, ErrMissingYear
	}

	if i.Month == 0 {
		return nil, ErrMissingMonth
	}

	path := fmt.Sprintf("/billing/year/%d/month/%02d", i.Year, i.Month)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var b *Billing
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}
