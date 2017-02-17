package datadog

import (
	"fmt"
	"time"
)

type RateLimit struct {
	Limit     int
	Period    time.Duration
	Remaining int
	Reset     time.Duration
}

func (r *RateLimit) Error() string {
	return fmt.Sprintf("Rate limiting: Limit %d, Period %d, Remaining %d, Reset in %d", r.Limit, r.Period, r.Remaining, r.Reset)
}

func NewRateLimit(limit, period, remaining, reset int) *RateLimit {
	return &RateLimit{
		Limit:     limit,
		Period:    time.Duration(period) * time.Second,
		Remaining: remaining,
		Reset:     time.Duration(reset) * time.Second,
	}
}
