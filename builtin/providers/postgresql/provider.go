package postgresql

import (
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
				Description: "The PostgreSQL server address",
			},
			"port": {
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PGPORT", 5432),
				Description: "The PostgreSQL server port",
			},
			"username": {
				Type:     schema.TypeString,
				Optional: true,
				// TODO(sean@): Remove POSTGRESQL_USER in 0.8
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"PGUSER", "POSTGRESQL_USER"}, "postgres"),
				Description: "Username for PostgreSQL server connection",
			},
			"password": {
				Type:     schema.TypeString,
				Optional: true,
				// TODO(sean@): Remove POSTGRESQL_PASSWORD in 0.8
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"PGPASSWORD", "POSTGRESQL_PASSWORD"}, nil),
				Description: "Password for PostgreSQL server connection",
			},
			"sslmode": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PGSSLMODE", nil),
				Description: "Connection mode for PostgreSQL server",
			},
			"connect_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     15,
				DefaultFunc: schema.EnvDefaultFunc("PGCONNECT_TIMEOUT", nil),
				Description: "Maximum wait for connection, in seconds. Zero or not specified means wait indefinitely.",
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

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Host:     d.Get("host").(string),
		Port:     d.Get("port").(int),
		Username: d.Get("username").(string),
		Password: d.Get("password").(string),
		Timeout:  d.Get("connect_timeout").(int),
		SslMode:  d.Get("sslmode").(string),
	}

	client, err := config.NewClient()
	if err != nil {
		return nil, errwrap.Wrapf("Error initializing PostgreSQL client: %s", err)
	}

	return client, nil
}
