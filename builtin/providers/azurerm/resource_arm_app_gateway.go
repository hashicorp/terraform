package azurerm

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"path"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/jen20/riviera/azure"
)

func resourceArmAppGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmAppGatewayCreate,
		Read:   resourceArmAppGatewayRead,
		Update: resourceArmAppGatewayCreate,
		Delete: resourceArmAppGatewayDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": {
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"sku": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.StandardSmall),
								string(network.StandardMedium),
								string(network.StandardLarge),
								string(network.WAFLarge),
								string(network.WAFMedium),
							}, true),
						},

						"tier": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.Standard),
								string(network.WAF),
							}, false),
						},

						"capacity": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntBetween(1, 10),
						},
					},
				},
				Set: hashAppGatewaySku,
			},

			"disabled_ssl_protocols": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						string(network.TLSv10),
						string(network.TLSv11),
						string(network.TLSv12),
					}, false),
				},
			},

			"waf_configuration": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Required: true,
						},

						"firewall_mode": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.Detection),
								string(network.Prevention),
							}, false),
						},
					},
				},
				Set: hashAppGatewayWafConfig,
			},

			"gateway_ip_configuration": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"subnet_id": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"frontend_port": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"port": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},

			"frontend_ip_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"subnet_id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"private_ip_address": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"public_ip_address_id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"private_ip_address_allocation": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.Dynamic),
								string(network.Static),
							}, false),
						},
					},
				},
			},

			"backend_address_pool": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"ip_address_list": {
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},

						"fqdn_list": {
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},

			"backend_http_settings": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"port": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"protocol": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.HTTP),
								string(network.HTTPS),
							}, false),
						},

						"cookie_based_affinity": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.Enabled),
								string(network.Disabled),
							}, false),
						},

						"request_timeout": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"probe_name": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"probe_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"http_listener": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"frontend_ip_configuration_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"frontend_ip_configuration_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"frontend_port_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"frontend_port_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"protocol": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.HTTP),
								string(network.HTTPS),
							}, false),
						},

						"host_name": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"ssl_certificate_name": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"ssl_certificate_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"require_sni": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},

			"probe": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"protocol": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.HTTP),
								string(network.HTTPS),
							}, false),
						},

						"path": {
							Type:     schema.TypeString,
							Required: true,
						},

						"host": {
							Type:     schema.TypeString,
							Required: true,
						},

						"interval": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"timeout": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"unhealthy_threshold": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},

			"request_routing_rule": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"rule_type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.Basic),
								string(network.PathBasedRouting),
							}, false),
						},

						"http_listener_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"http_listener_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"backend_address_pool_name": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"backend_address_pool_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"backend_http_settings_name": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"backend_http_settings_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"url_path_map_name": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"url_path_map_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"url_path_map": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"default_backend_address_pool_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"default_backend_address_pool_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"default_backend_http_settings_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"default_backend_http_settings_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"path_rule": {
							Type:     schema.TypeList,
							Required: true,
							MinItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:     schema.TypeString,
										Computed: true,
									},

									"name": {
										Type:     schema.TypeString,
										Required: true,
									},

									"paths": {
										Type:     schema.TypeList,
										Required: true,
										MinItems: 1,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},

									"backend_address_pool_name": {
										Type:     schema.TypeString,
										Required: true,
									},

									"backend_address_pool_id": {
										Type:     schema.TypeString,
										Computed: true,
									},

									"backend_http_settings_name": {
										Type:     schema.TypeString,
										Required: true,
									},

									"backend_http_settings_id": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},

			"ssl_certificate": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"data": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},

						"password": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},

						"public_cert_data": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmAppGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)
	client := armClient.appGatewayClient

	log.Printf("[INFO] preparing arguments for AzureRM AppGateway creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})

	// Gateway ID is needed to link subresources together in expand functions
	gatewayID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/applicationGateways/%s",
		armClient.subscriptionId, resGroup, name)

	properties := network.ApplicationGatewayPropertiesFormat{}
	properties.Sku = expandAppGatewaySku(d)
	properties.SslPolicy = expandAppGatewaySslPolicy(d)
	properties.GatewayIPConfigurations = expandAppGatewayIPConfigurations(d)
	properties.FrontendPorts = expandAppGatewayFrontendPorts(d)
	properties.FrontendIPConfigurations = expandAppGatewayFrontendIPConfigurations(d)
	properties.BackendAddressPools = expandAppGatewayBackendAddressPools(d)
	properties.BackendHTTPSettingsCollection = expandAppGatewayBackendHTTPSettings(d, gatewayID)
	properties.HTTPListeners = expandAppGatewayHTTPListeners(d, gatewayID)
	properties.Probes = expandAppGatewayProbes(d)
	properties.RequestRoutingRules = expandAppGatewayRequestRoutingRules(d, gatewayID)
	properties.URLPathMaps = expandAppGatewayURLPathMaps(d, gatewayID)
	properties.SslCertificates = expandAppGatewaySslCertificates(d)

	if _, ok := d.GetOk("waf_configuration"); ok {
		properties.WebApplicationFirewallConfiguration = expandAppGatewayWafConfig(d)
	}

	gateway := network.ApplicationGateway{
		Name:       azure.String(name),
		Location:   azure.String(location),
		Tags:       expandTags(tags),
		Properties: &properties,
	}

	_, err := client.CreateOrUpdate(resGroup, name, gateway, make(chan struct{}))
	if err != nil {
		return errwrap.Wrapf("Error Creating/Updating AppGateway {{err}}", err)
	}

	read, err := client.Get(resGroup, name)
	if err != nil {
		return errwrap.Wrapf("Error Getting AppGateway {{err}", err)
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read AppGateway %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	log.Printf("[DEBUG] Waiting for AppGateway (%s) to become available", name)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  []string{"Succeeded"},
		Refresh: appGatewayStateRefreshFunc(meta.(*ArmClient), resGroup, name),
		Timeout: 60 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for AppGateway (%s) to become available: %s", name, err)
	}

	return resourceArmAppGatewayRead(d, meta)
}

