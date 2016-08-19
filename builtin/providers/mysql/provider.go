package mysql

import (
	"fmt"
	"strconv"
	"strings"

	mysqlc "github.com/ziutek/mymysql/mysql"
	mysqlts "github.com/ziutek/mymysql/thrsafe"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

type providerConfiguration struct {
	Conn         mysqlc.Conn
	VersionMajor uint
	VersionMinor uint
	VersionPatch uint
}

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"endpoint": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("MYSQL_ENDPOINT", nil),
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value == "" {
						errors = append(errors, fmt.Errorf("Endpoint must not be an empty string"))
					}

					return
				},
			},

			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("MYSQL_USERNAME", nil),
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value == "" {
						errors = append(errors, fmt.Errorf("Username must not be an empty string"))
					}

					return
				},
			},

			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("MYSQL_PASSWORD", nil),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"mysql_database": resourceDatabase(),
			"mysql_user":     resourceUser(),
			"mysql_grant":    resourceGrant(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {

	var username = d.Get("username").(string)
	var password = d.Get("password").(string)
	var endpoint = d.Get("endpoint").(string)

	proto := "tcp"
	if endpoint[0] == '/' {
		proto = "unix"
	}

	// mysqlts is the thread-safe implementation of mymysql, so we can
	// safely re-use the same connection between multiple parallel
	// operations.
	conn := mysqlts.New(proto, "", endpoint, username, password)

	err := conn.Connect()
	if err != nil {
		return nil, err
	}

	major, minor, patch, err := mysqlVersion(conn)
	if err != nil {
		return nil, err
	}

	return &providerConfiguration{
		Conn:         conn,
		VersionMajor: major,
		VersionMinor: minor,
		VersionPatch: patch,
	}, nil
}

var identQuoteReplacer = strings.NewReplacer("`", "``")

func quoteIdentifier(in string) string {
	return fmt.Sprintf("`%s`", identQuoteReplacer.Replace(in))
}

func mysqlVersion(conn mysqlc.Conn) (uint, uint, uint, error) {
	rows, _, err := conn.Query("SELECT VERSION()")
	if err != nil {
		return 0, 0, 0, err
	}
	if len(rows) == 0 {
		return 0, 0, 0, fmt.Errorf("SELECT VERSION() returned an empty set")
	}

	versionString := rows[0].Str(0)
	version := strings.Split(versionString, ".")
	invalidVersionErr := fmt.Errorf("Invalid major.minor.patch in %q", versionString)
	if len(version) != 3 {
		return 0, 0, 0, invalidVersionErr
	}

	major, err := strconv.ParseUint(version[0], 10, 32)
	if err != nil {
		return 0, 0, 0, invalidVersionErr
	}

	minor, err := strconv.ParseUint(version[1], 10, 32)
	if err != nil {
		return 0, 0, 0, invalidVersionErr
	}

	patch, err := strconv.ParseUint(version[2], 10, 32)
	if err != nil {
		return 0, 0, 0, invalidVersionErr
	}

	return uint(major), uint(minor), uint(patch), nil
}
