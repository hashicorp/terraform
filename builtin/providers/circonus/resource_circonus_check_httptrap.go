package circonus

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_check.httptrap.* resource attribute names
	_CheckHTTPTrapAsyncMetricsAttr _SchemaAttr = "async_metrics"
	_CheckHTTPTrapSecretAttr       _SchemaAttr = "secret"
)

var _CheckHTTPTrapDescriptions = _AttrDescrs{
	_CheckHTTPTrapAsyncMetricsAttr: "",
	_CheckHTTPTrapSecretAttr:       "",
}

var _SchemaCheckHTTPTrap = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      hashCheckHTTPTrap,
	Elem: &schema.Resource{
		Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
			_CheckHTTPTrapAsyncMetricsAttr: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  defaultCheckHTTPTrapAsync,
			},
			_CheckHTTPTrapSecretAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				ValidateFunc: _ValidateRegexp(_CheckHTTPTrapSecretAttr, `^[a-zA-Z0-9_]+$`),
			},
		}, _CheckHTTPTrapDescriptions),
	},
}

// _CheckAPIToStateHTTPTrap reads the Config data out of _Check.CheckBundle into
// the statefile.
func _CheckAPIToStateHTTPTrap(c *_Check, d *schema.ResourceData) error {
	httpTrapConfig := make(map[string]interface{}, len(c.Config))

	// swamp is a sanity check: it must be empty by the time this method returns
	swamp := make(map[config.Key]string, len(c.Config))
	for k, v := range c.Config {
		swamp[k] = v
	}

	saveBoolConfigToState := func(apiKey config.Key, attrName _SchemaAttr) {
		if s, ok := c.Config[apiKey]; ok {
			switch s {
			case "true", "on":
				httpTrapConfig[string(attrName)] = true
			case "false", "off":
				httpTrapConfig[string(attrName)] = false
			default:
				panic(fmt.Sprintf("PROVIDER BUG: unsupported value %q returned in key %q", s, apiKey))
			}
		}

		delete(swamp, apiKey)
	}

	saveStringConfigToState := func(apiKey config.Key, attrName _SchemaAttr) {
		if s, ok := c.Config[apiKey]; ok {
			httpTrapConfig[string(attrName)] = s
		}

		delete(swamp, apiKey)
	}

	saveBoolConfigToState(config.AsyncMetrics, _CheckHTTPTrapAsyncMetricsAttr)
	saveStringConfigToState(config.Secret, _CheckHTTPTrapSecretAttr)

	whitelistedConfigKeys := map[config.Key]struct{}{
		config.ReverseSecretKey: struct{}{},
		config.SubmissionURL:    struct{}{},
	}

	for k := range swamp {
		if _, ok := whitelistedConfigKeys[k]; ok {
			delete(c.Config, k)
		}

		if _, ok := whitelistedConfigKeys[k]; !ok {
			panic(fmt.Sprintf("PROVIDER BUG: API Config not empty: %#v", swamp))
		}
	}

	_StateSet(d, _CheckHTTPTrapAttr, schema.NewSet(hashCheckHTTPTrap, []interface{}{httpTrapConfig}))

	return nil
}

// hashCheckHTTPTrap creates a stable hash of the normalized values
func hashCheckHTTPTrap(v interface{}) int {
	m := v.(map[string]interface{})
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeBool := func(attrName _SchemaAttr) {
		if v, ok := m[string(attrName)]; ok {
			fmt.Fprintf(b, "%t", v.(bool))
		}
	}

	writeString := func(attrName _SchemaAttr) {
		if v, ok := m[string(attrName)]; ok && v.(string) != "" {
			fmt.Fprint(b, strings.TrimSpace(v.(string)))
		}
	}

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	writeBool(_CheckHTTPTrapAsyncMetricsAttr)
	writeString(_CheckHTTPTrapSecretAttr)

	s := b.String()
	return hashcode.String(s)
}

func _CheckConfigToAPIHTTPTrap(c *_Check, ctxt *_ProviderContext, l _InterfaceList) error {
	c.Type = string(_APICheckTypeHTTPTrapAttr)

	// Iterate over all `httptrap` attributes, even though we have a max of 1 in the
	// schema.
	for _, mapRaw := range l {
		httpTrapConfig := _NewInterfaceMap(mapRaw)
		ar := _NewMapReader(ctxt, httpTrapConfig)

		if b, ok := ar.GetBoolOK(_CheckHTTPTrapAsyncMetricsAttr); ok {
			c.Config[config.AsyncMetrics] = fmt.Sprintf("%t", b)
		}

		if s, ok := ar.GetStringOK(_CheckHTTPTrapSecretAttr); ok {
			c.Config[config.Secret] = s
		}
	}

	return nil
}