func resourceArmAppGatewayRead(d *schema.ResourceData, meta interface{}) error {
	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return errwrap.Wrapf("Error parsing AppGateway ID {{err}}", err)
	}

	appGateway, exists, err := retrieveAppGatewayById(d.Id(), meta)
	if err != nil {
		return errwrap.Wrapf("Error Getting AppGateway By ID {{err}}", err)
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] AppGateway %q not found. Removing from state", d.Get("name").(string))
		return nil
	}

	d.Set("name", appGateway.Name)
	d.Set("resource_group_name", id.ResourceGroup)
	d.Set("location", appGateway.Location)
	d.Set("sku", schema.NewSet(hashAppGatewaySku, flattenAppGatewaySku(appGateway.Properties.Sku)))
	d.Set("disabled_ssl_protocols", flattenAppGatewaySslPolicy(appGateway.Properties.SslPolicy))
	d.Set("gateway_ip_configuration", flattenAppGatewayIPConfigurations(appGateway.Properties.GatewayIPConfigurations))
	d.Set("frontend_port", flattenAppGatewayFrontendPorts(appGateway.Properties.FrontendPorts))
	d.Set("frontend_ip_configuration", flattenAppGatewayFrontendIPConfigurations(appGateway.Properties.FrontendIPConfigurations))
	d.Set("backend_address_pool", flattenAppGatewayBackendAddressPools(appGateway.Properties.BackendAddressPools))
	d.Set("backend_http_settings", flattenAppGatewayBackendHTTPSettings(appGateway.Properties.BackendHTTPSettingsCollection))
	d.Set("http_listener", flattenAppGatewayHTTPListeners(appGateway.Properties.HTTPListeners))
	d.Set("probe", flattenAppGatewayProbes(appGateway.Properties.Probes))
	d.Set("request_routing_rule", flattenAppGatewayRequestRoutingRules(appGateway.Properties.RequestRoutingRules))
	d.Set("url_path_map", flattenAppGatewayURLPathMaps(appGateway.Properties.URLPathMaps))
	d.Set("ssl_certificate", flattenAppGatewaySslCertificates(appGateway.Properties.SslCertificates))

	if appGateway.Properties.WebApplicationFirewallConfiguration != nil {
		d.Set("waf_configuration", schema.NewSet(hashAppGatewayWafConfig,
			flattenAppGatewayWafConfig(appGateway.Properties.WebApplicationFirewallConfiguration)))
	}

	flattenAndSetTags(d, appGateway.Tags)

	return nil
}

func resourceArmAppGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).appGatewayClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return errwrap.Wrapf("Error Parsing Azure Resource ID {{err}}", err)
	}
	resGroup := id.ResourceGroup
	name := id.Path["applicationGateways"]

	_, err = client.Delete(resGroup, name, make(chan struct{}))
	if err != nil {
		return errwrap.Wrapf("Error Deleting AppGateway {{err}}", err)
	}

	d.SetId("")
	return nil
}

