package circonus

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_check.caql.* resource attribute names
	checkCAQLQueryAttr = "query"
)

var checkCAQLDescriptions = attrDescrs{
	checkCAQLQueryAttr: "The query definition",
}

var schemaCheckCAQL = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      hashCheckCAQL,
	Elem: &schema.Resource{
		Schema: convertToHelperSchema(checkCAQLDescriptions, map[schemaAttr]*schema.Schema{
			checkCAQLQueryAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRegexp(checkCAQLQueryAttr, `.+`),
			},
		}),
	},
}

// checkAPIToStateCAQL reads the Config data out of circonusCheck.CheckBundle
// into the statefile.
func checkAPIToStateCAQL(c *circonusCheck, d *schema.ResourceData) error {
	caqlConfig := make(map[string]interface{}, len(c.Config))

	caqlConfig[string(checkCAQLQueryAttr)] = c.Config[config.Query]

	if err := d.Set(checkCAQLAttr, schema.NewSet(hashCheckCAQL, []interface{}{caqlConfig})); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkCAQLAttr), err)
	}

	return nil
}

// hashCheckCAQL creates a stable hash of the normalized values
func hashCheckCAQL(v interface{}) int {
	m := v.(map[string]interface{})
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeString := func(attrName schemaAttr) {
		if v, ok := m[string(attrName)]; ok && v.(string) != "" {
			fmt.Fprint(b, strings.TrimSpace(v.(string)))
		}
	}

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	writeString(checkCAQLQueryAttr)

	s := b.String()
	return hashcode.String(s)
}

func checkConfigToAPICAQL(c *circonusCheck, l interfaceList) error {
	c.Type = string(apiCheckTypeCAQL)
	c.Target = defaultCheckCAQLTarget

	// Iterate over all `caql` attributes, even though we have a max of 1 in the
	// schema.
	for _, mapRaw := range l {
		caqlConfig := newInterfaceMap(mapRaw)

		if v, found := caqlConfig[checkCAQLQueryAttr]; found {
			c.Config[config.Query] = v.(string)
		}
	}

	return nil
}
