package azurerm

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/azure"
)

func resourceArmLoadBalancerRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLoadBalancerRuleCreate,
		Read:   resourceArmLoadBalancerRuleRead,
		Update: resourceArmLoadBalancerRuleCreate,
		Delete: resourceArmLoadBalancerRuleDelete,
		Importer: &schema.ResourceImporter{
			State: loadBalancerSubResourceStateImporter,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArmLoadBalancerRuleName,
			},

			"location": deprecatedLocationSchema(),

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
				Type:             schema.TypeString,
				Required:         true,
				StateFunc:        ignoreCaseStateFunc,
				DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
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

func resourceArmLoadBalancerRuleCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	lbClient := client.loadBalancerClient

	loadBalancerID := d.Get("loadbalancer_id").(string)
	armMutexKV.Lock(loadBalancerID)
	defer armMutexKV.Unlock(loadBalancerID)

	loadBalancer, exists, err := retrieveLoadBalancerById(loadBalancerID, meta)
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer By ID {{err}}", err)
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] LoadBalancer %q not found. Removing from state", d.Get("name").(string))
		return nil
	}

	newLbRule, err := expandAzureRmLoadBalancerRule(d, loadBalancer)
	if err != nil {
		return errwrap.Wrapf("Error Exanding LoadBalancer Rule {{err}}", err)
	}

	lbRules := append(*loadBalancer.LoadBalancerPropertiesFormat.LoadBalancingRules, *newLbRule)

	existingRule, existingRuleIndex, exists := findLoadBalancerRuleByName(loadBalancer, d.Get("name").(string))
	if exists {
		if d.Get("name").(string) == *existingRule.Name {
			// this rule is being updated/reapplied remove old copy from the slice
			lbRules = append(lbRules[:existingRuleIndex], lbRules[existingRuleIndex+1:]...)
		}
	}

	loadBalancer.LoadBalancerPropertiesFormat.LoadBalancingRules = &lbRules
	resGroup, loadBalancerName, err := resourceGroupAndLBNameFromId(d.Get("loadbalancer_id").(string))
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer Name and Group: {{err}}", err)
	}

	_, error := lbClient.CreateOrUpdate(resGroup, loadBalancerName, *loadBalancer, make(chan struct{}))
	err = <-error
	if err != nil {
		return errwrap.Wrapf("Error Creating/Updating LoadBalancer {{err}}", err)
	}

	read, err := lbClient.Get(resGroup, loadBalancerName, "")
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer {{err}}", err)
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read LoadBalancer %s (resource group %s) ID", loadBalancerName, resGroup)
	}

	var rule_id string
	for _, LoadBalancingRule := range *(*read.LoadBalancerPropertiesFormat).LoadBalancingRules {
		if *LoadBalancingRule.Name == d.Get("name").(string) {
			rule_id = *LoadBalancingRule.ID
		}
	}

	if rule_id != "" {
		d.SetId(rule_id)
	} else {
		return fmt.Errorf("Cannot find created LoadBalancer Rule ID %q", rule_id)
	}

	log.Printf("[DEBUG] Waiting for LoadBalancer (%s) to become available", loadBalancerName)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  []string{"Succeeded"},
		Refresh: loadbalancerStateRefreshFunc(client, resGroup, loadBalancerName),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for LoadBalancer (%s) to become available: %s", loadBalancerName, err)
	}

	return resourceArmLoadBalancerRuleRead(d, meta)
}

