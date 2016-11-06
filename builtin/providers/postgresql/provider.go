package postgresql

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:     schema.TypeString,
				Optional: true,
				// TODO(sean@): Remove POSTGRESQL_HOST in 0.8
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"PGHOST", "POSTGRESQL_HOST"}, nil),
				Description: "Name of PostgreSQL server address to connect to",
			},
			"port": {
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PGPORT", 5432),
				Description: "The PostgreSQL port number to connect to at the server host, or socket file name extension for Unix-domain connections",
			},
			"database": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the database to connect to in order to conenct to (defaults to `postgres`).",
				DefaultFunc: schema.EnvDefaultFunc("PGDATABASE", "postgres"),
			},
			"username": {
				Type:     schema.TypeString,
				Optional: true,
				// TODO(sean@): Remove POSTGRESQL_USER in 0.8
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"PGUSER", "POSTGRESQL_USER"}, "postgres"),
				Description: "PostgreSQL user name to connect as",
			},
			"password": {
				Type:     schema.TypeString,
				Optional: true,
				// TODO(sean@): Remove POSTGRESQL_PASSWORD in 0.8
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"PGPASSWORD", "POSTGRESQL_PASSWORD"}, nil),
				Description: "Password to be used if the PostgreSQL server demands password authentication",
			},
			"sslmode": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PGSSLMODE", nil),
				Description: "This option determines whether or with what priority a secure SSL TCP/IP connection will be negotiated with the PostgreSQL server",
			},
			"connect_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     15,
				DefaultFunc: schema.EnvDefaultFunc("PGCONNECT_TIMEOUT", nil),
				Description: "Maximum wait for connection, in seconds. Zero or not specified means wait indefinitely.",
			},
			"ssl_mode": {
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "Rename PostgreSQL provider `ssl_mode` attribute to `sslmode`",
			},
			"connect_timeout": {
				Type:         schema.TypeInt,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("PGCONNECT_TIMEOUT", 180),
				Description:  "Maximum wait for connection, in seconds. Zero or not specified means wait indefinitely.",
				ValidateFunc: validateConnTimeout,
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"postgresql_database":  resourcePostgreSQLDatabase(),
			"postgresql_role":      resourcePostgreSQLRole(),
			"postgresql_extension": resourcePostgreSQLExtension(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func validateConnTimeout(v interface{}, key string) (warnings []string, errors []error) {
	value := v.(int)
	if value < 0 {
		errors = append(errors, fmt.Errorf("%d can not be less than 0", key))
	}
	return
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	var sslMode string
	var ok bool
	if sslMode, ok = d.GetOk("sslmode").(string); !ok {
		sslMode = d.Get("ssl_mode").(string)
	}
	config := Config{
		Host:              d.Get("host").(string),
		Port:              d.Get("port").(int),
		Database:          d.Get("database").(string),
		Username:          d.Get("username").(string),
		Password:          d.Get("password").(string),
		SSLMode:           sslMode,
		ApplicationName:   tfAppName(),
		ConnectTimeoutSec: d.Get("connect_timeout").(int),
	}

	client, err := config.NewClient()
	if err != nil {
		return nil, errwrap.Wrapf("Error initializing PostgreSQL client: %s", err)
	}

	return client, nil
}

func tfAppName() string {
	const VersionPrerelease = terraform.VersionPrerelease
	var versionString bytes.Buffer

	fmt.Fprintf(&versionString, "'Terraform v%s", terraform.Version)
	if terraform.VersionPrerelease != "" {
		fmt.Fprintf(&versionString, "-%s", terraform.VersionPrerelease)
	}
	fmt.Fprintf(&versionString, "'")

	return versionString.String()
}
