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
	// circonus_check.postgresql.* resource attribute names
	checkPostgreSQLDSNAttr = "dsn"
	// checkPostgreSQLHostAttr      = "host"
	// checkPostgreSQLNameAttr      = "name"
	// checkPostgreSQLPasswordAttr  = "password"
	// checkPostgreSQLPortAttr      = "port"
	checkPostgreSQLQueryAttr = "query"
	// checkPostgreSQLSSLModeAttr   = "sslmode"
	// checkPostgreSQLUserAttr      = "user"
)

var checkPostgreSQLDescriptions = attrDescrs{
	checkPostgreSQLDSNAttr: "The connect DSN for the PostgreSQL instance",
	// checkPostgreSQLHostAttr:     "The Hostname to connect to",
	// checkPostgreSQLNameAttr:     "The database name to connect to",
	// checkPostgreSQLPasswordAttr: "The password to use",
	// checkPostgreSQLPortAttr:     "The TCP port number to use to connect on",
	checkPostgreSQLQueryAttr: "The SQL to use as the query",
	// checkPostgreSQLSSLModeAttr:  "The SSL Mode to connect as",
	// checkPostgreSQLUserAttr:     "The username to connect as",
}

var schemaCheckPostgreSQL = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      hashCheckPostgreSQL,
	Elem: &schema.Resource{
		Schema: convertToHelperSchema(checkPostgreSQLDescriptions, map[schemaAttr]*schema.Schema{
			checkPostgreSQLDSNAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRegexp(checkPostgreSQLDSNAttr, `^.+$`),
			},
			// TODO(sean@): Parse out the DSN into individual PostgreSQL connect
			// options.
			//
			// checkPostgreSQLHostAttr: &schema.Schema{
			// 	Type:         schema.TypeString,
			// 	Optional:     true,
			// 	Default:      "/tmp",
			// 	ValidateFunc: validateRegexp(checkPostgreSQLHostAttr, `^(/.+|[\S]+)$`),
			// },
			// checkPostgreSQLNameAttr: &schema.Schema{
			// 	Type:         schema.TypeString,
			// 	Required:     true,
			// 	ValidateFunc: validateRegexp(checkPostgreSQLNameAttr, `^[\S]+$`),
			// },
			// checkPostgreSQLPasswordAttr: &schema.Schema{
			// 	Type:      schema.TypeString,
			// 	Optional:  true,
			// 	Sensitive: true,
			// },
			// checkPostgreSQLPortAttr: &schema.Schema{
			// 	Type:     schema.TypeInt,
			// 	Optional: true,
			// 	Default:  5432,
			// 	ValidateFunc: validateFuncs(
			// 		validateIntMin(checkPostgreSQLPortAttr, 1),
			// 		validateIntMax(checkPostgreSQLPortAttr, 65535),
			// 	),
			// },
			checkPostgreSQLQueryAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				StateFunc:    suppressWhitespace,
				ValidateFunc: validateRegexp(checkPostgreSQLQueryAttr, `.+`),
			},
			// checkPostgreSQLSSLModeAttr: &schema.Schema{
			// 	Type:         schema.TypeString,
			// 	Optional:     true,
			// 	Default:      "require",
			// 	ValidateFunc: validateRegexp(checkPostgreSQLSSLModeAttr, `^(disable|require|verify-ca|verify-full)$`),
			// },
			// checkPostgreSQLUserAttr: &schema.Schema{
			// 	Type:         schema.TypeString,
			// 	Required:     true,
			// 	ValidateFunc: validateRegexp(checkPostgreSQLUserAttr, `.+`),
			// },
		}),
	},
}

// checkAPIToStatePostgreSQL reads the Config data out of circonusCheck.CheckBundle into the
// statefile.
func checkAPIToStatePostgreSQL(c *circonusCheck, d *schema.ResourceData) error {
	postgresqlConfig := make(map[string]interface{}, len(c.Config))

	// TODO(sean@): Parse out the DSN into individual PostgreSQL connect options
	postgresqlConfig[string(checkPostgreSQLDSNAttr)] = c.Config[config.DSN]
	postgresqlConfig[string(checkPostgreSQLQueryAttr)] = c.Config[config.SQL]

	if err := d.Set(checkPostgreSQLAttr, schema.NewSet(hashCheckPostgreSQL, []interface{}{postgresqlConfig})); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkPostgreSQLAttr), err)
	}

	return nil
}

// hashCheckPostgreSQL creates a stable hash of the normalized values
func hashCheckPostgreSQL(v interface{}) int {
	m := v.(map[string]interface{})
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	// writeInt := func(attrName schemaAttr) {
	// 	if v, ok := m[string(attrName)]; ok {
	// 		fmt.Fprintf(b, "%x", v.(int))
	// 	}
	// }

	writeString := func(attrName schemaAttr) {
		if v, ok := m[string(attrName)]; ok && v.(string) != "" {
			fmt.Fprint(b, strings.TrimSpace(v.(string)))
		}
	}

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	writeString(checkPostgreSQLDSNAttr)
	// writeString(checkPostgreSQLHostAttr)
	// writeString(checkPostgreSQLNameAttr)
	// writeString(checkPostgreSQLPasswordAttr)
	// writeInt(checkPostgreSQLPortAttr)
	// writeString(checkPostgreSQLSSLModeAttr)
	writeString(checkPostgreSQLQueryAttr)
	// writeString(checkPostgreSQLUserAttr)

	s := b.String()
	return hashcode.String(s)
}

func checkConfigToAPIPostgreSQL(c *circonusCheck, l interfaceList) error {
	c.Type = string(apiCheckTypePostgreSQL)

	// Iterate over all `postgres` attributes, even though we have a max of 1 in
	// the schema.
	for _, mapRaw := range l {
		postgresConfig := newInterfaceMap(mapRaw)

		if v, found := postgresConfig[checkPostgreSQLDSNAttr]; found {
			c.Config[config.DSN] = v.(string)
		}

		if v, found := postgresConfig[checkPostgreSQLQueryAttr]; found {
			c.Config[config.SQL] = v.(string)
		}
	}

	return nil
}