func appGatewayResGroupAndNameFromID(appGatewayID string) (string, string, error) {
	id, err := parseAzureResourceID(appGatewayID)
	if err != nil {
		return "", "", err
	}
	name := id.Path["applicationGateways"]
	resGroup := id.ResourceGroup

	return resGroup, name, nil
}

func retrieveAppGatewayById(appGatewayID string, meta interface{}) (*network.ApplicationGateway, bool, error) {
	client := meta.(*ArmClient).appGatewayClient

	resGroup, name, err := appGatewayResGroupAndNameFromID(appGatewayID)
	if err != nil {
		return nil, false, errwrap.Wrapf("Error Getting AppGateway Name and Group: {{err}}", err)
	}

	resp, err := client.Get(resGroup, name)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("Error making Read request on Azure AppGateway %s: %s", name, err)
	}

	return &resp, true, nil
}

func appGatewayStateRefreshFunc(client *ArmClient, resourceGroupName string, name string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.appGatewayClient.Get(resourceGroupName, name)
		if err != nil {
			return nil, "", fmt.Errorf(
				"Error issuing read request in appGatewayStateRefreshFunc to Azure ARM for AppGateway '%s' (RG: '%s'): %s",
				name, resourceGroupName, err)
		}

		return res, *res.Properties.ProvisioningState, nil
	}
}

func expandAppGatewaySku(d *schema.ResourceData) *network.ApplicationGatewaySku {
	skuSet := d.Get("sku").(*schema.Set).List()
	sku := skuSet[0].(map[string]interface{})

	name := sku["name"].(string)
	tier := sku["tier"].(string)
	capacity := int32(sku["capacity"].(int))

	return &network.ApplicationGatewaySku{
		Name:     network.ApplicationGatewaySkuName(name),
		Tier:     network.ApplicationGatewayTier(tier),
		Capacity: &capacity,
	}
}

func expandAppGatewayWafConfig(d *schema.ResourceData) *network.ApplicationGatewayWebApplicationFirewallConfiguration {
	wafSet := d.Get("waf_configuration").(*schema.Set).List()
	waf := wafSet[0].(map[string]interface{})

	enabled := waf["enabled"].(bool)
	mode := waf["firewall_mode"].(string)

	return &network.ApplicationGatewayWebApplicationFirewallConfiguration{
		Enabled:      &enabled,
		FirewallMode: network.ApplicationGatewayFirewallMode(mode),
	}
}

func expandAppGatewaySslPolicy(d *schema.ResourceData) *network.ApplicationGatewaySslPolicy {
	disabledProtoList := d.Get("disabled_ssl_protocols").([]interface{})
	disabled := []network.ApplicationGatewaySslProtocol{}

	for _, proto := range disabledProtoList {
		disabled = append(disabled, network.ApplicationGatewaySslProtocol(proto.(string)))
	}

	return &network.ApplicationGatewaySslPolicy{
		DisabledSslProtocols: &disabled,
	}
}

func expandAppGatewayIPConfigurations(d *schema.ResourceData) *[]network.ApplicationGatewayIPConfiguration {
	configs := d.Get("gateway_ip_configuration").([]interface{})
	ipConfigurations := make([]network.ApplicationGatewayIPConfiguration, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		subnetID := data["subnet_id"].(string)

		ipConfig := network.ApplicationGatewayIPConfiguration{
			Name: &name,
			Properties: &network.ApplicationGatewayIPConfigurationPropertiesFormat{
				Subnet: &network.SubResource{
					ID: &subnetID,
				},
			},
		}
		ipConfigurations = append(ipConfigurations, ipConfig)
	}

	return &ipConfigurations
}

func expandAppGatewayFrontendPorts(d *schema.ResourceData) *[]network.ApplicationGatewayFrontendPort {
	configs := d.Get("frontend_port").([]interface{})
	frontendPorts := make([]network.ApplicationGatewayFrontendPort, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		port := int32(data["port"].(int))

		portConfig := network.ApplicationGatewayFrontendPort{
			Name: &name,
			Properties: &network.ApplicationGatewayFrontendPortPropertiesFormat{
				Port: &port,
			},
		}
		frontendPorts = append(frontendPorts, portConfig)
	}

	return &frontendPorts
}

