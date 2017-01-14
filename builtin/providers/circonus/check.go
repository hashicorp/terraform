package circonus

import (
	"fmt"

	"github.com/circonus-labs/circonus-gometrics/api"
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
	_CheckTypeJSON _CheckType = "json"
)

func _NewCheck() _Check {
	return _Check{
		CheckBundle: *api.NewCheckBundle(),
	}
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

func _LoadCheck(ctxt *_ProviderContext, cid api.CIDType) (_Check, error) {
	var c _Check
	cb, err := ctxt.client.FetchCheckBundle(cid)
	if err != nil {
		return _Check{}, err
	}
	c.CheckBundle = *cb

	return c, nil
}

func (c *_Check) ParseConfig(ar _AttrReader) error {
	if name, ok := ar.GetStringOk(_CheckNameAttr); ok {
		c.DisplayName = name
	}

	if status, ok := ar.GetBoolOK(_CheckActiveAttr); ok {
		statusString := _CheckStatusActive
		if !status {
			statusString = _CheckStatusDisabled
		}

		c.Status = statusString
	}

	if collectorsList, ok := ar.GetSetAsListOk(_CheckCollectorAttr); ok {
		c.Brokers = collectorsList.CollectList(_CheckCollectorIDAttr)
	}

	if streamList, ok := ar.GetSetAsListOk(_CheckStreamAttr); ok {
		c.Metrics = make([]api.CheckBundleMetric, 0, len(streamList))

		for _, metricListRaw := range streamList {
			metricAttrs := _NewInterfaceMap(metricListRaw)

			var id string
			if v, ok := ar.GetStringOk(_MetricIDAttr); ok {
				id = v
			} else {
				var err error
				id, err = _NewMetricID()
				if err != nil {
					return errwrap.Wrapf("unable to create a new metric ID: {{err}}", err)
				}
			}

			m := _NewMetric()
			mr := _NewMapReader(ar.Context(), metricAttrs)
			if err := m.ParseConfig(id, mr); err != nil {
				return errwrap.Wrapf("unable to parse config: {{err}}", err)
			}

			c.Metrics = append(c.Metrics, m.CheckBundleMetric)
		}
	}

	if l, ok := ar.GetSetAsListOk(_CheckJSONAttr); ok {
		if err := c.parseJSONCheck(l); err != nil {
			return err
		}
	}

	if i, ok := ar.GetIntOK(_CheckMetricLimitAttr); ok {
		c.MetricLimit = i
	}

	c.Notes = ar.GetStringPtr(_CheckNotesAttr)

	if d, ok := ar.GetDurationOK(_CheckPeriodAttr); ok {
		c.Period = uint(d.Seconds())
	}

	c.Tags = tagsToAPI(ar.GetTags(_CheckTagsAttr))

	if s, ok := ar.GetStringOk(_CheckTargetAttr); ok {
		c.Target = s
	}

	if d, ok := ar.GetDurationOK(_CheckTimeoutAttr); ok {
		var t float32 = float32(d.Seconds())
		c.Timeout = t
	}

	if err := c.Validate(); err != nil {
		return err
	}

	return nil
}

func (c *_Check) Validate() error {
	if c.Timeout > float32(c.Period) {
		return fmt.Errorf("Timeout (%f) can not exceed period (%d)", c.Timeout, c.Period)
	}

	return nil
}

func apiCheckStatusToBool(s string) bool {
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
