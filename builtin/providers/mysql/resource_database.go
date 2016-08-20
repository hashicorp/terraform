package mysql

import (
	"fmt"
	"log"
	"strings"

	mysqlc "github.com/ziutek/mymysql/mysql"

	"github.com/hashicorp/terraform/helper/schema"
)

const defaultCharacterSetKeyword = "CHARACTER SET "
const defaultCollateKeyword = "COLLATE "

func resourceDatabase() *schema.Resource {
	return &schema.Resource{
		Create: CreateDatabase,
		Update: UpdateDatabase,
		Read:   ReadDatabase,
		Delete: DeleteDatabase,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"default_character_set": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "utf8",
			},

			"default_collation": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "utf8_general_ci",
			},
		},
	}
}

func CreateDatabase(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*providerConfiguration).Conn

	stmtSQL := databaseConfigSQL("CREATE", d)
	log.Println("Executing statement:", stmtSQL)

	_, _, err := conn.Query(stmtSQL)
	if err != nil {
		return err
	}

	d.SetId(d.Get("name").(string))

	return nil
}

func UpdateDatabase(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*providerConfiguration).Conn

	stmtSQL := databaseConfigSQL("ALTER", d)
	log.Println("Executing statement:", stmtSQL)

	_, _, err := conn.Query(stmtSQL)
	if err != nil {
		return err
	}

	return nil
}

func ReadDatabase(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*providerConfiguration).Conn

	// This is kinda flimsy-feeling, since it depends on the formatting
	// of the SHOW CREATE DATABASE output... but this data doesn't seem
	// to be available any other way, so hopefully MySQL keeps this
	// compatible in future releases.

	name := d.Id()
	stmtSQL := "SHOW CREATE DATABASE " + quoteIdentifier(name)

	log.Println("Executing query:", stmtSQL)
	rows, _, err := conn.Query(stmtSQL)
	if err != nil {
		if mysqlErr, ok := err.(*mysqlc.Error); ok {
			if mysqlErr.Code == mysqlc.ER_BAD_DB_ERROR {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	row := rows[0]
	createSQL := string(row[1].([]byte))

	defaultCharset := extractIdentAfter(createSQL, defaultCharacterSetKeyword)
	defaultCollation := extractIdentAfter(createSQL, defaultCollateKeyword)

	if defaultCollation == "" && defaultCharset != "" {
		// MySQL doesn't return the collation if it's the default one for
		// the charset, so if we don't have a collation we need to go
		// hunt for the default.
		stmtSQL := "SHOW COLLATION WHERE `Charset` = '%s' AND `Default` = 'Yes'"
		rows, _, err := conn.Query(stmtSQL, defaultCharset)
		if err != nil {
			return fmt.Errorf("Error getting default charset: %s", err)
		}
		if len(rows) == 0 {
			return fmt.Errorf("Charset %s has no default collation", defaultCharset)
		}
		row := rows[0]
		defaultCollation = string(row[0].([]byte))
	}

	d.Set("default_character_set", defaultCharset)
	d.Set("default_collation", defaultCollation)

	return nil
}

func DeleteDatabase(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*providerConfiguration).Conn

	name := d.Id()
	stmtSQL := "DROP DATABASE " + quoteIdentifier(name)
	log.Println("Executing statement:", stmtSQL)

	_, _, err := conn.Query(stmtSQL)
	if err == nil {
		d.SetId("")
	}
	return err
}

func databaseConfigSQL(verb string, d *schema.ResourceData) string {
	name := d.Get("name").(string)
	defaultCharset := d.Get("default_character_set").(string)
	defaultCollation := d.Get("default_collation").(string)

	var defaultCharsetClause string
	var defaultCollationClause string

	if defaultCharset != "" {
		defaultCharsetClause = defaultCharacterSetKeyword + quoteIdentifier(defaultCharset)
	}
	if defaultCollation != "" {
		defaultCollationClause = defaultCollateKeyword + quoteIdentifier(defaultCollation)
	}

	return fmt.Sprintf(
		"%s DATABASE %s %s %s",
		verb,
		quoteIdentifier(name),
		defaultCharsetClause,
		defaultCollationClause,
	)
}

func extractIdentAfter(sql string, keyword string) string {
	charsetIndex := strings.Index(sql, keyword)
	if charsetIndex != -1 {
		charsetIndex += len(keyword)
		remain := sql[charsetIndex:]
		spaceIndex := strings.IndexRune(remain, ' ')
		return remain[:spaceIndex]
	}

	return ""
}
