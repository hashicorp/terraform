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
	checkICMPPingAvailabilityAttr = "availability"
	checkICMPPingCountAttr        = "count"
	checkICMPPingIntervalAttr     = "interval"
)

var checkICMPPingDescriptions = attrDescrs{
	checkICMPPingAvailabilityAttr: `The percentage of ICMP available required for the check to be considered "good."`,
	checkICMPPingCountAttr:        "The number of ICMP requests to send during a single check.",
	checkICMPPingIntervalAttr:     "The number of milliseconds between ICMP requests.",
}

var schemaCheckICMPPing = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      hashCheckICMPPing,
	Elem: &schema.Resource{
		Schema: convertToHelperSchema(checkICMPPingDescriptions, map[schemaAttr]*schema.Schema{
			checkICMPPingAvailabilityAttr: &schema.Schema{
				Type:     schema.TypeFloat,
				Optional: true,
				Default:  defaultCheckICMPPingAvailability,
				ValidateFunc: validateFuncs(
					validateFloatMin(checkICMPPingAvailabilityAttr, 0.0),
					validateFloatMax(checkICMPPingAvailabilityAttr, 100.0),
				),
			},
			checkICMPPingCountAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  defaultCheckICMPPingCount,
				ValidateFunc: validateFuncs(
					validateIntMin(checkICMPPingCountAttr, 0),
					validateIntMax(checkICMPPingCountAttr, 20),
				),
			},
			checkICMPPingIntervalAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  defaultCheckICMPPingInterval,
				ValidateFunc: validateFuncs(
					validateDurationMin(checkICMPPingIntervalAttr, "100µs"),
					validateDurationMax(checkICMPPingIntervalAttr, "5m"),
				),
			},
		}),
	},
}

// checkAPIToStateICMPPing reads the Config data out of circonusCheck.CheckBundle
// into the statefile.
func checkAPIToStateICMPPing(c *circonusCheck, d *schema.ResourceData) error {
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

	icmpPingConfig[string(checkICMPPingAvailabilityAttr)] = availNeeded
	icmpPingConfig[string(checkICMPPingCountAttr)] = int(count)
	icmpPingConfig[string(checkICMPPingIntervalAttr)] = interval.String()

	if err := d.Set(checkICMPPingAttr, schema.NewSet(hashCheckICMPPing, []interface{}{icmpPingConfig})); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkICMPPingAttr), err)
	}

	return nil
}

// hashCheckICMPPing creates a stable hash of the normalized values
func hashCheckICMPPing(v interface{}) int {
	m := v.(map[string]interface{})
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeFloat64 := func(attrName schemaAttr) {
		if v, ok := m[string(attrName)]; ok {
			fmt.Fprintf(b, "%f", v.(float64))
		}
	}

	writeInt := func(attrName schemaAttr) {
		if v, ok := m[string(attrName)]; ok {
			fmt.Fprintf(b, "%x", v.(int))
		}
	}

	writeDuration := func(attrName schemaAttr) {
		if v, ok := m[string(attrName)]; ok && v.(string) != "" {
			d, _ := time.ParseDuration(v.(string))
			fmt.Fprint(b, d.String())
		}
	}

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	writeFloat64(checkICMPPingAvailabilityAttr)
	writeInt(checkICMPPingCountAttr)
	writeDuration(checkICMPPingIntervalAttr)

	s := b.String()
	return hashcode.String(s)
}

func checkConfigToAPIICMPPing(c *circonusCheck, l interfaceList) error {
	c.Type = string(apiCheckTypeICMPPing)

	// Iterate over all `icmp_ping` attributes, even though we have a max of 1 in
	// the schema.
	for _, mapRaw := range l {
		icmpPingConfig := newInterfaceMap(mapRaw)

		if v, found := icmpPingConfig[checkICMPPingAvailabilityAttr]; found {
			f := v.(float64)
			c.Config[config.AvailNeeded] = fmt.Sprintf("%d", int(f))
		}

		if v, found := icmpPingConfig[checkICMPPingCountAttr]; found {
			c.Config[config.Count] = fmt.Sprintf("%d", v.(int))
		}

		if v, found := icmpPingConfig[checkICMPPingIntervalAttr]; found {
			d, _ := time.ParseDuration(v.(string))
			c.Config[config.Interval] = fmt.Sprintf("%d", int64(d/time.Millisecond))
		}
	}

	return nil
}
