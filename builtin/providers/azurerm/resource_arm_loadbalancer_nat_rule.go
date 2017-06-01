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

func resourceArmLoadBalancerNatRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLoadBalancerNatRuleCreate,
		Read:   resourceArmLoadBalancerNatRuleRead,
		Update: resourceArmLoadBalancerNatRuleCreate,
		Delete: resourceArmLoadBalancerNatRuleDelete,
		Importer: &schema.ResourceImporter{
			State: loadBalancerSubResourceStateImporter,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
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

			"frontend_ip_configuration_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"frontend_ip_configuration_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"backend_ip_configuration_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceArmLoadBalancerNatRuleCreate(d *schema.ResourceData, meta interface{}) error {
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

	newNatRule, err := expandAzureRmLoadBalancerNatRule(d, loadBalancer)
	if err != nil {
		return errwrap.Wrapf("Error Expanding NAT Rule {{err}}", err)
	}

	natRules := append(*loadBalancer.LoadBalancerPropertiesFormat.InboundNatRules, *newNatRule)

	existingNatRule, existingNatRuleIndex, exists := findLoadBalancerNatRuleByName(loadBalancer, d.Get("name").(string))
	if exists {
		if d.Get("name").(string) == *existingNatRule.Name {
			// this probe is being updated/reapplied remove old copy from the slice
			natRules = append(natRules[:existingNatRuleIndex], natRules[existingNatRuleIndex+1:]...)
		}
	}

	loadBalancer.LoadBalancerPropertiesFormat.InboundNatRules = &natRules
	resGroup, loadBalancerName, err := resourceGroupAndLBNameFromId(d.Get("loadbalancer_id").(string))
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer Name and Group: {{err}}", err)
	}

	_, error := lbClient.CreateOrUpdate(resGroup, loadBalancerName, *loadBalancer, make(chan struct{}))
	err = <-error
	if err != nil {
		return errwrap.Wrapf("Error Creating / Updating LoadBalancer {{err}}", err)
	}

	read, err := lbClient.Get(resGroup, loadBalancerName, "")
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer {{err}}", err)
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read LoadBalancer %s (resource group %s) ID", loadBalancerName, resGroup)
	}

	var natRule_id string
	for _, InboundNatRule := range *(*read.LoadBalancerPropertiesFormat).InboundNatRules {
		if *InboundNatRule.Name == d.Get("name").(string) {
			natRule_id = *InboundNatRule.ID
		}
	}

	if natRule_id != "" {
		d.SetId(natRule_id)
	} else {
		return fmt.Errorf("Cannot find created LoadBalancer NAT Rule ID %q", natRule_id)
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

	return resourceArmLoadBalancerNatRuleRead(d, meta)
}

func resourceArmLoadBalancerNatRuleRead(d *schema.ResourceData, meta interface{}) error {
	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	name := id.Path["inboundNatRules"]

	loadBalancer, exists, err := retrieveLoadBalancerById(d.Get("loadbalancer_id").(string), meta)
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer By ID {{err}}", err)
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] LoadBalancer %q not found. Removing from state", name)
		return nil
	}

	config, _, exists := findLoadBalancerNatRuleByName(loadBalancer, name)
	if !exists {
		d.SetId("")
		log.Printf("[INFO] LoadBalancer Nat Rule %q not found. Removing from state", name)
		return nil
	}

	d.Set("name", config.Name)
	d.Set("resource_group_name", id.ResourceGroup)
	d.Set("protocol", config.InboundNatRulePropertiesFormat.Protocol)
	d.Set("frontend_port", config.InboundNatRulePropertiesFormat.FrontendPort)
	d.Set("backend_port", config.InboundNatRulePropertiesFormat.BackendPort)

	if config.InboundNatRulePropertiesFormat.FrontendIPConfiguration != nil {
		fipID, err := parseAzureResourceID(*config.InboundNatRulePropertiesFormat.FrontendIPConfiguration.ID)
		if err != nil {
			return err
		}

		d.Set("frontend_ip_configuration_name", fipID.Path["frontendIPConfigurations"])
		d.Set("frontend_ip_configuration_id", config.InboundNatRulePropertiesFormat.FrontendIPConfiguration.ID)
	}

	if config.InboundNatRulePropertiesFormat.BackendIPConfiguration != nil {
		d.Set("backend_ip_configuration_id", config.InboundNatRulePropertiesFormat.BackendIPConfiguration.ID)
	}

	return nil
}

func resourceArmLoadBalancerNatRuleDelete(d *schema.ResourceData, meta interface{}) error {
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

	_, index, exists := findLoadBalancerNatRuleByName(loadBalancer, d.Get("name").(string))
	if !exists {
		return nil
	}

	oldNatRules := *loadBalancer.LoadBalancerPropertiesFormat.InboundNatRules
	newNatRules := append(oldNatRules[:index], oldNatRules[index+1:]...)
	loadBalancer.LoadBalancerPropertiesFormat.InboundNatRules = &newNatRules

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

func expandAzureRmLoadBalancerNatRule(d *schema.ResourceData, lb *network.LoadBalancer) (*network.InboundNatRule, error) {

	properties := network.InboundNatRulePropertiesFormat{
		Protocol:     network.TransportProtocol(d.Get("protocol").(string)),
		FrontendPort: azure.Int32(int32(d.Get("frontend_port").(int))),
		BackendPort:  azure.Int32(int32(d.Get("backend_port").(int))),
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

	natRule := network.InboundNatRule{
		Name: azure.String(d.Get("name").(string)),
		InboundNatRulePropertiesFormat: &properties,
	}

	return &natRule, nil
}
