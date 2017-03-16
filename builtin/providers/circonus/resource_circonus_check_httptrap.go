package circonus

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_check.httptrap.* resource attribute names
	checkHTTPTrapAsyncMetricsAttr = "async_metrics"
	checkHTTPTrapSecretAttr       = "secret"
)

var checkHTTPTrapDescriptions = attrDescrs{
	checkHTTPTrapAsyncMetricsAttr: "Specify whether httptrap metrics are logged immediately or held until the status message is emitted",
	checkHTTPTrapSecretAttr:       "",
}

var schemaCheckHTTPTrap = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      hashCheckHTTPTrap,
	Elem: &schema.Resource{
		Schema: convertToHelperSchema(checkHTTPTrapDescriptions, map[schemaAttr]*schema.Schema{
			checkHTTPTrapAsyncMetricsAttr: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  defaultCheckHTTPTrapAsync,
			},
			checkHTTPTrapSecretAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				ValidateFunc: validateRegexp(checkHTTPTrapSecretAttr, `^[a-zA-Z0-9_]+$`),
			},
		}),
	},
}

// checkAPIToStateHTTPTrap reads the Config data out of circonusCheck.CheckBundle into
// the statefile.
func checkAPIToStateHTTPTrap(c *circonusCheck, d *schema.ResourceData) error {
	httpTrapConfig := make(map[string]interface{}, len(c.Config))

	// swamp is a sanity check: it must be empty by the time this method returns
	swamp := make(map[config.Key]string, len(c.Config))
	for k, v := range c.Config {
		swamp[k] = v
	}

	saveBoolConfigToState := func(apiKey config.Key, attrName schemaAttr) {
		if s, ok := c.Config[apiKey]; ok {
			switch s {
			case "true", "on":
				httpTrapConfig[string(attrName)] = true
			case "false", "off":
				httpTrapConfig[string(attrName)] = false
			default:
				log.Printf("PROVIDER BUG: unsupported value %q returned in key %q", s, apiKey)
			}
		}

		delete(swamp, apiKey)
	}

	saveStringConfigToState := func(apiKey config.Key, attrName schemaAttr) {
		if s, ok := c.Config[apiKey]; ok {
			httpTrapConfig[string(attrName)] = s
		}

		delete(swamp, apiKey)
	}

	saveBoolConfigToState(config.AsyncMetrics, checkHTTPTrapAsyncMetricsAttr)
	saveStringConfigToState(config.Secret, checkHTTPTrapSecretAttr)

	whitelistedConfigKeys := map[config.Key]struct{}{
		config.ReverseSecretKey: struct{}{},
		config.SubmissionURL:    struct{}{},
	}

	for k := range swamp {
		if _, ok := whitelistedConfigKeys[k]; ok {
			delete(c.Config, k)
		}

		if _, ok := whitelistedConfigKeys[k]; !ok {
			log.Printf("[ERROR]: PROVIDER BUG: API Config not empty: %#v", swamp)
		}
	}

	if err := d.Set(checkHTTPTrapAttr, schema.NewSet(hashCheckHTTPTrap, []interface{}{httpTrapConfig})); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkHTTPTrapAttr), err)
	}

	return nil
}

// hashCheckHTTPTrap creates a stable hash of the normalized values
func hashCheckHTTPTrap(v interface{}) int {
	m := v.(map[string]interface{})
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeBool := func(attrName schemaAttr) {
		if v, ok := m[string(attrName)]; ok {
			fmt.Fprintf(b, "%t", v.(bool))
		}
	}

	writeString := func(attrName schemaAttr) {
		if v, ok := m[string(attrName)]; ok && v.(string) != "" {
			fmt.Fprint(b, strings.TrimSpace(v.(string)))
		}
	}

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	writeBool(checkHTTPTrapAsyncMetricsAttr)
	writeString(checkHTTPTrapSecretAttr)

	s := b.String()
	return hashcode.String(s)
}

func checkConfigToAPIHTTPTrap(c *circonusCheck, l interfaceList) error {
	c.Type = string(apiCheckTypeHTTPTrapAttr)

	// Iterate over all `httptrap` attributes, even though we have a max of 1 in the
	// schema.
	for _, mapRaw := range l {
		httpTrapConfig := newInterfaceMap(mapRaw)

		if v, found := httpTrapConfig[checkHTTPTrapAsyncMetricsAttr]; found {
			b := v.(bool)
			if b {
				c.Config[config.AsyncMetrics] = fmt.Sprintf("%t", b)
			}
		}

		if v, found := httpTrapConfig[checkHTTPTrapSecretAttr]; found {
			c.Config[config.Secret] = v.(string)
		}
	}

	return nil
}
