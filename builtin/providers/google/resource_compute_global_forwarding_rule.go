package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
)

func resourceComputeGlobalForwardingRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeGlobalForwardingRuleCreate,
		Read:   resourceComputeGlobalForwardingRuleRead,
		Update: resourceComputeGlobalForwardingRuleUpdate,
		Delete: resourceComputeGlobalForwardingRuleDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"target": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
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

			"port_range": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"region": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				ForceNew:   true,
				Deprecated: "Please remove this attribute (it was never used)",
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceComputeGlobalForwardingRuleCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	frule := &compute.ForwardingRule{
		IPAddress:   d.Get("ip_address").(string),
		IPProtocol:  d.Get("ip_protocol").(string),
		Description: d.Get("description").(string),
		Name:        d.Get("name").(string),
		PortRange:   d.Get("port_range").(string),
		Target:      d.Get("target").(string),
	}

	op, err := config.clientCompute.GlobalForwardingRules.Insert(
		project, frule).Do()
	if err != nil {
		return fmt.Errorf("Error creating Global Forwarding Rule: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(frule.Name)

	err = computeOperationWaitGlobal(config, op, project, "Creating Global Fowarding Rule")
	if err != nil {
		return err
	}

	return resourceComputeGlobalForwardingRuleRead(d, meta)
}

func resourceComputeGlobalForwardingRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	d.Partial(true)

	if d.HasChange("target") {
		target_name := d.Get("target").(string)
		target_ref := &compute.TargetReference{Target: target_name}
		op, err := config.clientCompute.GlobalForwardingRules.SetTarget(
			project, d.Id(), target_ref).Do()
		if err != nil {
			return fmt.Errorf("Error updating target: %s", err)
		}

		err = computeOperationWaitGlobal(config, op, project, "Updating Global Forwarding Rule")
		if err != nil {
			return err
		}

		d.SetPartial("target")
	}

	d.Partial(false)

	return resourceComputeGlobalForwardingRuleRead(d, meta)
}

func resourceComputeGlobalForwardingRuleRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	frule, err := config.clientCompute.GlobalForwardingRules.Get(
		project, d.Id()).Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Global Forwarding Rule %q", d.Get("name").(string)))
	}

	d.Set("ip_address", frule.IPAddress)
	d.Set("ip_protocol", frule.IPProtocol)
	d.Set("self_link", frule.SelfLink)

	return nil
}

func resourceComputeGlobalForwardingRuleDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Delete the GlobalForwardingRule
	log.Printf("[DEBUG] GlobalForwardingRule delete request")
	op, err := config.clientCompute.GlobalForwardingRules.Delete(
		project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting GlobalForwardingRule: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, project, "Deleting GlobalForwarding Rule")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
