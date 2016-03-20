package influxdb

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/influxdata/influxdb/client"
)

var quoteReplacer = strings.NewReplacer(`"`, `\"`)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"influxdb_database": ResourceDatabase(),
		},

		Schema: map[string]*schema.Schema{
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.EnvDefaultFunc(
					"INFLUXDB_URL", "http://localhost:8086/",
				),
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("INFLUXDB_USERNAME", ""),
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("INFLUXDB_PASSWORD", ""),
			},
		},

		ConfigureFunc: Configure,
	}
}

func Configure(d *schema.ResourceData) (interface{}, error) {
	url, err := url.Parse(d.Get("url").(string))
	if err != nil {
		return nil, fmt.Errorf("invalid InfluxDB URL: %s", err)
	}

	config := client.Config{
		URL:      *url,
		Username: d.Get("username").(string),
		Password: d.Get("password").(string),
	}

	conn, err := client.NewClient(config)
	if err != nil {
		return nil, err
	}

	_, _, err = conn.Ping()
	if err != nil {
		return nil, fmt.Errorf("error pinging server: %s", err)
	}

	return conn, nil
}

func quoteIdentifier(ident string) string {
	return fmt.Sprintf(`"%s"`, quoteReplacer.Replace(ident))
}
