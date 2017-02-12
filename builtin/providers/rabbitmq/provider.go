package rabbitmq

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/michaelklishin/rabbit-hole"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"endpoint": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("RABBITMQ_ENDPOINT", nil),
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
				DefaultFunc: schema.EnvDefaultFunc("RABBITMQ_USERNAME", nil),
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
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("RABBITMQ_PASSWORD", nil),
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value == "" {
						errors = append(errors, fmt.Errorf("Password must not be an empty string"))
					}

					return
				},
			},

			"insecure": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("RABBITMQ_INSECURE", nil),
			},

			"cacert_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("RABBITMQ_CACERT", ""),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"rabbitmq_binding":     resourceBinding(),
			"rabbitmq_exchange":    resourceExchange(),
			"rabbitmq_permissions": resourcePermissions(),
			"rabbitmq_policy":      resourcePolicy(),
			"rabbitmq_queue":       resourceQueue(),
			"rabbitmq_user":        resourceUser(),
			"rabbitmq_vhost":       resourceVhost(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {

	var username = d.Get("username").(string)
	var password = d.Get("password").(string)
	var endpoint = d.Get("endpoint").(string)
	var insecure = d.Get("insecure").(bool)
	var cacertFile = d.Get("cacert_file").(string)

	// Configure TLS/SSL:
	// Ignore self-signed cert warnings
	// Specify a custom CA / intermediary cert
	// Specify a certificate and key
	tlsConfig := &tls.Config{}
	if cacertFile != "" {
		caCert, err := ioutil.ReadFile(cacertFile)
		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}
	if insecure {
		tlsConfig.InsecureSkipVerify = true
	}

	// Connect to RabbitMQ management interface
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	rmqc, err := rabbithole.NewTLSClient(endpoint, username, password, transport)
	if err != nil {
		return nil, err
	}

	return rmqc, nil
}
