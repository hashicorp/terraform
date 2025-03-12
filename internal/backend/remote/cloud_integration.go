// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package remote

import (
	"context"
	"log"
	"time"
)

// IntegrationContext is a set of data that is useful when performing HCP Terraform integration operations
type IntegrationContext struct {
	StopContext   context.Context
	CancelContext context.Context
}

func (s *IntegrationContext) Poll(backoffMinInterval float64, backoffMaxInterval float64, every func(i int) (bool, error)) error {
	for i := 0; ; i++ {
		select {
		case <-s.StopContext.Done():
			log.Print("IntegrationContext.Poll: StopContext.Done() called")
			return s.StopContext.Err()
		case <-s.CancelContext.Done():
			log.Print("IntegrationContext.Poll: CancelContext.Done() called")
			return s.CancelContext.Err()
		case <-time.After(backoff(backoffMinInterval, backoffMaxInterval, i)):
			// blocks for a time between min and max
		}

		cont, err := every(i)
		if !cont {
			return err
		}
	}
}
