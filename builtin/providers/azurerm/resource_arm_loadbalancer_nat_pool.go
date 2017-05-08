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

func resourceArmLoadBalancerNatPool() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLoadBalancerNatPoolCreate,
		Read:   resourceArmLoadBalancerNatPoolRead,
		Update: resourceArmLoadBalancerNatPoolCreate,
		Delete: resourceArmLoadBalancerNatPoolDelete,
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

func resourceArmLoadBalancerNatPoolCreate(d *schema.ResourceData, meta interface{}) error {
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

	newNatPool, err := expandAzureRmLoadBalancerNatPool(d, loadBalancer)
	if err != nil {
		return errwrap.Wrapf("Error Expanding NAT Pool {{err}}", err)
	}

	natPools := append(*loadBalancer.LoadBalancerPropertiesFormat.InboundNatPools, *newNatPool)

	existingNatPool, existingNatPoolIndex, exists := findLoadBalancerNatPoolByName(loadBalancer, d.Get("name").(string))
	if exists {
		if d.Get("name").(string) == *existingNatPool.Name {
			// this probe is being updated/reapplied remove old copy from the slice
			natPools = append(natPools[:existingNatPoolIndex], natPools[existingNatPoolIndex+1:]...)
		}
	}

	loadBalancer.LoadBalancerPropertiesFormat.InboundNatPools = &natPools
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

	var natPool_id string
	for _, InboundNatPool := range *(*read.LoadBalancerPropertiesFormat).InboundNatPools {
		if *InboundNatPool.Name == d.Get("name").(string) {
			natPool_id = *InboundNatPool.ID
		}
	}

	if natPool_id != "" {
		d.SetId(natPool_id)
	} else {
		return fmt.Errorf("Cannot find created LoadBalancer NAT Pool ID %q", natPool_id)
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

	return resourceArmLoadBalancerNatPoolRead(d, meta)
}

func resourceArmLoadBalancerNatPoolRead(d *schema.ResourceData, meta interface{}) error {
	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	name := id.Path["inboundNatPools"]

	loadBalancer, exists, err := retrieveLoadBalancerById(d.Get("loadbalancer_id").(string), meta)
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer By ID {{err}}", err)
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] LoadBalancer %q not found. Removing from state", name)
		return nil
	}

	config, _, exists := findLoadBalancerNatPoolByName(loadBalancer, name)
	if !exists {
		d.SetId("")
		log.Printf("[INFO] LoadBalancer Nat Pool %q not found. Removing from state", name)
		return nil
	}

	d.Set("name", config.Name)
	d.Set("resource_group_name", id.ResourceGroup)
	d.Set("protocol", config.InboundNatPoolPropertiesFormat.Protocol)
	d.Set("frontend_port_start", config.InboundNatPoolPropertiesFormat.FrontendPortRangeStart)
	d.Set("frontend_port_end", config.InboundNatPoolPropertiesFormat.FrontendPortRangeEnd)
	d.Set("backend_port", config.InboundNatPoolPropertiesFormat.BackendPort)

	if config.InboundNatPoolPropertiesFormat.FrontendIPConfiguration != nil {
		fipID, err := parseAzureResourceID(*config.InboundNatPoolPropertiesFormat.FrontendIPConfiguration.ID)
		if err != nil {
			return err
		}

		d.Set("frontend_ip_configuration_name", fipID.Path["frontendIPConfigurations"])
		d.Set("frontend_ip_configuration_id", config.InboundNatPoolPropertiesFormat.FrontendIPConfiguration.ID)
	}

	return nil
}

func resourceArmLoadBalancerNatPoolDelete(d *schema.ResourceData, meta interface{}) error {
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

	_, index, exists := findLoadBalancerNatPoolByName(loadBalancer, d.Get("name").(string))
	if !exists {
		return nil
	}

	oldNatPools := *loadBalancer.LoadBalancerPropertiesFormat.InboundNatPools
	newNatPools := append(oldNatPools[:index], oldNatPools[index+1:]...)
	loadBalancer.LoadBalancerPropertiesFormat.InboundNatPools = &newNatPools

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

func expandAzureRmLoadBalancerNatPool(d *schema.ResourceData, lb *network.LoadBalancer) (*network.InboundNatPool, error) {

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
		Name: azure.String(d.Get("name").(string)),
		InboundNatPoolPropertiesFormat: &properties,
	}

	return &natPool, nil
}
