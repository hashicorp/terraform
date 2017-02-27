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
	// circonus_check.caql.* resource attribute names
	checkCAQLQueryAttr schemaAttr = "query"
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
		Schema: castSchemaToTF(map[schemaAttr]*schema.Schema{
			checkCAQLQueryAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRegexp(checkCAQLQueryAttr, `.+`),
			},
		}, checkCAQLDescriptions),
	},
}

// checkAPIToStateCAQL reads the Config data out of circonusCheck.CheckBundle
// into the statefile.
func checkAPIToStateCAQL(c *circonusCheck, d *schema.ResourceData) error {
	caqlConfig := make(map[string]interface{}, len(c.Config))

	caqlConfig[string(checkCAQLQueryAttr)] = c.Config[config.Query]

	stateSet(d, checkCAQLAttr, schema.NewSet(hashCheckCAQL, []interface{}{caqlConfig}))

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

func checkConfigToAPICAQL(c *circonusCheck, ctxt *providerContext, l interfaceList) error {
	c.Type = string(apiCheckTypeCAQL)
	c.Target = defaultCheckCAQLTarget

	// Iterate over all `icmp_ping` attributes, even though we have a max of 1 in
	// the schema.
	for _, mapRaw := range l {
		caqlConfig := newInterfaceMap(mapRaw)
		ar := newMapReader(ctxt, caqlConfig)

		if s, ok := ar.GetStringOK(checkCAQLQueryAttr); ok {
			c.Config[config.Query] = s
		}
	}

	return nil
}
