package google

import (
	"fmt"
	"log"
	"time"

	"code.google.com/p/google-api-go-client/compute/v1"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceComputeNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeNetworkCreate,
		Read:   resourceComputeNetworkRead,
		Delete: resourceComputeNetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ipv4_range": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"gateway_ipv4": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceComputeNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Build the network parameter
	network := &compute.Network{
		Name:      d.Get("name").(string),
		IPv4Range: d.Get("ipv4_range").(string),
	}
	log.Printf("[DEBUG] Network insert request: %#v", network)
	op, err := config.clientCompute.Networks.Insert(
		config.Project, network).Do()
	if err != nil {
		return fmt.Errorf("Error creating network: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(network.Name)

	// Wait for the operation to complete
	w := &OperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: config.Project,
		Type:    OperationWaitGlobal,
	}
	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	if _, err := state.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for address to create: %s", err)
	}

	return resourceComputeNetworkRead(d, meta)
}

func resourceComputeNetworkRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	network, err := config.clientCompute.Networks.Get(
		config.Project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error reading address: %s", err)
	}

	d.Set("gateway_ipv4", network.GatewayIPv4)

	return nil
}

func resourceComputeNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Delete the address
	op, err := config.clientCompute.Networks.Delete(
		config.Project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting network: %s", err)
	}

	// Wait for the operation to complete
	w := &OperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: config.Project,
		Type:    OperationWaitGlobal,
	}
	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	if _, err := state.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for address to delete: %s", err)
	}

	d.SetId("")
	return nil
}
