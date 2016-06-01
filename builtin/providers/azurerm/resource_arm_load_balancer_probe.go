package azurerm

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"strings"
)

// resourceArmLoadBalancer returns the *schema.Resource
// associated to load balancer resources on ARM.
func resourceArmProbe() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmProbeCreate,
		Read:   resourceProbeRead,
		Update: resourceArmProbeUpdate,
		Delete: resourceProbeDelete,

		Schema: map[string]*schema.Schema{

			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"protocol": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateProtocolType,
			},
			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"interval": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"number_of_probes": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"request_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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

func validateProtocolType(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	allocations := map[string]bool{
		"tcp": true,
		"udp": true,
	}

	if !allocations[value] {
		errors = append(errors, fmt.Errorf("Protocol can only be tcp or udp"))
	}
	return
}

func resourceArmProbeCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmProbe] resourceArmProbeCreate[enter]")
	defer log.Printf("[resourceArmProbe] resourceArmProbeCreate[exit]")

	// first; fetch a bunch of fields:
	probeName := d.Get("name").(string)
	protocol := d.Get("protocol").(string)
	requestPath := d.Get("request_path").(string)
	port := d.Get("port").(int)
	interval := d.Get("interval").(int)
	numberOfProbes := d.Get("number_of_probes").(int)
	loadBalancerName := d.Get("load_balancer_name").(string)
	resourceGroupName := d.Get("resource_group_name").(string)

	lbClient := meta.(*ArmClient).loadBalancerClient
	loadBalancer, err := lbClient.Get(resourceGroupName, loadBalancerName, "")
	if err != nil {
		return err
	}

	probeProps := network.ProbePropertiesFormat{
		Port:              &port,
		IntervalInSeconds: &interval,
		NumberOfProbes:    &numberOfProbes,
		Protocol:          network.ProbeProtocol(protocol),
	}

	if requestPath != "" && network.ProbeProtocol(protocol) != network.ProbeProtocolHTTP {
		return fmt.Errorf("When using HTTP there must be a request path '%s': %s", probeName, err)
	}
	if requestPath != "" {
		probeProps.RequestPath = &requestPath
	}

	probeStruct := network.Probe{Name: &probeName, Properties: &probeProps}
	i, err := findProbeConf(loadBalancer.Properties.Probes, probeName)
	if err == nil {
		// If one by that name exists update it
		(*loadBalancer.Properties.Probes)[i] = probeStruct
	} else {
		probeArray := append(*loadBalancer.Properties.Probes, probeStruct)
		loadBalancer.Properties.Probes = &probeArray
	}

	loadBalancer, err = lbClient.CreateOrUpdate(resourceGroupName, loadBalancerName, loadBalancer)
	if err != nil {
		log.Printf("[resourceArmProbe] ERROR LB got status %s", err.Error())
		return fmt.Errorf("Error issuing Azure ARM creation request forprobe '%s': %s", probeName, err)
	}
	i, err = findProbeConf(loadBalancer.Properties.Probes, probeName)
	if err != nil {
		return err
	}

	probeOut := (*loadBalancer.Properties.Probes)[i]

	d.SetId(*probeOut.ID)
	return nil
}

func resourceArmProbeUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmProbe] resourceArmProbeUpdate[enter]")
	defer log.Printf("[resourceArmProbe] resourceArmProbeUpdate[exit]")

	return resourceArmProbeCreate(d, meta)
}

func resourceProbeDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmProbe] resourceProbeDelete[enter]")
	defer log.Printf("[resourceArmProbe] resourceProbeDelete[exit]")

	lbClient := meta.(*ArmClient).loadBalancerClient

	probeName := d.Get("name").(string)
	loadBalancerName := d.Get("load_balancer_name").(string)
	resourceGroupName := d.Get("resource_group_name").(string)

	loadBalancer, err := lbClient.Get(resourceGroupName, loadBalancerName, "")
	if err != nil {
		return err
	}
	i, err := findProbeConf(loadBalancer.Properties.Probes, probeName)
	if err != nil {
		return err
	}
	probeA := append((*loadBalancer.Properties.Probes)[:i], (*loadBalancer.Properties.Probes)[i+1:]...)
	loadBalancer.Properties.Probes = &probeA
	_, err = lbClient.CreateOrUpdate(resourceGroupName, loadBalancerName, loadBalancer)
	if err != nil {
		return err
	}
	return nil
}

// resourceArmLoadBalancerRead goes ahead and reads the state of the corresponding ARM load balancer.
func resourceProbeRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmProbe] resourceProbeRead[enter]")
	defer log.Printf("[resourceArmProbe] resourceProbeRead[exit]")

	lbClient := meta.(*ArmClient).loadBalancerClient

	probeName := d.Get("name").(string)
	loadBalancerName := d.Get("load_balancer_name").(string)
	resourceGroupName := d.Get("resource_group_name").(string)

	log.Printf("[INFO] Issuing read request of probe '%s'.", probeName)

	loadBalancer, err := lbClient.Get(resourceGroupName, loadBalancerName, "")
	if err != nil {
		return err
	}
	i, err := findProbeConf(loadBalancer.Properties.Probes, probeName)
	if err != nil {
		return err
	}

	probeStruct := (*loadBalancer.Properties.Probes)[i]

	d.Set("port", *probeStruct.Properties.Port)
	d.Set("interval", *probeStruct.Properties.IntervalInSeconds)
	d.Set("number_of_probes", *probeStruct.Properties.NumberOfProbes)
	if probeStruct.Properties.RequestPath != nil {
		d.Set("request_path", *probeStruct.Properties.RequestPath)
	}
	d.Set("protocol", string(probeStruct.Properties.Protocol))
	d.SetId(*probeStruct.ID)
	return nil
}

func findProbeConf(probeArray *[]network.Probe, probeName string) (int, error) {
	// Find the correct LB
	for i := 0; i < len(*probeArray); i++ {
		tmpProbe := (*probeArray)[i]
		if *tmpProbe.Name == probeName {
			return i, nil
		}
	}
	return -1, fmt.Errorf("Error loading the probe named %s", probeName)
}
