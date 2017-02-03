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
	// circonus_check.mysql.* resource attribute names
	_CheckMySQLDSNAttr   _SchemaAttr = "dsn"
	_CheckMySQLQueryAttr _SchemaAttr = "query"
)

var _CheckMySQLDescriptions = _AttrDescrs{
	_CheckMySQLDSNAttr:   "The connect DSN for the MySQL instance",
	_CheckMySQLQueryAttr: "The SQL to use as the query",
}

var _SchemaCheckMySQL = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      hashCheckMySQL,
	Elem: &schema.Resource{
		Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
			_CheckMySQLDSNAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: _ValidateRegexp(_CheckMySQLDSNAttr, `^.+$`),
			},
			_CheckMySQLQueryAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				StateFunc:    func(v interface{}) string { return strings.TrimSpace(v.(string)) },
				ValidateFunc: _ValidateRegexp(_CheckMySQLQueryAttr, `.+`),
			},
		}, _CheckMySQLDescriptions),
	},
}

// _CheckAPIToStateMySQL reads the Config data out of _Check.CheckBundle into the
// statefile.
func _CheckAPIToStateMySQL(c *_Check, d *schema.ResourceData) error {
	MySQLConfig := make(map[string]interface{}, len(c.Config))

	MySQLConfig[string(_CheckMySQLDSNAttr)] = c.Config[config.DSN]
	MySQLConfig[string(_CheckMySQLQueryAttr)] = c.Config[config.SQL]

	_StateSet(d, _CheckMySQLAttr, schema.NewSet(hashCheckMySQL, []interface{}{MySQLConfig}))

	return nil
}

// hashCheckMySQL creates a stable hash of the normalized values
func hashCheckMySQL(v interface{}) int {
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
	writeString(_CheckMySQLDSNAttr)
	writeString(_CheckMySQLQueryAttr)

	s := b.String()
	return hashcode.String(s)
}

func _CheckConfigToAPIMySQL(c *_Check, ctxt *_ProviderContext, l _InterfaceList) error {
	c.Type = string(_APICheckTypeMySQL)

	// Iterate over all `postgres` attributes, even though we have a max of 1 in
	// the schema.
	for _, mapRaw := range l {
		mysqlConfig := _NewInterfaceMap(mapRaw)
		ar := _NewMapReader(ctxt, mysqlConfig)

		if s, ok := ar.GetStringOK(_CheckMySQLDSNAttr); ok {
			c.Config[config.DSN] = s
		}

		if s, ok := ar.GetStringOK(_CheckMySQLQueryAttr); ok {
			c.Config[config.SQL] = s
		}
	}

	return nil
}
