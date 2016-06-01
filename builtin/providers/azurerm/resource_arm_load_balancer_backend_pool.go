package azurerm

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
)

// resourceArmLoadBalancer returns the *schema.Resource
// associated to load balancer resources on ARM.
func resourceArmBackendPool() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmBackendPoolCreate,
		Read:   resourceArmBackendPoolRead,
		Update: resourceArmBackendPoolUpdate,
		Delete: resourceArmBackendPoolDelete,

		Schema: map[string]*schema.Schema{

			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"load_balancer_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceArmBackendPoolCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmBackendPool] resourceArmBackendPoolCreate[enter]")
	defer log.Printf("[resourceArmBackendPool] resourceArmBackendPoolCreate[exit]")

	// first; fetch a bunch of fields:
	backendPoolName := d.Get("name").(string)
	loadBalancerName := d.Get("load_balancer_name").(string)
	resourceGroupName := d.Get("resource_group_name").(string)

	lbClient := meta.(*ArmClient).loadBalancerClient
	loadBalancer, err := lbClient.Get(resourceGroupName, loadBalancerName, "")
	if err != nil {
		return err
	}

	backendAddressPoolStruct := network.BackendAddressPool{
		Name: &backendPoolName,
	}

	i, err := findBackendAddressConf(loadBalancer.Properties.BackendAddressPools, backendPoolName)
	if err == nil {
		// If one by that name exists update it
		(*loadBalancer.Properties.BackendAddressPools)[i] = backendAddressPoolStruct
	} else {
		backendPoolArray := append(*loadBalancer.Properties.BackendAddressPools, backendAddressPoolStruct)
		loadBalancer.Properties.BackendAddressPools = &backendPoolArray
	}

	loadBalancer, err = lbClient.CreateOrUpdate(resourceGroupName, loadBalancerName, loadBalancer)
	if err != nil {
		log.Printf("[resourceArmBackendPool] ERROR LB got status %s", err.Error())
		return fmt.Errorf("Error issuing Azure ARM creation request for backend pool '%s': %s", backendPoolName, err)
	}
	i, err = findBackendAddressConf(loadBalancer.Properties.BackendAddressPools, backendPoolName)
	if err != nil {
		return err
	}

	backendOut := (*loadBalancer.Properties.BackendAddressPools)[i]
	log.Printf("[resourceArmBackendPool] Created the backend pool %s with ID %s", *backendOut.Name, *backendOut.ID)

	d.SetId(*backendOut.ID)
	return nil
}

func resourceArmBackendPoolUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmBackendPool] resourceArmBackendPoolUpdate[enter]")
	defer log.Printf("[resourceArmBackendPool] resourceArmBackendPoolUpdate[exit]")

	return resourceArmBackendPoolCreate(d, meta)
}

func resourceArmBackendPoolDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmBackendPool] resourceArmBackendPoolDelete[enter]")
	defer log.Printf("[resourceArmBackendPool] resourceArmBackendPoolDelete[exit]")

	lbClient := meta.(*ArmClient).loadBalancerClient

	backendPoolName := d.Get("name").(string)
	loadBalancerName := d.Get("load_balancer_name").(string)
	resourceGroupName := d.Get("resource_group_name").(string)

	loadBalancer, err := lbClient.Get(resourceGroupName, loadBalancerName, "")
	if err != nil {
		return err
	}
	i, err := findBackendAddressConf(loadBalancer.Properties.BackendAddressPools, backendPoolName)
	if err != nil {
		return err
	}
	backendA := append((*loadBalancer.Properties.BackendAddressPools)[:i], (*loadBalancer.Properties.BackendAddressPools)[i+1:]...)
	loadBalancer.Properties.BackendAddressPools = &backendA
	_, err = lbClient.CreateOrUpdate(resourceGroupName, loadBalancerName, loadBalancer)
	if err != nil {
		return err
	}
	return nil
}

// resourceArmLoadBalancerRead goes ahead and reads the state of the corresponding ARM load balancer.
func resourceArmBackendPoolRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmBackendPool] resourceArmBackendPoolRead[enter]")
	defer log.Printf("[resourceArmBackendPool] resourceArmBackendPoolRead[exit]")

	lbClient := meta.(*ArmClient).loadBalancerClient

	backendPoolName := d.Get("name").(string)
	loadBalancerName := d.Get("load_balancer_name").(string)
	resourceGroupName := d.Get("resource_group_name").(string)

	loadBalancer, err := lbClient.Get(resourceGroupName, loadBalancerName, "")
	if err != nil {
		return err
	}
	i, err := findBackendAddressConf(loadBalancer.Properties.BackendAddressPools, backendPoolName)
	if err != nil {
		return err
	}

	backendPoolStruct := (*loadBalancer.Properties.BackendAddressPools)[i]
	d.SetId(*backendPoolStruct.ID)
	return nil
}

func findBackendAddressConf(backendAddressArray *[]network.BackendAddressPool, backendAddressName string) (int, error) {
	// Find the correct LB
	for i := 0; i < len(*backendAddressArray); i++ {
		tmpProbe := (*backendAddressArray)[i]
		if *tmpProbe.Name == backendAddressName {
			return i, nil
		}
	}
	return -1, fmt.Errorf("Error finding the backend pool named %s", backendAddressName)
}
