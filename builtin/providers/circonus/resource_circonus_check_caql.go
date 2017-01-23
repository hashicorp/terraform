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
	_CheckCAQLQueryAttr _SchemaAttr = "query"
)

var _CheckCAQLDescriptions = _AttrDescrs{
	_CheckCAQLQueryAttr: "The query definition",
}

var _SchemaCheckCAQL = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      hashCheckCAQL,
	Elem: &schema.Resource{
		Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
			_CheckCAQLQueryAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: _ValidateRegexp(_CheckCAQLQueryAttr, `.+`),
			},
		}, _CheckCAQLDescriptions),
	},
}

// _CheckAPIToStateCAQL reads the Config data out of _Check.CheckBundle
// into the statefile.
func _CheckAPIToStateCAQL(c *_Check, d *schema.ResourceData) error {
	caqlConfig := make(map[string]interface{}, len(c.Config))

	caqlConfig[string(_CheckCAQLQueryAttr)] = c.Config[config.Query]

	_StateSet(d, _CheckCAQLAttr, schema.NewSet(hashCheckCAQL, []interface{}{caqlConfig}))

	return nil
}

// hashCheckCAQL creates a stable hash of the normalized values
func hashCheckCAQL(v interface{}) int {
	m := v.(map[string]interface{})
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeString := func(attrName _SchemaAttr) {
		if v, ok := m[string(attrName)]; ok && v.(string) != "" {
			fmt.Fprint(b, strings.TrimSpace(v.(string)))
		}
	}

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	writeString(_CheckCAQLQueryAttr)

	s := b.String()
	return hashcode.String(s)
}

func _CheckConfigToAPICAQL(c *_Check, ctxt *_ProviderContext, l _InterfaceList) error {
	c.Type = string(_APICheckTypeCAQL)
	c.Target = defaultCheckCAQLTarget

	// Iterate over all `icmp_ping` attributes, even though we have a max of 1 in
	// the schema.
	for _, mapRaw := range l {
		caqlConfig := _NewInterfaceMap(mapRaw)
		ar := _NewMapReader(ctxt, caqlConfig)

		if s, ok := ar.GetStringOK(_CheckCAQLQueryAttr); ok {
			c.Config[config.Query] = s
		}
	}

	return nil
}
