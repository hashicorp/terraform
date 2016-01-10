package azurerm

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/azure"
	"log"
	"time"
)

func resourceArmLoadbalancerRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLoadbalancerRuleCreate,
		Read:   resourceArmLoadbalancerRuleRead,
		Update: resourceArmLoadbalancerRuleCreate,
		Delete: resourceArmLoadbalancerRuleDelete,

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

			"loadbalancer_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"frontend_ip_configuration_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"frontend_ip_configuration_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"backend_address_pool_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"protocol": {
				Type:     schema.TypeString,
				Required: true,
			},

			"frontend_port": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"backend_port": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"probe_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"enable_floating_ip": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"idle_timeout_in_minutes": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"load_distribution": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceArmLoadbalancerRuleCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	lbClient := client.loadBalancerClient

	loadBalancer, exists, err := retrieveLoadbalancerById(d.Get("loadbalancer_id").(string), meta)
	if err != nil {
		return err
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] Loadbalancer %q not found. Refreshing from state", d.Get("name").(string))
		return nil
	}

	_, _, exists = findLoadBalancerRuleByName(loadBalancer, d.Get("name").(string))
	if exists {
		return fmt.Errorf("A Nat Rule with name %q already exists.", d.Get("name").(string))
	}

	newLbRule, err := expandAzureRmLoadbalancerRule(d, loadBalancer)
	if err != nil {
		return err
	}

	lbRules := append(*loadBalancer.Properties.LoadBalancingRules, *newLbRule)
	loadBalancer.Properties.LoadBalancingRules = &lbRules
	resGroup, loadBalancerName, err := resourceGroupAndLBNameFromId(d.Get("loadbalancer_id").(string))
	if err != nil {
		return errwrap.Wrapf("TODO: {{err}}", err)
	}

	_, err = lbClient.CreateOrUpdate(resGroup, loadBalancerName, *loadBalancer, make(chan struct{}))
	if err != nil {
		return err
	}

	read, err := lbClient.Get(resGroup, loadBalancerName, "")
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Loadbalancer %s (resource group %s) ID", loadBalancerName, resGroup)
	}

	d.SetId(*read.ID)

	log.Printf("[DEBUG] Waiting for LoadBalancer (%s) to become available", loadBalancerName)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  []string{"Succeeded"},
		Refresh: loadbalancerStateRefreshFunc(client, resGroup, loadBalancerName),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Loadbalancer (%s) to become available: %s", loadBalancerName, err)
	}

	return resourceArmLoadbalancerRuleRead(d, meta)
}

func resourceArmLoadbalancerRuleRead(d *schema.ResourceData, meta interface{}) error {
	loadBalancer, exists, err := retrieveLoadbalancerById(d.Id(), meta)
	if err != nil {
		return err
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] Loadbalancer %q not found. Refreshing from state", d.Get("name").(string))
		return nil
	}

	configs := *loadBalancer.Properties.LoadBalancingRules
	for _, config := range configs {
		if *config.Name == d.Get("name").(string) {
			d.Set("name", config.Name)

			d.Set("protocol", config.Properties.Protocol)
			d.Set("frontend_port", config.Properties.FrontendPort)
			d.Set("backend_port", config.Properties.BackendPort)

			if config.Properties.EnableFloatingIP != nil {
				d.Set("enable_floating_ip", config.Properties.EnableFloatingIP)
			}

			if config.Properties.IdleTimeoutInMinutes != nil {
				d.Set("idle_timeout_in_minutes", config.Properties.IdleTimeoutInMinutes)
			}

			if config.Properties.FrontendIPConfiguration != nil {
				d.Set("frontend_ip_configuration_id", config.Properties.FrontendIPConfiguration.ID)
			}

			if config.Properties.BackendAddressPool != nil {
				d.Set("backend_address_pool_id", config.Properties.BackendAddressPool.ID)
			}

			if config.Properties.Probe != nil {
				d.Set("probe_id", config.Properties.Probe.ID)
			}

			if config.Properties.LoadDistribution != "" {
				d.Set("load_distribution", config.Properties.LoadDistribution)
			}
		}
	}

	return nil
}

func resourceArmLoadbalancerRuleDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	lbClient := client.loadBalancerClient

	loadBalancer, exists, err := retrieveLoadbalancerById(d.Get("loadbalancer_id").(string), meta)
	if err != nil {
		return err
	}
	if !exists {
		d.SetId("")
		return nil
	}

	_, index, exists := findLoadBalancerRuleByName(loadBalancer, d.Get("name").(string))
	if !exists {
		return nil
	}

	oldLbRules := *loadBalancer.Properties.LoadBalancingRules
	newLbRules := append(oldLbRules[:index], oldLbRules[index+1:]...)
	loadBalancer.Properties.LoadBalancingRules = &newLbRules

	resGroup, loadBalancerName, err := resourceGroupAndLBNameFromId(d.Get("loadbalancer_id").(string))
	if err != nil {
		return errwrap.Wrapf("TODO: {{err}}", err)
	}

	_, err = lbClient.CreateOrUpdate(resGroup, loadBalancerName, *loadBalancer, make(chan struct{}))
	if err != nil {
		return err
	}

	read, err := lbClient.Get(resGroup, loadBalancerName, "")
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Loadbalancer %s (resource group %s) ID", loadBalancerName, resGroup)
	}

	return nil
}

func expandAzureRmLoadbalancerRule(d *schema.ResourceData, lb *network.LoadBalancer) (*network.LoadBalancingRule, error) {

	properties := network.LoadBalancingRulePropertiesFormat{
		Protocol:         network.TransportProtocol(d.Get("protocol").(string)),
		FrontendPort:     azure.Int32(int32(d.Get("frontend_port").(int))),
		BackendPort:      azure.Int32(int32(d.Get("backend_port").(int))),
		EnableFloatingIP: azure.Bool(d.Get("enable_floating_ip").(bool)),
	}

	if v, ok := d.GetOk("idle_timeout_in_minutes"); ok {
		properties.IdleTimeoutInMinutes = azure.Int32(int32(v.(int)))
	}

	if v := d.Get("load_distribution").(string); v != "" {
		properties.LoadDistribution = network.LoadDistribution(v)
	}

	if v := d.Get("frontend_ip_configuration_name").(string); v != "" {
		rule, _, exists := findLoadBalancerFrontEndIpConfigurationByName(lb, v)
		if !exists {
			return nil, fmt.Errorf("[ERROR] Cannot find FrontEnd IP Configuration with the name %s", v)
		}

		feip := network.SubResource{
			ID: rule.ID,
		}

		properties.FrontendIPConfiguration = &feip
	}

	if v := d.Get("backend_address_pool_id").(string); v != "" {
		beAP := network.SubResource{
			ID: &v,
		}

		properties.BackendAddressPool = &beAP
	}

	if v := d.Get("probe_id").(string); v != "" {
		pid := network.SubResource{
			ID: &v,
		}

		properties.Probe = &pid
	}

	lbRule := network.LoadBalancingRule{
		Name:       azure.String(d.Get("name").(string)),
		Properties: &properties,
	}

	return &lbRule, nil
}