func expandAppGatewayFrontendIPConfigurations(d *schema.ResourceData) *[]network.ApplicationGatewayFrontendIPConfiguration {
	configs := d.Get("frontend_ip_configuration").([]interface{})
	frontEndConfigs := make([]network.ApplicationGatewayFrontendIPConfiguration, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		properties := network.ApplicationGatewayFrontendIPConfigurationPropertiesFormat{}

		if v := data["subnet_id"].(string); v != "" {
			properties.Subnet = &network.SubResource{
				ID: &v,
			}
		}

		if v := data["private_ip_address_allocation"].(string); v != "" {
			properties.PrivateIPAllocationMethod = network.IPAllocationMethod(v)
		}

		if v := data["private_ip_address"].(string); v != "" {
			properties.PrivateIPAddress = &v
		}

		if v := data["public_ip_address_id"].(string); v != "" {
			properties.PublicIPAddress = &network.SubResource{
				ID: &v,
			}
		}

		name := data["name"].(string)
		frontEndConfig := network.ApplicationGatewayFrontendIPConfiguration{
			Name:       &name,
			Properties: &properties,
		}

		frontEndConfigs = append(frontEndConfigs, frontEndConfig)
	}

	return &frontEndConfigs
}

func expandAppGatewayBackendAddressPools(d *schema.ResourceData) *[]network.ApplicationGatewayBackendAddressPool {
	configs := d.Get("backend_address_pool").([]interface{})
	backendPools := make([]network.ApplicationGatewayBackendAddressPool, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		backendAddresses := []network.ApplicationGatewayBackendAddress{}

		for _, rawIP := range data["ip_address_list"].([]interface{}) {
			ip := rawIP.(string)
			backendAddresses = append(backendAddresses, network.ApplicationGatewayBackendAddress{IPAddress: &ip})
		}

		for _, rawFQDN := range data["fqdn_list"].([]interface{}) {
			fqdn := rawFQDN.(string)
			backendAddresses = append(backendAddresses, network.ApplicationGatewayBackendAddress{Fqdn: &fqdn})
		}

		name := data["name"].(string)
		pool := network.ApplicationGatewayBackendAddressPool{
			Name: &name,
			Properties: &network.ApplicationGatewayBackendAddressPoolPropertiesFormat{
				BackendAddresses: &backendAddresses,
			},
		}

		backendPools = append(backendPools, pool)
	}

	return &backendPools
}

func expandAppGatewayBackendHTTPSettings(d *schema.ResourceData, gatewayID string) *[]network.ApplicationGatewayBackendHTTPSettings {
	configs := d.Get("backend_http_settings").([]interface{})
	backendSettings := make([]network.ApplicationGatewayBackendHTTPSettings, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		port := int32(data["port"].(int))
		protocol := data["protocol"].(string)
		cookieBasedAffinity := data["cookie_based_affinity"].(string)
		requestTimeout := int32(data["request_timeout"].(int))

		setting := network.ApplicationGatewayBackendHTTPSettings{
			Name: &name,
			Properties: &network.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
				Port:                &port,
				Protocol:            network.ApplicationGatewayProtocol(protocol),
				CookieBasedAffinity: network.ApplicationGatewayCookieBasedAffinity(cookieBasedAffinity),
				RequestTimeout:      &requestTimeout,
			},
		}

		probeName := data["probe_name"].(string)
		if probeName != "" {
			probeID := fmt.Sprintf("%s/probes/%s", gatewayID, probeName)
			setting.Properties.Probe = &network.SubResource{
				ID: &probeID,
			}
		}

		backendSettings = append(backendSettings, setting)
	}

	return &backendSettings
}

func expandAppGatewayHTTPListeners(d *schema.ResourceData, gatewayID string) *[]network.ApplicationGatewayHTTPListener {
	configs := d.Get("http_listener").([]interface{})
	httpListeners := make([]network.ApplicationGatewayHTTPListener, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		frontendIPConfigName := data["frontend_ip_configuration_name"].(string)
		frontendIPConfigID := fmt.Sprintf("%s/frontendIPConfigurations/%s", gatewayID, frontendIPConfigName)
		frontendPortName := data["frontend_port_name"].(string)
		frontendPortID := fmt.Sprintf("%s/frontendPorts/%s", gatewayID, frontendPortName)
		protocol := data["protocol"].(string)

		listener := network.ApplicationGatewayHTTPListener{
			Name: &name,
			Properties: &network.ApplicationGatewayHTTPListenerPropertiesFormat{
				FrontendIPConfiguration: &network.SubResource{
					ID: &frontendIPConfigID,
				},
				FrontendPort: &network.SubResource{
					ID: &frontendPortID,
				},
				Protocol: network.ApplicationGatewayProtocol(protocol),
			},
		}

		if host := data["host_name"].(string); host != "" {
			listener.Properties.HostName = &host
		}

		if sslCertName := data["ssl_certificate_name"].(string); sslCertName != "" {
			certID := fmt.Sprintf("%s/sslCertificates/%s", gatewayID, sslCertName)
			listener.Properties.SslCertificate = &network.SubResource{
				ID: &certID,
			}
		}

		if requireSNI, ok := data["require_sni"].(bool); ok {
			listener.Properties.RequireServerNameIndication = &requireSNI
		}

		httpListeners = append(httpListeners, listener)
	}

	return &httpListeners
}

