package mssql

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("MSSQL_HOST", nil),
				Description: "The MS SQL Server address",
			},
			"port": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1433,
				Description: "The MS SQL Server port",
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("MSSQL_USERNAME", nil),
				Description: "Username for MS SQL Server connection",
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("MSSQL_PASSWORD", nil),
				Description: "Password for MS SQL Server connection",
			},
			"encrypt": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "false",
				Description: "Encrypt data send between client and MS SQL Server",
			},
			"trust_server_certificate": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     "false",
				Description: "Trust(true) or check(false) the MS SQL Server SSL certificate",
			},
			"certificate": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The file that contains the public key certificate of the CA that signed the SQL Server certificate",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"mssql_database": resourceMSsqlDatabase(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Host:                   d.Get("host").(string),
		Port:                   d.Get("port").(int),
		Username:               d.Get("username").(string),
		Password:               d.Get("password").(string),
		Encrypt:                d.Get("encrypt").(string),
		TrustServerCertificate: d.Get("trust_server_certificate").(bool),
		Certificate:            d.Get("certificate").(string),
	}

	client, err := config.NewClient()
	if err != nil {
		return nil, fmt.Errorf("Error initializing MSSQL client: %s", err)
	}

	return client, nil
}
