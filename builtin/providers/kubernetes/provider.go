package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"k8s.io/kubernetes/pkg/client"
)

func Provider() terraform.ResourceProvider {

	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"endpoint": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBERNETES_ENDPOINT", nil),
				Description: descriptions["endpoint"],
			},

			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBERNETES_USERNAME", nil),
				Description: descriptions["username"],
			},

			"password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBERNETES_PASSWORD", nil),
				Description: descriptions["password"],
			},

			"insecure": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				DefaultFunc: schema.EnvDefaultFunc("KUBERNETES_INSECURE", nil),
				Description: descriptions["insecure"],
			},

			"client_certificate": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["client_certificate"],
				// TODO: KUBERNETES_CLIENT_CERTIFICATE or KUBERNETES_CLIENT_CERTIFICATE_PATH ?
			},

			"client_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["client_key"],
				// TODO: KUBERNETES_CLIENT_KEY or KUBERNETES_CLIENT_KEY_PATH ?
			},

			"cluster_ca_certificate": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["cluster_ca_certificate"],
				// TODO: KUBERNETES_CLUSTER_CA_CERTIFICATE or KUBERNETES_CLUSTER_CA_CERTIFICATE_PATH ?
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"kubernetes_namespace":              resourceKubernetesNamespace(),
			"kubernetes_persistent_volume":      resourceKubernetesPersistentVolume(),
			"kubernetes_pod":                    resourceKubernetesPod(),
			"kubernetes_replication_controller": resourceKubernetesReplicationController(),
			"kubernetes_resource_quota":         resourceKubernetesResourceQuota(),
			"kubernetes_service":                resourceKubernetesService(),
		},

		ConfigureFunc: providerConfigure,
	}
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"endpoint": "The hostname (in form of URI) of Kubernetes master.",

		"username": "The username to use for HTTP basic authentication\n" +
			"when accessing the Kubernetes master endpoint.",

		"password": "The password to use for HTTP basic authentication\n" +
			"when accessing the Kubernetes master endpoint.",

		"insecure": "Whether server should be accessed without verifying " +
			"the TLS certificate.",

		"client_certificate": "PEM-encoded client certificate for " +
			"TLS authentication.",

		"client_key": "PEM-encoded client certificate key for TLS " +
			"authentication.",

		"cluster_ca_certificate": "PEM-encoded root certificates bundle " +
			"for TLS authentication.",
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	insecure := d.Get("insecure").(bool)

	config := client.Config{
		Host:     d.Get("endpoint").(string),
		Username: d.Get("username").(string),
		Password: d.Get("password").(string),
		Insecure: insecure,
	}

	tlsConfig := client.TLSClientConfig{}
	addTlsConfig := false

	if v, ok := d.GetOk("client_certificate"); ok {
		tlsConfig.CertData = []byte(v.(string))
		addTlsConfig = true
	}
	if v, ok := d.GetOk("client_key"); ok {
		tlsConfig.KeyData = []byte(v.(string))
		addTlsConfig = true
	}
	if v, ok := d.GetOk("cluster_ca_certificate"); ok {
		tlsConfig.CAData = []byte(v.(string))
		addTlsConfig = true
	}

	if insecure && addTlsConfig {
		return nil, fmt.Errorf("You can either specify 'insecure' " +
			"or provide TLS configuration, not both")
	}

	if addTlsConfig {
		config.TLSClientConfig = tlsConfig
	}

	return client.New(&config)
}