func expandAppGatewayProbes(d *schema.ResourceData) *[]network.ApplicationGatewayProbe {
	configs := d.Get("probe").([]interface{})
	backendSettings := make([]network.ApplicationGatewayProbe, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		protocol := data["protocol"].(string)
		path := data["path"].(string)
		host := data["host"].(string)
		interval := int32(data["interval"].(int))
		timeout := int32(data["timeout"].(int))
		unhealthyThreshold := int32(data["unhealthy_threshold"].(int))

		setting := network.ApplicationGatewayProbe{
			Name: &name,
			Properties: &network.ApplicationGatewayProbePropertiesFormat{
				Protocol:           network.ApplicationGatewayProtocol(protocol),
				Path:               &path,
				Host:               &host,
				Interval:           &interval,
				Timeout:            &timeout,
				UnhealthyThreshold: &unhealthyThreshold,
			},
		}

		backendSettings = append(backendSettings, setting)
	}

	return &backendSettings
}

func expandAppGatewayRequestRoutingRules(d *schema.ResourceData, gatewayID string) *[]network.ApplicationGatewayRequestRoutingRule {
	configs := d.Get("request_routing_rule").([]interface{})
	rules := make([]network.ApplicationGatewayRequestRoutingRule, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		ruleType := data["rule_type"].(string)
		httpListenerName := data["http_listener_name"].(string)
		httpListenerID := fmt.Sprintf("%s/httpListeners/%s", gatewayID, httpListenerName)

		rule := network.ApplicationGatewayRequestRoutingRule{
			Name: &name,
			Properties: &network.ApplicationGatewayRequestRoutingRulePropertiesFormat{
				RuleType: network.ApplicationGatewayRequestRoutingRuleType(ruleType),
				HTTPListener: &network.SubResource{
					ID: &httpListenerID,
				},
			},
		}

		if backendAddressPoolName := data["backend_address_pool_name"].(string); backendAddressPoolName != "" {
			backendAddressPoolID := fmt.Sprintf("%s/backendAddressPools/%s", gatewayID, backendAddressPoolName)
			rule.Properties.BackendAddressPool = &network.SubResource{
				ID: &backendAddressPoolID,
			}
		}

		if backendHTTPSettingsName := data["backend_http_settings_name"].(string); backendHTTPSettingsName != "" {
			backendHTTPSettingsID := fmt.Sprintf("%s/backendHttpSettingsCollection/%s", gatewayID, backendHTTPSettingsName)
			rule.Properties.BackendHTTPSettings = &network.SubResource{
				ID: &backendHTTPSettingsID,
			}
		}

		if urlPathMapName := data["url_path_map_name"].(string); urlPathMapName != "" {
			urlPathMapID := fmt.Sprintf("%s/urlPathMaps/%s", gatewayID, urlPathMapName)
			rule.Properties.URLPathMap = &network.SubResource{
				ID: &urlPathMapID,
			}
		}

		rules = append(rules, rule)
	}

	return &rules
}

