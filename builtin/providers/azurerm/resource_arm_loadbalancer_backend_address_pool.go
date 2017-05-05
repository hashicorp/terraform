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

func resourceArmLoadBalancerBackendAddressPool() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLoadBalancerBackendAddressPoolCreate,
		Read:   resourceArmLoadBalancerBackendAddressPoolRead,
		Delete: resourceArmLoadBalancerBackendAddressPoolDelete,
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

			"backend_ip_configurations": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"load_balancing_rules": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceArmLoadBalancerBackendAddressPoolCreate(d *schema.ResourceData, meta interface{}) error {
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

	backendAddressPools := append(*loadBalancer.LoadBalancerPropertiesFormat.BackendAddressPools, expandAzureRmLoadBalancerBackendAddressPools(d))
	existingPool, existingPoolIndex, exists := findLoadBalancerBackEndAddressPoolByName(loadBalancer, d.Get("name").(string))
	if exists {
		if d.Get("name").(string) == *existingPool.Name {
			// this pool is being updated/reapplied remove old copy from the slice
			backendAddressPools = append(backendAddressPools[:existingPoolIndex], backendAddressPools[existingPoolIndex+1:]...)
		}
	}

	loadBalancer.LoadBalancerPropertiesFormat.BackendAddressPools = &backendAddressPools
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

	var pool_id string
	for _, BackendAddressPool := range *(*read.LoadBalancerPropertiesFormat).BackendAddressPools {
		if *BackendAddressPool.Name == d.Get("name").(string) {
			pool_id = *BackendAddressPool.ID
		}
	}

	if pool_id != "" {
		d.SetId(pool_id)
	} else {
		return fmt.Errorf("Cannot find created LoadBalancer Backend Address Pool ID %q", pool_id)
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

	return resourceArmLoadBalancerBackendAddressPoolRead(d, meta)
}

func resourceArmLoadBalancerBackendAddressPoolRead(d *schema.ResourceData, meta interface{}) error {
	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	name := id.Path["backendAddressPools"]

	loadBalancer, exists, err := retrieveLoadBalancerById(d.Get("loadbalancer_id").(string), meta)
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer By ID {{err}}", err)
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] LoadBalancer %q not found. Removing from state", name)
		return nil
	}

	config, _, exists := findLoadBalancerBackEndAddressPoolByName(loadBalancer, name)
	if !exists {
		d.SetId("")
		log.Printf("[INFO] LoadBalancer Backend Address Pool %q not found. Removing from state", name)
		return nil
	}

	d.Set("name", config.Name)
	d.Set("resource_group_name", id.ResourceGroup)

	var backend_ip_configurations []string
	if config.BackendAddressPoolPropertiesFormat.BackendIPConfigurations != nil {
		for _, backendConfig := range *config.BackendAddressPoolPropertiesFormat.BackendIPConfigurations {
			backend_ip_configurations = append(backend_ip_configurations, *backendConfig.ID)
		}

	}
	d.Set("backend_ip_configurations", backend_ip_configurations)

	var load_balancing_rules []string
	if config.BackendAddressPoolPropertiesFormat.LoadBalancingRules != nil {
		for _, rule := range *config.BackendAddressPoolPropertiesFormat.LoadBalancingRules {
			load_balancing_rules = append(load_balancing_rules, *rule.ID)
		}
	}
	d.Set("load_balancing_rules", load_balancing_rules)

	return nil
}

func resourceArmLoadBalancerBackendAddressPoolDelete(d *schema.ResourceData, meta interface{}) error {
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

	_, index, exists := findLoadBalancerBackEndAddressPoolByName(loadBalancer, d.Get("name").(string))
	if !exists {
		return nil
	}

	oldBackEndPools := *loadBalancer.LoadBalancerPropertiesFormat.BackendAddressPools
	newBackEndPools := append(oldBackEndPools[:index], oldBackEndPools[index+1:]...)
	loadBalancer.LoadBalancerPropertiesFormat.BackendAddressPools = &newBackEndPools

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

func expandAzureRmLoadBalancerBackendAddressPools(d *schema.ResourceData) network.BackendAddressPool {
	return network.BackendAddressPool{
		Name: azure.String(d.Get("name").(string)),
	}
}
