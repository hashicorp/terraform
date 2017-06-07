package azurerm

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/helper/mutexkv"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	riviera "github.com/jen20/riviera/azure"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	var p *schema.Provider
	p = &schema.Provider{
		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_SUBSCRIPTION_ID", ""),
			},

			"client_id": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_ID", ""),
			},

			"client_secret": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_SECRET", ""),
			},

			"tenant_id": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_TENANT_ID", ""),
			},

			"environment": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_ENVIRONMENT", "public"),
			},

			"skip_provider_registration": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_SKIP_PROVIDER_REGISTRATION", false),
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"azurerm_client_config": dataSourceArmClientConfig(),
			"azurerm_public_ip":     dataSourceArmPublicIP(),
		},

		ResourcesMap: map[string]*schema.Resource{
			// These resources use the Azure ARM SDK
			"azurerm_availability_set":   resourceArmAvailabilitySet(),
			"azurerm_cdn_endpoint":       resourceArmCdnEndpoint(),
			"azurerm_cdn_profile":        resourceArmCdnProfile(),
			"azurerm_container_registry": resourceArmContainerRegistry(),
			"azurerm_container_service":  resourceArmContainerService(),

			"azurerm_eventhub":                    resourceArmEventHub(),
			"azurerm_eventhub_authorization_rule": resourceArmEventHubAuthorizationRule(),
			"azurerm_eventhub_consumer_group":     resourceArmEventHubConsumerGroup(),
			"azurerm_eventhub_namespace":          resourceArmEventHubNamespace(),

			"azurerm_express_route_circuit": resourceArmExpressRouteCircuit(),

			"azurerm_lb":                      resourceArmLoadBalancer(),
			"azurerm_lb_backend_address_pool": resourceArmLoadBalancerBackendAddressPool(),
			"azurerm_lb_nat_rule":             resourceArmLoadBalancerNatRule(),
			"azurerm_lb_nat_pool":             resourceArmLoadBalancerNatPool(),
			"azurerm_lb_probe":                resourceArmLoadBalancerProbe(),
			"azurerm_lb_rule":                 resourceArmLoadBalancerRule(),

			"azurerm_managed_disk": resourceArmManagedDisk(),

			"azurerm_key_vault":                 resourceArmKeyVault(),
			"azurerm_local_network_gateway":     resourceArmLocalNetworkGateway(),
			"azurerm_network_interface":         resourceArmNetworkInterface(),
			"azurerm_network_security_group":    resourceArmNetworkSecurityGroup(),
			"azurerm_network_security_rule":     resourceArmNetworkSecurityRule(),
			"azurerm_public_ip":                 resourceArmPublicIp(),
			"azurerm_redis_cache":               resourceArmRedisCache(),
			"azurerm_route":                     resourceArmRoute(),
			"azurerm_route_table":               resourceArmRouteTable(),
			"azurerm_servicebus_namespace":      resourceArmServiceBusNamespace(),
			"azurerm_servicebus_subscription":   resourceArmServiceBusSubscription(),
			"azurerm_servicebus_topic":          resourceArmServiceBusTopic(),
			"azurerm_sql_elasticpool":           resourceArmSqlElasticPool(),
			"azurerm_storage_account":           resourceArmStorageAccount(),
			"azurerm_storage_blob":              resourceArmStorageBlob(),
			"azurerm_storage_container":         resourceArmStorageContainer(),
			"azurerm_storage_share":             resourceArmStorageShare(),
			"azurerm_storage_queue":             resourceArmStorageQueue(),
			"azurerm_storage_table":             resourceArmStorageTable(),
			"azurerm_subnet":                    resourceArmSubnet(),
			"azurerm_template_deployment":       resourceArmTemplateDeployment(),
			"azurerm_traffic_manager_endpoint":  resourceArmTrafficManagerEndpoint(),
			"azurerm_traffic_manager_profile":   resourceArmTrafficManagerProfile(),
			"azurerm_virtual_machine_extension": resourceArmVirtualMachineExtensions(),
			"azurerm_virtual_machine":           resourceArmVirtualMachine(),
			"azurerm_virtual_machine_scale_set": resourceArmVirtualMachineScaleSet(),
			"azurerm_virtual_network":           resourceArmVirtualNetwork(),
			"azurerm_virtual_network_peering":   resourceArmVirtualNetworkPeering(),

			// These resources use the Riviera SDK
			"azurerm_dns_a_record":      resourceArmDnsARecord(),
			"azurerm_dns_aaaa_record":   resourceArmDnsAAAARecord(),
			"azurerm_dns_cname_record":  resourceArmDnsCNameRecord(),
			"azurerm_dns_mx_record":     resourceArmDnsMxRecord(),
			"azurerm_dns_ns_record":     resourceArmDnsNsRecord(),
			"azurerm_dns_srv_record":    resourceArmDnsSrvRecord(),
			"azurerm_dns_txt_record":    resourceArmDnsTxtRecord(),
			"azurerm_dns_zone":          resourceArmDnsZone(),
			"azurerm_resource_group":    resourceArmResourceGroup(),
			"azurerm_search_service":    resourceArmSearchService(),
			"azurerm_sql_database":      resourceArmSqlDatabase(),
			"azurerm_sql_firewall_rule": resourceArmSqlFirewallRule(),
			"azurerm_sql_server":        resourceArmSqlServer(),
		},
	}

	p.ConfigureFunc = providerConfigure(p)

	return p
}

