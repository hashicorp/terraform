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
	// circonus_check.mysql.* resource attribute names
	checkMySQLDSNAttr   schemaAttr = "dsn"
	checkMySQLQueryAttr schemaAttr = "query"
)

var checkMySQLDescriptions = attrDescrs{
	checkMySQLDSNAttr:   "The connect DSN for the MySQL instance",
	checkMySQLQueryAttr: "The SQL to use as the query",
}

var schemaCheckMySQL = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      hashCheckMySQL,
	Elem: &schema.Resource{
		Schema: castSchemaToTF(map[schemaAttr]*schema.Schema{
			checkMySQLDSNAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRegexp(checkMySQLDSNAttr, `^.+$`),
			},
			checkMySQLQueryAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				StateFunc:    func(v interface{}) string { return strings.TrimSpace(v.(string)) },
				ValidateFunc: validateRegexp(checkMySQLQueryAttr, `.+`),
			},
		}, checkMySQLDescriptions),
	},
}

// checkAPIToStateMySQL reads the Config data out of circonusCheck.CheckBundle into the
// statefile.
func checkAPIToStateMySQL(c *circonusCheck, d *schema.ResourceData) error {
	MySQLConfig := make(map[string]interface{}, len(c.Config))

	MySQLConfig[string(checkMySQLDSNAttr)] = c.Config[config.DSN]
	MySQLConfig[string(checkMySQLQueryAttr)] = c.Config[config.SQL]

	if err := d.Set(checkMySQLAttr, schema.NewSet(hashCheckMySQL, []interface{}{MySQLConfig})); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkMySQLAttr), err)
	}

	return nil
}

// hashCheckMySQL creates a stable hash of the normalized values
func hashCheckMySQL(v interface{}) int {
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
	writeString(checkMySQLDSNAttr)
	writeString(checkMySQLQueryAttr)

	s := b.String()
	return hashcode.String(s)
}

func checkConfigToAPIMySQL(c *circonusCheck, ctxt *providerContext, l interfaceList) error {
	c.Type = string(apiCheckTypeMySQL)

	// Iterate over all `postgres` attributes, even though we have a max of 1 in
	// the schema.
	for _, mapRaw := range l {
		mysqlConfig := newInterfaceMap(mapRaw)
		ar := newMapReader(ctxt, mysqlConfig)

		if s, ok := ar.GetStringOK(checkMySQLDSNAttr); ok {
			c.Config[config.DSN] = s
		}

		if s, ok := ar.GetStringOK(checkMySQLQueryAttr); ok {
			c.Config[config.SQL] = s
		}
	}

	return nil
}