func expandAppGatewayURLPathMaps(d *schema.ResourceData, gatewayID string) *[]network.ApplicationGatewayURLPathMap {
	configs := d.Get("url_path_map").([]interface{})
	pathMaps := make([]network.ApplicationGatewayURLPathMap, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		defaultBackendAddressPoolName := data["default_backend_address_pool_name"].(string)
		defaultBackendAddressPoolID := fmt.Sprintf("%s/backendAddressPools/%s", gatewayID, defaultBackendAddressPoolName)
		defaultBackendHTTPSettingsName := data["default_backend_http_settings_name"].(string)
		defaultBackendHTTPSettingsID := fmt.Sprintf("%s/backendHttpSettingsCollection/%s", gatewayID, defaultBackendHTTPSettingsName)

		pathRules := []network.ApplicationGatewayPathRule{}
		for _, ruleConfig := range data["path_rule"].([]interface{}) {
			ruleConfigMap := ruleConfig.(map[string]interface{})

			ruleName := ruleConfigMap["name"].(string)

			rulePaths := []string{}
			for _, rulePath := range ruleConfigMap["paths"].([]interface{}) {
				rulePaths = append(rulePaths, rulePath.(string))
			}

			rule := network.ApplicationGatewayPathRule{
				Name: &ruleName,
				Properties: &network.ApplicationGatewayPathRulePropertiesFormat{
					Paths: &rulePaths,
				},
			}

			if backendAddressPoolName := ruleConfigMap["backend_address_pool_name"].(string); backendAddressPoolName != "" {
				backendAddressPoolID := fmt.Sprintf("%s/backendAddressPools/%s", gatewayID, backendAddressPoolName)
				rule.Properties.BackendAddressPool = &network.SubResource{
					ID: &backendAddressPoolID,
				}
			}

			if backendHTTPSettingsName := ruleConfigMap["backend_http_settings_name"].(string); backendHTTPSettingsName != "" {
				backendHTTPSettingsID := fmt.Sprintf("%s/backendHttpSettingsCollection/%s", gatewayID, backendHTTPSettingsName)
				rule.Properties.BackendHTTPSettings = &network.SubResource{
					ID: &backendHTTPSettingsID,
				}
			}

			pathRules = append(pathRules, rule)
		}

		pathMap := network.ApplicationGatewayURLPathMap{
			Name: &name,
			Properties: &network.ApplicationGatewayURLPathMapPropertiesFormat{
				DefaultBackendAddressPool: &network.SubResource{
					ID: &defaultBackendAddressPoolID,
				},
				DefaultBackendHTTPSettings: &network.SubResource{
					ID: &defaultBackendHTTPSettingsID,
				},
				PathRules: &pathRules,
			},
		}

		pathMaps = append(pathMaps, pathMap)
	}

	return &pathMaps
}

func expandAppGatewaySslCertificates(d *schema.ResourceData) *[]network.ApplicationGatewaySslCertificate {
	configs := d.Get("ssl_certificate").([]interface{})
	sslCerts := make([]network.ApplicationGatewaySslCertificate, 0, len(configs))

	for _, configRaw := range configs {
		raw := configRaw.(map[string]interface{})

		name := raw["name"].(string)
		data := raw["data"].(string)
		password := raw["password"].(string)

		// data must be base64 encoded
		data = base64.StdEncoding.EncodeToString([]byte(data))

		cert := network.ApplicationGatewaySslCertificate{
			Name: &name,
			Properties: &network.ApplicationGatewaySslCertificatePropertiesFormat{
				Data:     &data,
				Password: &password,
			},
		}

		sslCerts = append(sslCerts, cert)
	}

	return &sslCerts
}

func flattenAppGatewaySku(sku *network.ApplicationGatewaySku) []interface{} {
	result := make(map[string]interface{})

	result["name"] = string(sku.Name)
	result["tier"] = string(sku.Tier)
	result["capacity"] = int(*sku.Capacity)

	return []interface{}{result}
}

func flattenAppGatewayWafConfig(waf *network.ApplicationGatewayWebApplicationFirewallConfiguration) []interface{} {
	result := make(map[string]interface{})

	result["enabled"] = *waf.Enabled
	result["firewall_mode"] = string(waf.FirewallMode)

	return []interface{}{result}
}

func flattenAppGatewaySslPolicy(policy *network.ApplicationGatewaySslPolicy) []interface{} {
	result := make([]interface{}, 0, len(*policy.DisabledSslProtocols))

	for _, proto := range *policy.DisabledSslProtocols {
		result = append(result, string(proto))
	}

	return result
}

func flattenAppGatewayIPConfigurations(ipConfigs *[]network.ApplicationGatewayIPConfiguration) []interface{} {
	result := make([]interface{}, 0, len(*ipConfigs))

	for _, config := range *ipConfigs {
		ipConfig := map[string]interface{}{
			"id":        *config.ID,
			"name":      *config.Name,
			"subnet_id": *config.Properties.Subnet.ID,
		}

		result = append(result, ipConfig)
	}

	return result
}

func flattenAppGatewayFrontendPorts(portConfigs *[]network.ApplicationGatewayFrontendPort) []interface{} {
	result := make([]interface{}, 0, len(*portConfigs))

	for _, config := range *portConfigs {
		port := map[string]interface{}{
			"id":   *config.ID,
			"name": *config.Name,
			"port": int(*config.Properties.Port),
		}

		result = append(result, port)
	}

	return result
}