// Config is the configuration structure used to instantiate a
// new Azure management client.
type Config struct {
	ManagementURL string

	SubscriptionID           string
	ClientID                 string
	ClientSecret             string
	TenantID                 string
	Environment              string
	SkipProviderRegistration bool

	validateCredentialsOnce sync.Once
}

func (c *Config) validate() error {
	var err *multierror.Error

	if c.SubscriptionID == "" {
		err = multierror.Append(err, fmt.Errorf("Subscription ID must be configured for the AzureRM provider"))
	}
	if c.ClientID == "" {
		err = multierror.Append(err, fmt.Errorf("Client ID must be configured for the AzureRM provider"))
	}
	if c.ClientSecret == "" {
		err = multierror.Append(err, fmt.Errorf("Client Secret must be configured for the AzureRM provider"))
	}
	if c.TenantID == "" {
		err = multierror.Append(err, fmt.Errorf("Tenant ID must be configured for the AzureRM provider"))
	}
	if c.Environment == "" {
		err = multierror.Append(err, fmt.Errorf("Environment must be configured for the AzureRM provider"))
	}

	return err.ErrorOrNil()
}

func providerConfigure(p *schema.Provider) schema.ConfigureFunc {
	return func(d *schema.ResourceData) (interface{}, error) {
		config := &Config{
			SubscriptionID:           d.Get("subscription_id").(string),
			ClientID:                 d.Get("client_id").(string),
			ClientSecret:             d.Get("client_secret").(string),
			TenantID:                 d.Get("tenant_id").(string),
			Environment:              d.Get("environment").(string),
			SkipProviderRegistration: d.Get("skip_provider_registration").(bool),
		}

		if err := config.validate(); err != nil {
			return nil, err
		}

		client, err := config.getArmClient()
		if err != nil {
			return nil, err
		}

		client.StopContext = p.StopContext()

		// replaces the context between tests
		p.MetaReset = func() error {
			client.StopContext = p.StopContext()
			return nil
		}

		// List all the available providers and their registration state to avoid unnecessary
		// requests. This also lets us check if the provider credentials are correct.
		providerList, err := client.providers.List(nil, "")
		if err != nil {
			return nil, fmt.Errorf("Unable to list provider registration status, it is possible that this is due to invalid "+
				"credentials or the service principal does not have permission to use the Resource Manager API, Azure "+
				"error: %s", err)
		}

		if !config.SkipProviderRegistration {
			err = registerAzureResourceProvidersWithSubscription(*providerList.Value, client.providers)
			if err != nil {
				return nil, err
			}
		}

		return client, nil
	}
}

func registerProviderWithSubscription(providerName string, client resources.ProvidersClient) error {
	_, err := client.Register(providerName)
	if err != nil {
		return fmt.Errorf("Cannot register provider %s with Azure Resource Manager: %s.", providerName, err)
	}

	return nil
}

var providerRegistrationOnce sync.Once

