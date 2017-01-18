package circonus

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_check.icmp_ping.* resource attribute names
	_CheckICMPPingAvailabilityAttr _SchemaAttr = "availability"
	_CheckICMPPingCountAttr        _SchemaAttr = "count"
	_CheckICMPPingIntervalAttr     _SchemaAttr = "interval"
)

var _CheckICMPPingDescriptions = _AttrDescrs{
	_CheckICMPPingAvailabilityAttr: `The percentage of ICMP available required for the check to be considered "good."`,
	_CheckICMPPingCountAttr:        "The number of ICMP requests to send during a single check.",
	_CheckICMPPingIntervalAttr:     "The number of milliseconds between ICMP requests.",
}

var _SchemaCheckICMPPing = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      hashCheckICMPPing,
	Elem: &schema.Resource{
		Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
			_CheckICMPPingAvailabilityAttr: &schema.Schema{
				Type:     schema.TypeFloat,
				Optional: true,
				Default:  defaultCheckICMPPingAvailability,
				ValidateFunc: _ValidateFuncs(
					_ValidateFloatMin(_CheckICMPPingAvailabilityAttr, 0.0),
					_ValidateFloatMax(_CheckICMPPingAvailabilityAttr, 100.0),
				),
			},
			_CheckICMPPingCountAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  defaultCheckICMPPingCount,
				ValidateFunc: _ValidateFuncs(
					_ValidateIntMin(_CheckICMPPingCountAttr, 0),
					_ValidateIntMax(_CheckICMPPingCountAttr, 20),
				),
			},
			_CheckICMPPingIntervalAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  defaultCheckICMPPingInterval,
				ValidateFunc: _ValidateFuncs(
					_ValidateDurationMin(_CheckICMPPingIntervalAttr, "100µs"),
					_ValidateDurationMax(_CheckICMPPingIntervalAttr, "5m"),
				),
			},
		}, _CheckICMPPingDescriptions),
	},
}

// _ReadAPICheckConfigICMPPing reads the Config data out of _Check.CheckBundle
// into the statefile.
func _ReadAPICheckConfigICMPPing(c *_Check, d *schema.ResourceData) error {
	icmpPingConfig := make(map[string]interface{}, len(c.Config))

	availNeeded, err := strconv.ParseFloat(c.Config[config.AvailNeeded], 64)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to parse %s: {{err}}", config.AvailNeeded), err)
	}

	count, err := strconv.ParseInt(c.Config[config.Count], 10, 64)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to parse %s: {{err}}", config.Count), err)
	}

	interval, err := time.ParseDuration(fmt.Sprintf("%sms", c.Config[config.Interval]))
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to parse %s: {{err}}", config.Interval), err)
	}

	icmpPingConfig[string(_CheckICMPPingAvailabilityAttr)] = availNeeded
	icmpPingConfig[string(_CheckICMPPingCountAttr)] = int(count)
	icmpPingConfig[string(_CheckICMPPingIntervalAttr)] = interval.String()

	_StateSet(d, _CheckICMPPingAttr, schema.NewSet(hashCheckICMPPing, []interface{}{icmpPingConfig}))

	return nil
}

// hashCheckICMPPing creates a stable hash of the normalized values
func hashCheckICMPPing(v interface{}) int {
	m := v.(map[string]interface{})
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeFloat64 := func(attrName _SchemaAttr) {
		if v, ok := m[string(attrName)]; ok {
			fmt.Fprintf(b, "%f", v.(float64))
		}
	}

	writeInt := func(attrName _SchemaAttr) {
		if v, ok := m[string(attrName)]; ok {
			fmt.Fprintf(b, "%x", v.(int))
		}
	}

	writeDuration := func(attrName _SchemaAttr) {
		if v, ok := m[string(attrName)]; ok && v.(string) != "" {
			d, _ := time.ParseDuration(v.(string))
			fmt.Fprint(b, d.String())
		}
	}

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	writeFloat64(_CheckICMPPingAvailabilityAttr)
	writeInt(_CheckICMPPingCountAttr)
	writeDuration(_CheckICMPPingIntervalAttr)

	s := b.String()
	return hashcode.String(s)
}

func parseCheckConfigICMPPing(c *_Check, ctxt *_ProviderContext, l _InterfaceList) error {
	c.Type = string(_APICheckTypeICMPPing)

	// Iterate over all `icmp_ping` attributes, even though we have a max of 1 in
	// the schema.
	for _, mapRaw := range l {
		icmpPingConfig := _NewInterfaceMap(mapRaw)
		ar := _NewMapReader(ctxt, icmpPingConfig)

		if f, ok := ar.GetFloat64OK(_CheckICMPPingAvailabilityAttr); ok {
			c.Config[config.AvailNeeded] = fmt.Sprintf("%d", int(f))
		}

		if i, ok := ar.GetIntOK(_CheckICMPPingCountAttr); ok {
			c.Config[config.Count] = fmt.Sprintf("%d", i)
		}

		if s, ok := ar.GetStringOK(_CheckICMPPingIntervalAttr); ok {
			d, _ := time.ParseDuration(s)
			c.Config[config.Interval] = fmt.Sprintf("%d", int64(d/time.Millisecond))
		}
	}

	return nil
}
