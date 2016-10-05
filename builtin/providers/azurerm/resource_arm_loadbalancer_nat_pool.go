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

func resourceArmLoadbalancerNatPool() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLoadbalancerNatPoolCreate,
		Read:   resourceArmLoadbalancerNatPoolRead,
		Update: resourceArmLoadbalancerNatPoolCreate,
		Delete: resourceArmLoadbalancerNatPoolDelete,

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

			"protocol": {
				Type:     schema.TypeString,
				Required: true,
			},

			"frontend_port_start": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"frontend_port_end": {
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
		},
	}
}

func resourceArmLoadbalancerNatPoolCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	lbClient := client.loadBalancerClient

	loadBalancer, exists, err := retrieveLoadbalancerById(d.Get("loadbalancer_id").(string), meta)
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer By ID {{err}}", err)
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] LoadBalancer %q not found. Removing from state", d.Get("name").(string))
		return nil
	}

	_, _, exists = findLoadBalancerNatPoolByName(loadBalancer, d.Get("name").(string))
	if exists {
		return fmt.Errorf("A NAT Pool with name %q already exists.", d.Get("name").(string))
	}

	newNatPool, err := expandAzureRmLoadbalancerNatPool(d, loadBalancer)
	if err != nil {
		return errwrap.Wrapf("Error Expanding NAT Pool {{err}}", err)
	}

	natPools := append(*loadBalancer.Properties.InboundNatPools, *newNatPool)
	loadBalancer.Properties.InboundNatPools = &natPools
	resGroup, loadBalancerName, err := resourceGroupAndLBNameFromId(d.Get("loadbalancer_id").(string))
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer Name and Group: {{err}}", err)
	}

	_, err = lbClient.CreateOrUpdate(resGroup, loadBalancerName, *loadBalancer, make(chan struct{}))
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

	d.SetId(*read.ID)

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

	return resourceArmLoadbalancerNatPoolRead(d, meta)
}

func resourceArmLoadbalancerNatPoolRead(d *schema.ResourceData, meta interface{}) error {
	loadBalancer, exists, err := retrieveLoadbalancerById(d.Id(), meta)
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer By ID {{err}}", err)
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] LoadBalancer %q not found. Removing from state", d.Get("name").(string))
		return nil
	}

	configs := *loadBalancer.Properties.InboundNatPools
	for _, config := range configs {
		if *config.Name == d.Get("name").(string) {
			d.Set("name", config.Name)

			d.Set("protocol", config.Properties.Protocol)
			d.Set("frontend_port_start", config.Properties.FrontendPortRangeStart)
			d.Set("frontend_port_end", config.Properties.FrontendPortRangeEnd)
			d.Set("backend_port", config.Properties.BackendPort)

			if config.Properties.FrontendIPConfiguration != nil {
				d.Set("frontend_ip_configuration_id", config.Properties.FrontendIPConfiguration.ID)
			}

			break
		}
	}

	return nil
}

func resourceArmLoadbalancerNatPoolDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	lbClient := client.loadBalancerClient

	loadBalancer, exists, err := retrieveLoadbalancerById(d.Get("loadbalancer_id").(string), meta)
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer By ID {{err}}", err)
	}
	if !exists {
		d.SetId("")
		return nil
	}

	_, index, exists := findLoadBalancerNatPoolByName(loadBalancer, d.Get("name").(string))
	if !exists {
		return nil
	}

	oldNatPools := *loadBalancer.Properties.InboundNatPools
	newNatPools := append(oldNatPools[:index], oldNatPools[index+1:]...)
	loadBalancer.Properties.InboundNatPools = &newNatPools

	resGroup, loadBalancerName, err := resourceGroupAndLBNameFromId(d.Get("loadbalancer_id").(string))
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer Name and Group: {{err}}", err)
	}

	_, err = lbClient.CreateOrUpdate(resGroup, loadBalancerName, *loadBalancer, make(chan struct{}))
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

func expandAzureRmLoadbalancerNatPool(d *schema.ResourceData, lb *network.LoadBalancer) (*network.InboundNatPool, error) {

	properties := network.InboundNatPoolPropertiesFormat{
		Protocol:               network.TransportProtocol(d.Get("protocol").(string)),
		FrontendPortRangeStart: azure.Int32(int32(d.Get("frontend_port_start").(int))),
		FrontendPortRangeEnd:   azure.Int32(int32(d.Get("frontend_port_end").(int))),
		BackendPort:            azure.Int32(int32(d.Get("backend_port").(int))),
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

	natPool := network.InboundNatPool{
		Name:       azure.String(d.Get("name").(string)),
		Properties: &properties,
	}

	return &natPool, nil
}