func flattenAppGatewayFrontendIPConfigurations(ipConfigs *[]network.ApplicationGatewayFrontendIPConfiguration) []interface{} {
	result := make([]interface{}, 0, len(*ipConfigs))
	for _, config := range *ipConfigs {
		ipConfig := make(map[string]interface{})
		ipConfig["id"] = *config.ID
		ipConfig["name"] = *config.Name

		if config.Properties.PrivateIPAllocationMethod != "" {
			ipConfig["private_ip_address_allocation"] = config.Properties.PrivateIPAllocationMethod
		}

		if config.Properties.Subnet != nil {
			ipConfig["subnet_id"] = *config.Properties.Subnet.ID
		}

		if config.Properties.PrivateIPAddress != nil {
			ipConfig["private_ip_address"] = *config.Properties.PrivateIPAddress
		}

		if config.Properties.PublicIPAddress != nil {
			ipConfig["public_ip_address_id"] = *config.Properties.PublicIPAddress.ID
		}

		result = append(result, ipConfig)
	}
	return result
}

func flattenAppGatewayBackendAddressPools(poolConfigs *[]network.ApplicationGatewayBackendAddressPool) []interface{} {
	result := make([]interface{}, 0, len(*poolConfigs))

	for _, config := range *poolConfigs {
		ipAddressList := []interface{}{}
		fqdnList := []interface{}{}
		for _, address := range *config.Properties.BackendAddresses {
			if address.IPAddress != nil {
				ipAddressList = append(ipAddressList, *address.IPAddress)
			} else if address.Fqdn != nil {
				fqdnList = append(fqdnList, *address.Fqdn)
			}
		}

		pool := map[string]interface{}{
			"id":              *config.ID,
			"name":            *config.Name,
			"ip_address_list": ipAddressList,
			"fqdn_list":       fqdnList,
		}

		result = append(result, pool)
	}

	return result
}

func flattenAppGatewayBackendHTTPSettings(backendSettings *[]network.ApplicationGatewayBackendHTTPSettings) []interface{} {
	result := make([]interface{}, 0, len(*backendSettings))

	for _, config := range *backendSettings {
		settings := map[string]interface{}{
			"id":                    *config.ID,
			"name":                  *config.Name,
			"port":                  int(*config.Properties.Port),
			"protocol":              string(config.Properties.Protocol),
			"cookie_based_affinity": string(config.Properties.CookieBasedAffinity),
			"request_timeout":       int(*config.Properties.RequestTimeout),
		}

		if config.Properties.Probe != nil {
			settings["probe_name"] = path.Base(*config.Properties.Probe.ID)
			settings["probe_id"] = *config.Properties.Probe.ID
		}

		result = append(result, settings)
	}

	return result
}

func flattenAppGatewayHTTPListeners(httpListeners *[]network.ApplicationGatewayHTTPListener) []interface{} {
	result := make([]interface{}, 0, len(*httpListeners))

	for _, config := range *httpListeners {
		listener := map[string]interface{}{
			"id":   *config.ID,
			"name": *config.Name,
			"frontend_ip_configuration_id":   *config.Properties.FrontendIPConfiguration.ID,
			"frontend_ip_configuration_name": path.Base(*config.Properties.FrontendIPConfiguration.ID),
			"frontend_port_name":             path.Base(*config.Properties.FrontendPort.ID),
			"frontend_port_id":               *config.Properties.FrontendPort.ID,
			"protocol":                       string(config.Properties.Protocol),
		}

		if config.Properties.HostName != nil {
			listener["host_name"] = *config.Properties.HostName
		}

		if config.Properties.SslCertificate != nil {
			listener["ssl_certificate_name"] = path.Base(*config.Properties.SslCertificate.ID)
			listener["ssl_certificate_id"] = *config.Properties.SslCertificate.ID
		}

		if config.Properties.RequireServerNameIndication != nil {
			listener["require_sni"] = *config.Properties.RequireServerNameIndication
		}

		result = append(result, listener)
	}

	return result
}

func flattenAppGatewayProbes(probes *[]network.ApplicationGatewayProbe) []interface{} {
	result := make([]interface{}, 0, len(*probes))

	for _, config := range *probes {
		settings := map[string]interface{}{
			"id":                  *config.ID,
			"name":                *config.Name,
			"protocol":            string(config.Properties.Protocol),
			"path":                *config.Properties.Path,
			"host":                *config.Properties.Host,
			"interval":            int(*config.Properties.Interval),
			"timeout":             int(*config.Properties.Timeout),
			"unhealthy_threshold": int(*config.Properties.UnhealthyThreshold),
		}

		result = append(result, settings)
	}

	return result
}

