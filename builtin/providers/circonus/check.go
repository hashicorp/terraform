package circonus

import (
	"fmt"
	"log"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
)

// The circonusCheck type is the backing store of the `circonus_check` resource.

type circonusCheck struct {
	api.CheckBundle
}

type circonusCheckType string

const (
	// CheckBundle.Status can be one of these values
	checkStatusActive   = "active"
	checkStatusDisabled = "disabled"
)

const (
	apiCheckTypeCAQL       circonusCheckType = "caql"
	apiCheckTypeConsul     circonusCheckType = "consul"
	apiCheckTypeICMPPing   circonusCheckType = "ping_icmp"
	apiCheckTypeHTTP       circonusCheckType = "http"
	apiCheckTypeJSON       circonusCheckType = "json"
	apiCheckTypeMySQL      circonusCheckType = "mysql"
	apiCheckTypeStatsd     circonusCheckType = "statsd"
	apiCheckTypePostgreSQL circonusCheckType = "postgres"
	apiCheckTypeTCP        circonusCheckType = "tcp"
)

func newCheck() circonusCheck {
	return circonusCheck{
		CheckBundle: *api.NewCheckBundle(),
	}
}

func loadCheck(ctxt *providerContext, cid api.CIDType) (circonusCheck, error) {
	var c circonusCheck
	cb, err := ctxt.client.FetchCheckBundle(cid)
	if err != nil {
		return circonusCheck{}, err
	}
	c.CheckBundle = *cb

	return c, nil
}

func checkAPIStatusToBool(s string) bool {
	var active bool
	switch s {
	case checkStatusActive:
		active = true
	case checkStatusDisabled:
		active = false
	default:
		log.Printf("[ERROR] PROVIDER BUG: check status %q unsupported", s)
	}

	return active
}

func checkActiveToAPIStatus(active bool) string {
	if active {
		return checkStatusActive
	}

	return checkStatusDisabled
}

func (c *circonusCheck) Create(ctxt *providerContext) error {
	cb, err := ctxt.client.CreateCheckBundle(&c.CheckBundle)
	if err != nil {
		return err
	}

	c.CID = cb.CID

	return nil
}

func (c *circonusCheck) Update(ctxt *providerContext) error {
	_, err := ctxt.client.UpdateCheckBundle(&c.CheckBundle)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to update check bundle %s: {{err}}", c.CID), err)
	}

	return nil
}

func (c *circonusCheck) Fixup() error {
	switch apiCheckType(c.Type) {
	case apiCheckTypeCloudWatchAttr:
		switch c.Period {
		case 60:
			c.Config[config.Granularity] = "1"
		case 300:
			c.Config[config.Granularity] = "5"
		}
	}

	return nil
}

func (c *circonusCheck) Validate() error {
	if len(c.Metrics) == 0 {
		return fmt.Errorf("At least one %s must be specified", checkMetricAttr)
	}

	if c.Timeout > float32(c.Period) {
		return fmt.Errorf("Timeout (%f) can not exceed period (%d)", c.Timeout, c.Period)
	}

	// Check-type specific validation
	switch apiCheckType(c.Type) {
	case apiCheckTypeCloudWatchAttr:
		if !(c.Period == 60 || c.Period == 300) {
			return fmt.Errorf("Period must be either 1m or 5m for a %s check", apiCheckTypeCloudWatchAttr)
		}
	case apiCheckTypeConsulAttr:
		if v, found := c.Config[config.URL]; !found || v == "" {
			return fmt.Errorf("%s must have at least one check mode set: %s, %s, or %s must be set", checkConsulAttr, checkConsulServiceAttr, checkConsulNodeAttr, checkConsulStateAttr)
		}
	}

	return nil
}
