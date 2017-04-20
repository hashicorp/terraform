package circonus

import (
	"fmt"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_check.statsd.* resource attribute names
	checkStatsdSourceIPAttr = "source_ip"
)

var checkStatsdDescriptions = attrDescrs{
	checkStatsdSourceIPAttr: "The source IP of the statsd metrics stream",
}

var schemaCheckStatsd = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Elem: &schema.Resource{
		Schema: convertToHelperSchema(checkStatsdDescriptions, map[schemaAttr]*schema.Schema{
			checkStatsdSourceIPAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRegexp(checkStatsdSourceIPAttr, `.+`),
			},
		}),
	},
}

// checkAPIToStateStatsd reads the Config data out of circonusCheck.CheckBundle
// into the statefile.
func checkAPIToStateStatsd(c *circonusCheck, d *schema.ResourceData) error {
	statsdConfig := make(map[string]interface{}, len(c.Config))

	// Unconditionally map the target to the source_ip config attribute
	statsdConfig[string(checkStatsdSourceIPAttr)] = c.Target

	if err := d.Set(checkStatsdAttr, []interface{}{statsdConfig}); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkStatsdAttr), err)
	}

	return nil
}

func checkConfigToAPIStatsd(c *circonusCheck, l interfaceList) error {
	c.Type = string(apiCheckTypeStatsd)

	// Iterate over all `statsd` attributes, even though we have a max of 1 in the
	// schema.
	for _, mapRaw := range l {
		statsdConfig := newInterfaceMap(mapRaw)

		if v, found := statsdConfig[checkStatsdSourceIPAttr]; found {
			switch {
			case c.Target == "":
				c.Target = v.(string)
			case c.Target != v.(string):
				return fmt.Errorf("Target (%q) must match %s (%q)", c.Target, checkStatsdSourceIPAttr, v.(string))
			}
		}
	}

	return nil
}