func resourceArmLoadBalancerRuleRead(d *schema.ResourceData, meta interface{}) error {
	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	name := id.Path["loadBalancingRules"]

	loadBalancer, exists, err := retrieveLoadBalancerById(d.Get("loadbalancer_id").(string), meta)
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer By ID {{err}}", err)
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] LoadBalancer %q not found. Removing from state", name)
		return nil
	}

	config, _, exists := findLoadBalancerRuleByName(loadBalancer, name)
	if !exists {
		d.SetId("")
		log.Printf("[INFO] LoadBalancer Rule %q not found. Removing from state", name)
		return nil
	}

	d.Set("name", config.Name)
	d.Set("resource_group_name", id.ResourceGroup)

	d.Set("protocol", config.LoadBalancingRulePropertiesFormat.Protocol)
	d.Set("frontend_port", config.LoadBalancingRulePropertiesFormat.FrontendPort)
	d.Set("backend_port", config.LoadBalancingRulePropertiesFormat.BackendPort)

	if config.LoadBalancingRulePropertiesFormat.EnableFloatingIP != nil {
		d.Set("enable_floating_ip", config.LoadBalancingRulePropertiesFormat.EnableFloatingIP)
	}

	if config.LoadBalancingRulePropertiesFormat.IdleTimeoutInMinutes != nil {
		d.Set("idle_timeout_in_minutes", config.LoadBalancingRulePropertiesFormat.IdleTimeoutInMinutes)
	}

	if config.LoadBalancingRulePropertiesFormat.FrontendIPConfiguration != nil {
		fipID, err := parseAzureResourceID(*config.LoadBalancingRulePropertiesFormat.FrontendIPConfiguration.ID)
		if err != nil {
			return err
		}

		d.Set("frontend_ip_configuration_name", fipID.Path["frontendIPConfigurations"])
		d.Set("frontend_ip_configuration_id", config.LoadBalancingRulePropertiesFormat.FrontendIPConfiguration.ID)
	}

	if config.LoadBalancingRulePropertiesFormat.BackendAddressPool != nil {
		d.Set("backend_address_pool_id", config.LoadBalancingRulePropertiesFormat.BackendAddressPool.ID)
	}

	if config.LoadBalancingRulePropertiesFormat.Probe != nil {
		d.Set("probe_id", config.LoadBalancingRulePropertiesFormat.Probe.ID)
	}

	if config.LoadBalancingRulePropertiesFormat.LoadDistribution != "" {
		d.Set("load_distribution", config.LoadBalancingRulePropertiesFormat.LoadDistribution)
	}

	return nil
}

func resourceArmLoadBalancerRuleDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	lbClient := client.loadBalancerClient

	loadBalancerID := d.Get("loadbalancer_id").(string)
	armMutexKV.Lock(loadBalancerID)
	defer armMutexKV.Unlock(loadBalancerID)

	loadBalancer, exists, err := retrieveLoadBalancerById(loadBalancerID, meta)
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer By ID {{err}}", err)
	}
	if !exists {
		d.SetId("")
		return nil
	}

	_, index, exists := findLoadBalancerRuleByName(loadBalancer, d.Get("name").(string))
	if !exists {
		return nil
	}

	oldLbRules := *loadBalancer.LoadBalancerPropertiesFormat.LoadBalancingRules
	newLbRules := append(oldLbRules[:index], oldLbRules[index+1:]...)
	loadBalancer.LoadBalancerPropertiesFormat.LoadBalancingRules = &newLbRules

	resGroup, loadBalancerName, err := resourceGroupAndLBNameFromId(d.Get("loadbalancer_id").(string))
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer Name and Group: {{err}}", err)
	}

	_, error := lbClient.CreateOrUpdate(resGroup, loadBalancerName, *loadBalancer, make(chan struct{}))
	err = <-error
	if err != nil {
		return errwrap.Wrapf("Error Creating/Updating LoadBalancer {{err}}", err)
	}

	read, err := lbClient.Get(resGroup, loadBalancerName, "")
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer {{err}}", err)
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read LoadBalancer %s (resource group %s) ID", loadBalancerName, resGroup)
	}

	return nil
}

func expandAzureRmLoadBalancerRule(d *schema.ResourceData, lb *network.LoadBalancer) (*network.LoadBalancingRule, error) {

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
		Name: azure.String(d.Get("name").(string)),
		LoadBalancingRulePropertiesFormat: &properties,
	}

	return &lbRule, nil
}

func validateArmLoadBalancerRuleName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[a-zA-Z_0-9.-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only word characters, numbers, underscores, periods, and hyphens allowed in %q: %q",
			k, value))
	}

	if len(value) > 80 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 80 characters: %q", k, value))
	}

	if len(value) == 0 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be an empty string: %q", k, value))
	}
	if !regexp.MustCompile(`[a-zA-Z0-9_]$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q must end with a word character, number, or underscore: %q", k, value))
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q must start with a word character or number: %q", k, value))
	}

	return
}