func flattenAppGatewayRequestRoutingRules(rules *[]network.ApplicationGatewayRequestRoutingRule) []interface{} {
	result := make([]interface{}, 0, len(*rules))

	for _, config := range *rules {
		listener := map[string]interface{}{
			"id":                 *config.ID,
			"name":               *config.Name,
			"rule_type":          string(config.Properties.RuleType),
			"http_listener_id":   *config.Properties.HTTPListener.ID,
			"http_listener_name": path.Base(*config.Properties.HTTPListener.ID),
		}

		if config.Properties.BackendAddressPool != nil {
			listener["backend_address_pool_name"] = path.Base(*config.Properties.BackendAddressPool.ID)
			listener["backend_address_pool_id"] = *config.Properties.BackendAddressPool.ID
		}

		if config.Properties.BackendHTTPSettings != nil {
			listener["backend_http_settings_name"] = path.Base(*config.Properties.BackendHTTPSettings.ID)
			listener["backend_http_settings_id"] = *config.Properties.BackendHTTPSettings.ID
		}

		if config.Properties.URLPathMap != nil {
			listener["url_path_map_name"] = path.Base(*config.Properties.URLPathMap.ID)
			listener["url_path_map_id"] = *config.Properties.URLPathMap.ID
		}

		result = append(result, listener)
	}

	return result
}

func flattenAppGatewayURLPathMaps(pathMaps *[]network.ApplicationGatewayURLPathMap) []interface{} {
	result := make([]interface{}, 0, len(*pathMaps))

	for _, config := range *pathMaps {
		pathMap := map[string]interface{}{
			"id":   *config.ID,
			"name": *config.Name,
		}

		if config.Properties.DefaultBackendAddressPool != nil {
			pathMap["default_backend_address_pool_name"] = path.Base(*config.Properties.DefaultBackendAddressPool.ID)
			pathMap["default_backend_address_pool_id"] = *config.Properties.DefaultBackendAddressPool.ID
		}

		if config.Properties.DefaultBackendHTTPSettings != nil {
			pathMap["default_backend_http_settings_name"] = path.Base(*config.Properties.DefaultBackendHTTPSettings.ID)
			pathMap["default_backend_http_settings_id"] = *config.Properties.DefaultBackendHTTPSettings.ID
		}

		pathRules := make([]interface{}, 0, len(*config.Properties.PathRules))
		for _, pathRuleConfig := range *config.Properties.PathRules {
			rule := map[string]interface{}{
				"id":   *pathRuleConfig.ID,
				"name": *pathRuleConfig.Name,
			}

			if pathRuleConfig.Properties.BackendAddressPool != nil {
				rule["backend_address_pool_name"] = path.Base(*pathRuleConfig.Properties.BackendAddressPool.ID)
				rule["backend_address_pool_id"] = *pathRuleConfig.Properties.BackendAddressPool.ID
			}

			if pathRuleConfig.Properties.BackendHTTPSettings != nil {
				rule["backend_http_settings_name"] = path.Base(*pathRuleConfig.Properties.BackendHTTPSettings.ID)
				rule["backend_http_settings_id"] = *pathRuleConfig.Properties.BackendHTTPSettings.ID
			}

			paths := make([]interface{}, 0, len(*pathRuleConfig.Properties.Paths))
			for _, path := range *pathRuleConfig.Properties.Paths {
				paths = append(paths, path)
			}
			rule["paths"] = paths

			pathRules = append(pathRules, rule)
		}
		pathMap["path_rule"] = pathRules

		result = append(result, pathMap)
	}

	return result
}

func flattenAppGatewaySslCertificates(certs *[]network.ApplicationGatewaySslCertificate) []interface{} {
	result := make([]interface{}, 0, len(*certs))

	for _, config := range *certs {
		certConfig := map[string]interface{}{
			"id":               *config.ID,
			"name":             *config.Name,
			"data":             *config.Properties.Data,
			"password":         *config.Properties.Password,
			"public_cert_data": *config.Properties.PublicCertData,
		}

		result = append(result, certConfig)
	}

	return result
}

func hashAppGatewaySku(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["tier"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["capacity"].(int)))

	return hashcode.String(buf.String())
}

func hashAppGatewayWafConfig(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%t-", m["enabled"].(bool)))
	buf.WriteString(fmt.Sprintf("%s-", m["firewall_mode"].(string)))

	return hashcode.String(buf.String())
}
