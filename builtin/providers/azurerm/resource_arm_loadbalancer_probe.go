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

func resourceArmLoadBalancerProbe() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLoadBalancerProbeCreate,
		Read:   resourceArmLoadBalancerProbeRead,
		Update: resourceArmLoadBalancerProbeCreate,
		Delete: resourceArmLoadBalancerProbeDelete,

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

			"loadbalancer_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"protocol": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},

			"port": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"request_path": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"interval_in_seconds": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  15,
			},

			"number_of_probes": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  2,
			},

			"load_balance_rules": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceArmLoadBalancerProbeCreate(d *schema.ResourceData, meta interface{}) error {
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

	newProbe, err := expandAzureRmLoadBalancerProbe(d, loadBalancer)
	if err != nil {
		return errwrap.Wrapf("Error Expanding Probe {{err}}", err)
	}

	probes := append(*loadBalancer.LoadBalancerPropertiesFormat.Probes, *newProbe)

	existingProbe, existingProbeIndex, exists := findLoadBalancerProbeByName(loadBalancer, d.Get("name").(string))
	if exists {
		if d.Id() == *existingProbe.ID {
			// this probe is being updated remove old copy from the slice
			probes = append(probes[:existingProbeIndex], probes[existingProbeIndex+1:]...)
		} else {
			return fmt.Errorf("A Probe with name %q already exists.", d.Get("name").(string))
		}
	}

	loadBalancer.LoadBalancerPropertiesFormat.Probes = &probes
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

	var createdProbe_id string
	for _, Probe := range *(*read.LoadBalancerPropertiesFormat).Probes {
		if *Probe.Name == d.Get("name").(string) {
			createdProbe_id = *Probe.ID
		}
	}

	if createdProbe_id != "" {
		d.SetId(createdProbe_id)
	} else {
		return fmt.Errorf("Cannot find created LoadBalancer Probe ID %q", createdProbe_id)
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

	return resourceArmLoadBalancerProbeRead(d, meta)
}

func resourceArmLoadBalancerProbeRead(d *schema.ResourceData, meta interface{}) error {
	loadBalancer, exists, err := retrieveLoadBalancerById(d.Get("loadbalancer_id").(string), meta)
	if err != nil {
		return errwrap.Wrapf("Error Getting LoadBalancer By ID {{err}}", err)
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] LoadBalancer %q not found. Removing from state", d.Get("name").(string))
		return nil
	}

	configs := *loadBalancer.LoadBalancerPropertiesFormat.Probes
	for _, config := range configs {
		if *config.Name == d.Get("name").(string) {
			d.Set("name", config.Name)

			d.Set("protocol", config.ProbePropertiesFormat.Protocol)
			d.Set("interval_in_seconds", config.ProbePropertiesFormat.IntervalInSeconds)
			d.Set("number_of_probes", config.ProbePropertiesFormat.NumberOfProbes)
			d.Set("port", config.ProbePropertiesFormat.Port)
			d.Set("request_path", config.ProbePropertiesFormat.RequestPath)

			break
		}
	}

	return nil
}

func resourceArmLoadBalancerProbeDelete(d *schema.ResourceData, meta interface{}) error {
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

	_, index, exists := findLoadBalancerProbeByName(loadBalancer, d.Get("name").(string))
	if !exists {
		return nil
	}

	oldProbes := *loadBalancer.LoadBalancerPropertiesFormat.Probes
	newProbes := append(oldProbes[:index], oldProbes[index+1:]...)
	loadBalancer.LoadBalancerPropertiesFormat.Probes = &newProbes

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

func expandAzureRmLoadBalancerProbe(d *schema.ResourceData, lb *network.LoadBalancer) (*network.Probe, error) {

	properties := network.ProbePropertiesFormat{
		NumberOfProbes:    azure.Int32(int32(d.Get("number_of_probes").(int))),
		IntervalInSeconds: azure.Int32(int32(d.Get("interval_in_seconds").(int))),
		Port:              azure.Int32(int32(d.Get("port").(int))),
	}

	if v, ok := d.GetOk("protocol"); ok {
		properties.Protocol = network.ProbeProtocol(v.(string))
	}

	if v, ok := d.GetOk("request_path"); ok {
		properties.RequestPath = azure.String(v.(string))
	}

	probe := network.Probe{
		Name: azure.String(d.Get("name").(string)),
		ProbePropertiesFormat: &properties,
	}

	return &probe, nil
}
