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
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"PGHOST", "POSTGRESQL_HOST"}, nil),
				Description: "The PostgreSQL server address",
			},
			"port": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     5432,
				Description: "The PostgreSQL server port",
			},
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"PGUSER", "POSTGRESQL_USER"}, nil),
				Description: "Username for PostgreSQL server connection",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"PGPASSWORD", "POSTGRESQL_PASSWORD"}, nil),
				Description: "Password for PostgreSQL server connection",
			},
			"ssl_mode": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PGSSLMODE", "require"),
				Description: "Connection mode for PostgreSQL server",
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
		SslMode:  d.Get("ssl_mode").(string),
	}

	client, err := config.NewClient()
	if err != nil {
		return nil, errwrap.Wrapf("Error initializing PostgreSQL client: %s", err)
	}

	return client, nil
}
