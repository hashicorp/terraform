package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
)

func resourceComputeForwardingRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeForwardingRuleCreate,
		Read:   resourceComputeForwardingRuleRead,
		Delete: resourceComputeForwardingRuleDelete,
		Update: resourceComputeForwardingRuleUpdate,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"target": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"backend_service": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"ip_protocol": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"load_balancing_scheme": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "EXTERNAL",
			},

			"network": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"port_range": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == new+"-"+new {
						return true
					}
					return false
				},
			},

			"ports": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				ForceNew: true,
				Set:      schema.HashString,
				MaxItems: 5,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"subnetwork": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
		},
	}
}

func resourceComputeForwardingRuleCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	ps := d.Get("ports").(*schema.Set).List()
	ports := make([]string, 0, len(ps))
	for _, v := range ps {
		ports = append(ports, v.(string))
	}

	frule := &compute.ForwardingRule{
		BackendService:      d.Get("backend_service").(string),
		IPAddress:           d.Get("ip_address").(string),
		IPProtocol:          d.Get("ip_protocol").(string),
		Description:         d.Get("description").(string),
		LoadBalancingScheme: d.Get("load_balancing_scheme").(string),
		Name:                d.Get("name").(string),
		Network:             d.Get("network").(string),
		PortRange:           d.Get("port_range").(string),
		Ports:               ports,
		Subnetwork:          d.Get("subnetwork").(string),
		Target:              d.Get("target").(string),
	}

	log.Printf("[DEBUG] ForwardingRule insert request: %#v", frule)
	op, err := config.clientCompute.ForwardingRules.Insert(
		project, region, frule).Do()
	if err != nil {
		return fmt.Errorf("Error creating ForwardingRule: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(frule.Name)

	err = computeOperationWaitRegion(config, op, project, region, "Creating Fowarding Rule")
	if err != nil {
		return err
	}

	return resourceComputeForwardingRuleRead(d, meta)
}

func resourceComputeForwardingRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	d.Partial(true)

	if d.HasChange("target") {
		target_name := d.Get("target").(string)
		target_ref := &compute.TargetReference{Target: target_name}
		op, err := config.clientCompute.ForwardingRules.SetTarget(
			project, region, d.Id(), target_ref).Do()
		if err != nil {
			return fmt.Errorf("Error updating target: %s", err)
		}

		err = computeOperationWaitRegion(config, op, project, region, "Updating Forwarding Rule")
		if err != nil {
			return err
		}

		d.SetPartial("target")
	}

	d.Partial(false)

	return resourceComputeForwardingRuleRead(d, meta)
}

func resourceComputeForwardingRuleRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	frule, err := config.clientCompute.ForwardingRules.Get(
		project, region, d.Id()).Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Forwarding Rule %q", d.Get("name").(string)))
	}

	d.Set("name", frule.Name)
	d.Set("target", frule.Target)
	d.Set("backend_service", frule.BackendService)
	d.Set("description", frule.Description)
	d.Set("load_balancing_scheme", frule.LoadBalancingScheme)
	d.Set("network", frule.Network)
	d.Set("port_range", frule.PortRange)
	d.Set("ports", frule.Ports)
	d.Set("project", project)
	d.Set("region", region)
	d.Set("subnetwork", frule.Subnetwork)
	d.Set("ip_address", frule.IPAddress)
	d.Set("ip_protocol", frule.IPProtocol)
	d.Set("self_link", frule.SelfLink)
	return nil
}

func resourceComputeForwardingRuleDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Delete the ForwardingRule
	log.Printf("[DEBUG] ForwardingRule delete request")
	op, err := config.clientCompute.ForwardingRules.Delete(
		project, region, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting ForwardingRule: %s", err)
	}

	err = computeOperationWaitRegion(config, op, project, region, "Deleting Forwarding Rule")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
