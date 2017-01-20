package circonus

import (
	"fmt"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
)

// The _Check type is the backing store of the `circonus_check` resource.

type _Check struct {
	api.CheckBundle
}

type _CheckType string

const (
	// CheckBundle.Status can be one of these values
	_CheckStatusActive   = "active"
	_CheckStatusDisabled = "disabled"
)

const (
	_APICheckTypeCAQL       _CheckType = "caql"
	_APICheckTypeICMPPing   _CheckType = "ping_icmp"
	_APICheckTypeHTTP       _CheckType = "http"
	_APICheckTypeJSON       _CheckType = "json"
	_APICheckTypePostgreSQL _CheckType = "postgres"
	_APICheckTypeTCP        _CheckType = "tcp"
)

func _NewCheck() _Check {
	return _Check{
		CheckBundle: *api.NewCheckBundle(),
	}
}

func _LoadCheck(ctxt *_ProviderContext, cid api.CIDType) (_Check, error) {
	var c _Check
	cb, err := ctxt.client.FetchCheckBundle(cid)
	if err != nil {
		return _Check{}, err
	}
	c.CheckBundle = *cb

	return c, nil
}

func _CheckAPIStatusToBool(s string) bool {
	var active bool
	switch s {
	case _CheckStatusActive:
		active = true
	case _CheckStatusDisabled:
		active = false
	default:
		panic(fmt.Sprintf("PROVIDER BUG: check status %q unsupported", s))
	}

	return active
}

func _CheckActiveToAPIStatus(active bool) string {
	switch active {
	case true:
		return _CheckStatusActive
	case false:
		return _CheckStatusDisabled
	}

	panic("suppress Go error message")
}

func (c *_Check) Create(ctxt *_ProviderContext) error {
	cb, err := ctxt.client.CreateCheckBundle(&c.CheckBundle)
	if err != nil {
		return err
	}

	c.CID = cb.CID

	return nil
}

func (c *_Check) Update(ctxt *_ProviderContext) error {
	_, err := ctxt.client.UpdateCheckBundle(&c.CheckBundle)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to update check bundle %s: {{err}}", c.CID), err)
	}

	return nil
}

func (c *_Check) Fixup() error {
	switch _APICheckType(c.Type) {
	case _APICheckTypeCloudWatchAttr:
		switch c.Period {
		case 60:
			c.Config[config.Granularity] = "1"
		case 300:
			c.Config[config.Granularity] = "5"
		}
	}

	return nil
}

func (c *_Check) Validate() error {
	if c.Timeout > float32(c.Period) {
		return fmt.Errorf("Timeout (%f) can not exceed period (%d)", c.Timeout, c.Period)
	}

	switch _APICheckType(c.Type) {
	case _APICheckTypeCloudWatchAttr:
		if !(c.Period == 60 || c.Period == 300) {
			return fmt.Errorf("Period must be either 1m or 5m for a %s check", _APICheckTypeCloudWatchAttr)
		}
	}

	return nil
}
