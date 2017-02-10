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
	// circonus_check.postgresql.* resource attribute names
	_CheckPostgreSQLDSNAttr _SchemaAttr = "dsn"
	// _CheckPostgreSQLHostAttr     _SchemaAttr = "host"
	// _CheckPostgreSQLNameAttr     _SchemaAttr = "name"
	// _CheckPostgreSQLPasswordAttr _SchemaAttr = "password"
	// _CheckPostgreSQLPortAttr     _SchemaAttr = "port"
	_CheckPostgreSQLQueryAttr _SchemaAttr = "query"
	// _CheckPostgreSQLSSLModeAttr  _SchemaAttr = "sslmode"
	// _CheckPostgreSQLUserAttr     _SchemaAttr = "user"
)

var _CheckPostgreSQLDescriptions = _AttrDescrs{
	_CheckPostgreSQLDSNAttr: "The connect DSN for the PostgreSQL instance",
	// _CheckPostgreSQLHostAttr:     "The Hostname to connect to",
	// _CheckPostgreSQLNameAttr:     "The database name to connect to",
	// _CheckPostgreSQLPasswordAttr: "The password to use",
	// _CheckPostgreSQLPortAttr:     "The TCP port number to use to connect on",
	_CheckPostgreSQLQueryAttr: "The SQL to use as the query",
	// _CheckPostgreSQLSSLModeAttr:  "The SSL Mode to connect as",
	// _CheckPostgreSQLUserAttr:     "The username to connect as",
}

var _SchemaCheckPostgreSQL = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      hashCheckPostgreSQL,
	Elem: &schema.Resource{
		Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
			_CheckPostgreSQLDSNAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: _ValidateRegexp(_CheckPostgreSQLDSNAttr, `^.+$`),
			},
			// TODO(sean@): Parse out the DSN into individual PostgreSQL connect
			// options.
			//
			// _CheckPostgreSQLHostAttr: &schema.Schema{
			// 	Type:         schema.TypeString,
			// 	Optional:     true,
			// 	Default:      "/tmp",
			// 	ValidateFunc: _ValidateRegexp(_CheckPostgreSQLHostAttr, `^(/.+|[\S]+)$`),
			// },
			// _CheckPostgreSQLNameAttr: &schema.Schema{
			// 	Type:         schema.TypeString,
			// 	Required:     true,
			// 	ValidateFunc: _ValidateRegexp(_CheckPostgreSQLNameAttr, `^[\S]+$`),
			// },
			// _CheckPostgreSQLPasswordAttr: &schema.Schema{
			// 	Type:      schema.TypeString,
			// 	Optional:  true,
			// 	Sensitive: true,
			// },
			// _CheckPostgreSQLPortAttr: &schema.Schema{
			// 	Type:     schema.TypeInt,
			// 	Optional: true,
			// 	Default:  5432,
			// 	ValidateFunc: validateFuncs(
			// 		validateIntMin(_CheckPostgreSQLPortAttr, 1),
			// 		validateIntMax(_CheckPostgreSQLPortAttr, 65535),
			// 	),
			// },
			_CheckPostgreSQLQueryAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				StateFunc:    suppressWhitespace,
				ValidateFunc: _ValidateRegexp(_CheckPostgreSQLQueryAttr, `.+`),
			},
			// _CheckPostgreSQLSSLModeAttr: &schema.Schema{
			// 	Type:         schema.TypeString,
			// 	Optional:     true,
			// 	Default:      "require",
			// 	ValidateFunc: _ValidateRegexp(_CheckPostgreSQLSSLModeAttr, `^(disable|require|verify-ca|verify-full)$`),
			// },
			// _CheckPostgreSQLUserAttr: &schema.Schema{
			// 	Type:         schema.TypeString,
			// 	Required:     true,
			// 	ValidateFunc: _ValidateRegexp(_CheckPostgreSQLUserAttr, `.+`),
			// },
		}, _CheckPostgreSQLDescriptions),
	},
}

// _CheckAPIToStatePostgreSQL reads the Config data out of _Check.CheckBundle into the
// statefile.
func _CheckAPIToStatePostgreSQL(c *_Check, d *schema.ResourceData) error {
	postgresqlConfig := make(map[string]interface{}, len(c.Config))

	// TODO(sean@): Parse out the DSN into individual PostgreSQL connect options
	postgresqlConfig[string(_CheckPostgreSQLDSNAttr)] = c.Config[config.DSN]
	postgresqlConfig[string(_CheckPostgreSQLQueryAttr)] = c.Config[config.SQL]

	_StateSet(d, _CheckPostgreSQLAttr, schema.NewSet(hashCheckPostgreSQL, []interface{}{postgresqlConfig}))

	return nil
}

// hashCheckPostgreSQL creates a stable hash of the normalized values
func hashCheckPostgreSQL(v interface{}) int {
	m := v.(map[string]interface{})
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	// writeInt := func(attrName _SchemaAttr) {
	// 	if v, ok := m[string(attrName)]; ok {
	// 		fmt.Fprintf(b, "%x", v.(int))
	// 	}
	// }

	writeString := func(attrName _SchemaAttr) {
		if v, ok := m[string(attrName)]; ok && v.(string) != "" {
			fmt.Fprint(b, strings.TrimSpace(v.(string)))
		}
	}

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	writeString(_CheckPostgreSQLDSNAttr)
	// writeString(_CheckPostgreSQLHostAttr)
	// writeString(_CheckPostgreSQLNameAttr)
	// writeString(_CheckPostgreSQLPasswordAttr)
	// writeInt(_CheckPostgreSQLPortAttr)
	// writeString(_CheckPostgreSQLSSLModeAttr)
	writeString(_CheckPostgreSQLQueryAttr)
	// writeString(_CheckPostgreSQLUserAttr)

	s := b.String()
	return hashcode.String(s)
}

func _CheckConfigToAPIPostgreSQL(c *_Check, ctxt *_ProviderContext, l _InterfaceList) error {
	c.Type = string(_APICheckTypePostgreSQL)

	// Iterate over all `postgres` attributes, even though we have a max of 1 in
	// the schema.
	for _, mapRaw := range l {
		postgresConfig := _NewInterfaceMap(mapRaw)
		ar := _NewMapReader(ctxt, postgresConfig)

		if s, ok := ar.GetStringOK(_CheckPostgreSQLDSNAttr); ok {
			c.Config[config.DSN] = s
		}

		if s, ok := ar.GetStringOK(_CheckPostgreSQLQueryAttr); ok {
			c.Config[config.SQL] = s
		}
	}

	return nil
}