// registerAzureResourceProvidersWithSubscription uses the providers client to register
// all Azure resource providers which the Terraform provider may require (regardless of
// whether they are actually used by the configuration or not). It was confirmed by Microsoft
// that this is the approach their own internal tools also take.
func registerAzureResourceProvidersWithSubscription(providerList []resources.Provider, client resources.ProvidersClient) error {
	var err error
	providerRegistrationOnce.Do(func() {
		providers := map[string]struct{}{
			"Microsoft.Compute":           struct{}{},
			"Microsoft.Cache":             struct{}{},
			"Microsoft.ContainerRegistry": struct{}{},
			"Microsoft.ContainerService":  struct{}{},
			"Microsoft.Network":           struct{}{},
			"Microsoft.Cdn":               struct{}{},
			"Microsoft.Storage":           struct{}{},
			"Microsoft.Sql":               struct{}{},
			"Microsoft.Search":            struct{}{},
			"Microsoft.Resources":         struct{}{},
			"Microsoft.ServiceBus":        struct{}{},
			"Microsoft.KeyVault":          struct{}{},
			"Microsoft.EventHub":          struct{}{},
		}

		// filter out any providers already registered
		for _, p := range providerList {
			if _, ok := providers[*p.Namespace]; !ok {
				continue
			}

			if strings.ToLower(*p.RegistrationState) == "registered" {
				log.Printf("[DEBUG] Skipping provider registration for namespace %s\n", *p.Namespace)
				delete(providers, *p.Namespace)
			}
		}

		var wg sync.WaitGroup
		wg.Add(len(providers))
		for providerName := range providers {
			go func(p string) {
				defer wg.Done()
				log.Printf("[DEBUG] Registering provider with namespace %s\n", p)
				if innerErr := registerProviderWithSubscription(p, client); err != nil {
					err = innerErr
				}
			}(providerName)
		}
		wg.Wait()
	})

	return err
}

// armMutexKV is the instance of MutexKV for ARM resources
var armMutexKV = mutexkv.NewMutexKV()

func azureStateRefreshFunc(resourceURI string, client *ArmClient, command riviera.APICall) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		req := client.rivieraClient.NewRequestForURI(resourceURI)
		req.Command = command

		res, err := req.Execute()
		if err != nil {
			return nil, "", fmt.Errorf("Error executing %T command in azureStateRefreshFunc", req.Command)
		}

		var value reflect.Value
		if reflect.ValueOf(res.Parsed).Kind() == reflect.Ptr {
			value = reflect.ValueOf(res.Parsed).Elem()
		} else {
			value = reflect.ValueOf(res.Parsed)
		}

		for i := 0; i < value.NumField(); i++ { // iterates through every struct type field
			tag := value.Type().Field(i).Tag // returns the tag string
			tagValue := tag.Get("mapstructure")
			if tagValue == "provisioningState" {
				return res.Parsed, value.Field(i).Elem().String(), nil
			}
		}

		panic(fmt.Errorf("azureStateRefreshFunc called on structure %T with no mapstructure:provisioningState tag. This is a bug", res.Parsed))
	}
}

// Resource group names can be capitalised, but we store them in lowercase.
// Use a custom diff function to avoid creation of new resources.
func resourceAzurermResourceGroupNameDiffSuppress(k, old, new string, d *schema.ResourceData) bool {
	return strings.ToLower(old) == strings.ToLower(new)
}

// ignoreCaseDiffSuppressFunc is a DiffSuppressFunc from helper/schema that is
// used to ignore any case-changes in a return value.
func ignoreCaseDiffSuppressFunc(k, old, new string, d *schema.ResourceData) bool {
	return strings.ToLower(old) == strings.ToLower(new)
}

// ignoreCaseStateFunc is a StateFunc from helper/schema that converts the
// supplied value to lower before saving to state for consistency.
func ignoreCaseStateFunc(val interface{}) string {
	return strings.ToLower(val.(string))
}

func userDataStateFunc(v interface{}) string {
	switch s := v.(type) {
	case string:
		s = base64Encode(s)
		hash := sha1.Sum([]byte(s))
		return hex.EncodeToString(hash[:])
	default:
		return ""
	}
}

// base64Encode encodes data if the input isn't already encoded using
// base64.StdEncoding.EncodeToString. If the input is already base64 encoded,
// return the original input unchanged.
func base64Encode(data string) string {
	// Check whether the data is already Base64 encoded; don't double-encode
	if isBase64Encoded(data) {
		return data
	}
	// data has not been encoded encode and return
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func isBase64Encoded(data string) bool {
	_, err := base64.StdEncoding.DecodeString(data)
	return err == nil
}
