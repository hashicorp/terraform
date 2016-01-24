package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeRoute() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeRouteCreate,
		Read:   resourceComputeRouteRead,
		Delete: resourceComputeRouteDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"dest_range": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"next_hop_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"next_hop_instance": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"next_hop_instance_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"next_hop_gateway": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"next_hop_network": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"next_hop_vpn_tunnel": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"priority": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"tags": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceComputeRouteCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Look up the network to attach the route to
	network, err := config.clientCompute.Networks.Get(
		config.Project, d.Get("network").(string)).Do()
	if err != nil {
		return fmt.Errorf("Error reading network: %s", err)
	}

	// Next hop data
	var nextHopInstance, nextHopIp, nextHopNetwork, nextHopGateway,
		nextHopVpnTunnel string
	if v, ok := d.GetOk("next_hop_ip"); ok {
		nextHopIp = v.(string)
	}
	if v, ok := d.GetOk("next_hop_gateway"); ok {
		nextHopGateway = v.(string)
	}
	if v, ok := d.GetOk("next_hop_vpn_tunnel"); ok {
		nextHopVpnTunnel = v.(string)
	}
	if v, ok := d.GetOk("next_hop_instance"); ok {
		nextInstance, err := config.clientCompute.Instances.Get(
			config.Project,
			d.Get("next_hop_instance_zone").(string),
			v.(string)).Do()
		if err != nil {
			return fmt.Errorf("Error reading instance: %s", err)
		}

		nextHopInstance = nextInstance.SelfLink
	}
	if v, ok := d.GetOk("next_hop_network"); ok {
		nextNetwork, err := config.clientCompute.Networks.Get(
			config.Project, v.(string)).Do()
		if err != nil {
			return fmt.Errorf("Error reading network: %s", err)
		}

		nextHopNetwork = nextNetwork.SelfLink
	}

	// Tags
	var tags []string
	if v := d.Get("tags").(*schema.Set); v.Len() > 0 {
		tags = make([]string, v.Len())
		for i, v := range v.List() {
			tags[i] = v.(string)
		}
	}

	// Build the route parameter
	route := &compute.Route{
		Name:             d.Get("name").(string),
		DestRange:        d.Get("dest_range").(string),
		Network:          network.SelfLink,
		NextHopInstance:  nextHopInstance,
		NextHopVpnTunnel: nextHopVpnTunnel,
		NextHopIp:        nextHopIp,
		NextHopNetwork:   nextHopNetwork,
		NextHopGateway:   nextHopGateway,
		Priority:         int64(d.Get("priority").(int)),
		Tags:             tags,
	}
	log.Printf("[DEBUG] Route insert request: %#v", route)
	op, err := config.clientCompute.Routes.Insert(
		config.Project, route).Do()
	if err != nil {
		return fmt.Errorf("Error creating route: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(route.Name)

	err = computeOperationWaitGlobal(config, op, "Creating Route")
	if err != nil {
		return err
	}

	return resourceComputeRouteRead(d, meta)
}

func resourceComputeRouteRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	route, err := config.clientCompute.Routes.Get(
		config.Project, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Route %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading route: %#v", err)
	}

	d.Set("self_link", route.SelfLink)

	return nil
}

func resourceComputeRouteDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Delete the route
	op, err := config.clientCompute.Routes.Delete(
		config.Project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting route: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, "Deleting Route")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
