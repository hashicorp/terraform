package azurerm

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/azure"
)

func resourceArmLoadBalancer() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLoadBalancerCreate,
		Read:   resourecArmLoadBalancerRead,
		Update: resourceArmLoadBalancerCreate,
		Delete: resourceArmLoadBalancerDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": locationSchema(),

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"frontend_ip_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
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
							Type:             schema.TypeString,
							Optional:         true,
							Computed:         true,
							ValidateFunc:     validateLoadBalancerPrivateIpAddressAllocation,
							StateFunc:        ignoreCaseStateFunc,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
						},

						"load_balancer_rules": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"inbound_nat_rules": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},

			"private_ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmLoadBalancerCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	loadBalancerClient := client.loadBalancerClient

	log.Printf("[INFO] preparing arguments for Azure ARM LoadBalancer creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	properties := network.LoadBalancerPropertiesFormat{}

	if _, ok := d.GetOk("frontend_ip_configuration"); ok {
		properties.FrontendIPConfigurations = expandAzureRmLoadBalancerFrontendIpConfigurations(d)
	}

	loadbalancer := network.LoadBalancer{
		Name:     azure.String(name),
		Location: azure.String(location),
		Tags:     expandedTags,
		LoadBalancerPropertiesFormat: &properties,
	}

	_, error := loadBalancerClient.CreateOrUpdate(resGroup, name, loadbalancer, make(chan struct{}))
	err := <-error
	if err != nil {
		return errwrap.Wrapf("Error Creating/Updating LoadBalancer {{err}}", err)
	}

	read, err := loadBalancerClient.Get(resGroup, name, "")
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer {{err}", err)
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read LoadBalancer %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	log.Printf("[DEBUG] Waiting for LoadBalancer (%s) to become available", name)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  []string{"Succeeded"},
		Refresh: loadbalancerStateRefreshFunc(client, resGroup, name),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for LoadBalancer (%s) to become available: %s", name, err)
	}

	return resourecArmLoadBalancerRead(d, meta)
}

func resourecArmLoadBalancerRead(d *schema.ResourceData, meta interface{}) error {
	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	loadBalancer, exists, err := retrieveLoadBalancerById(d.Id(), meta)
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer By ID {{err}}", err)
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] LoadBalancer %q not found. Removing from state", d.Get("name").(string))
		return nil
	}

	d.Set("name", loadBalancer.Name)
	d.Set("location", loadBalancer.Location)
	d.Set("resource_group_name", id.ResourceGroup)

	if loadBalancer.LoadBalancerPropertiesFormat != nil && loadBalancer.LoadBalancerPropertiesFormat.FrontendIPConfigurations != nil {
		ipconfigs := loadBalancer.LoadBalancerPropertiesFormat.FrontendIPConfigurations
		d.Set("frontend_ip_configuration", flattenLoadBalancerFrontendIpConfiguration(ipconfigs))

		for _, config := range *ipconfigs {
			if config.FrontendIPConfigurationPropertiesFormat.PrivateIPAddress != nil {
				d.Set("private_ip_address", config.FrontendIPConfigurationPropertiesFormat.PrivateIPAddress)

				// set the private IP address at most once
				break
			}
		}
	}

	flattenAndSetTags(d, loadBalancer.Tags)

	return nil
}

func resourceArmLoadBalancerDelete(d *schema.ResourceData, meta interface{}) error {
	loadBalancerClient := meta.(*ArmClient).loadBalancerClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return errwrap.Wrapf("Error Parsing Azure Resource ID {{err}}", err)
	}
	resGroup := id.ResourceGroup
	name := id.Path["loadBalancers"]

	_, error := loadBalancerClient.Delete(resGroup, name, make(chan struct{}))
	err = <-error
	if err != nil {
		return errwrap.Wrapf("Error Deleting LoadBalancer {{err}}", err)
	}

	d.SetId("")
	return nil
}

func expandAzureRmLoadBalancerFrontendIpConfigurations(d *schema.ResourceData) *[]network.FrontendIPConfiguration {
	configs := d.Get("frontend_ip_configuration").([]interface{})
	frontEndConfigs := make([]network.FrontendIPConfiguration, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		private_ip_allocation_method := data["private_ip_address_allocation"].(string)
		properties := network.FrontendIPConfigurationPropertiesFormat{
			PrivateIPAllocationMethod: network.IPAllocationMethod(private_ip_allocation_method),
		}

		if v := data["private_ip_address"].(string); v != "" {
			properties.PrivateIPAddress = &v
		}

		if v := data["public_ip_address_id"].(string); v != "" {
			properties.PublicIPAddress = &network.PublicIPAddress{
				ID: &v,
			}
		}

		if v := data["subnet_id"].(string); v != "" {
			properties.Subnet = &network.Subnet{
				ID: &v,
			}
		}

		name := data["name"].(string)
		frontEndConfig := network.FrontendIPConfiguration{
			Name: &name,
			FrontendIPConfigurationPropertiesFormat: &properties,
		}

		frontEndConfigs = append(frontEndConfigs, frontEndConfig)
	}

	return &frontEndConfigs
}

func flattenLoadBalancerFrontendIpConfiguration(ipConfigs *[]network.FrontendIPConfiguration) []interface{} {
	result := make([]interface{}, 0, len(*ipConfigs))
	for _, config := range *ipConfigs {
		ipConfig := make(map[string]interface{})
		ipConfig["name"] = *config.Name
		ipConfig["private_ip_address_allocation"] = config.FrontendIPConfigurationPropertiesFormat.PrivateIPAllocationMethod

		if config.FrontendIPConfigurationPropertiesFormat.Subnet != nil {
			ipConfig["subnet_id"] = *config.FrontendIPConfigurationPropertiesFormat.Subnet.ID
		}

		if config.FrontendIPConfigurationPropertiesFormat.PrivateIPAddress != nil {
			ipConfig["private_ip_address"] = *config.FrontendIPConfigurationPropertiesFormat.PrivateIPAddress
		}

		if config.FrontendIPConfigurationPropertiesFormat.PublicIPAddress != nil {
			ipConfig["public_ip_address_id"] = *config.FrontendIPConfigurationPropertiesFormat.PublicIPAddress.ID
		}

		if config.FrontendIPConfigurationPropertiesFormat.LoadBalancingRules != nil {
			load_balancing_rules := make([]string, 0, len(*config.FrontendIPConfigurationPropertiesFormat.LoadBalancingRules))
			for _, rule := range *config.FrontendIPConfigurationPropertiesFormat.LoadBalancingRules {
				load_balancing_rules = append(load_balancing_rules, *rule.ID)
			}

			ipConfig["load_balancer_rules"] = load_balancing_rules

		}

		if config.FrontendIPConfigurationPropertiesFormat.InboundNatRules != nil {
			inbound_nat_rules := make([]string, 0, len(*config.FrontendIPConfigurationPropertiesFormat.InboundNatRules))
			for _, rule := range *config.FrontendIPConfigurationPropertiesFormat.InboundNatRules {
				inbound_nat_rules = append(inbound_nat_rules, *rule.ID)
			}

			ipConfig["inbound_nat_rules"] = inbound_nat_rules

		}

		result = append(result, ipConfig)
	}
	return result
}
