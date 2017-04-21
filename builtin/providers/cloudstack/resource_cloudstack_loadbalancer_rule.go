package cloudstack

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackLoadBalancerRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackLoadBalancerRuleCreate,
		Read:   resourceCloudStackLoadBalancerRuleRead,
		Update: resourceCloudStackLoadBalancerRuleUpdate,
		Delete: resourceCloudStackLoadBalancerRuleDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"ip_address_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"algorithm": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"private_port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"public_port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"member_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

func resourceCloudStackLoadBalancerRuleCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)
	d.Partial(true)

	// Create a new parameter struct
	p := cs.LoadBalancer.NewCreateLoadBalancerRuleParams(
		d.Get("algorithm").(string),
		d.Get("name").(string),
		d.Get("private_port").(int),
		d.Get("public_port").(int),
	)

	// Don't autocreate a firewall rule, use a resource if needed
	p.SetOpenfirewall(false)

	// Set the description
	if description, ok := d.GetOk("description"); ok {
		p.SetDescription(description.(string))
	} else {
		p.SetDescription(d.Get("name").(string))
	}

	if networkid, ok := d.GetOk("network_id"); ok {
		// Set the network id
		p.SetNetworkid(networkid.(string))
	}

	// Set the ipaddress id
	p.SetPublicipid(d.Get("ip_address_id").(string))

	// Create the load balancer rule
	r, err := cs.LoadBalancer.CreateLoadBalancerRule(p)
	if err != nil {
		return err
	}

	// Set the load balancer rule ID and set partials
	d.SetId(r.Id)
	d.SetPartial("name")
	d.SetPartial("description")
	d.SetPartial("ip_address_id")
	d.SetPartial("network_id")
	d.SetPartial("algorithm")
	d.SetPartial("private_port")
	d.SetPartial("public_port")

	// Create a new parameter struct
	ap := cs.LoadBalancer.NewAssignToLoadBalancerRuleParams(r.Id)

	var mbs []string
	for _, id := range d.Get("member_ids").(*schema.Set).List() {
		mbs = append(mbs, id.(string))
	}

	ap.SetVirtualmachineids(mbs)

	_, err = cs.LoadBalancer.AssignToLoadBalancerRule(ap)
	if err != nil {
		return err
	}

	d.SetPartial("member_ids")
	d.Partial(false)

	return resourceCloudStackLoadBalancerRuleRead(d, meta)
}

func resourceCloudStackLoadBalancerRuleRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the load balancer details
	lb, count, err := cs.LoadBalancer.GetLoadBalancerRuleByID(
		d.Id(),
		cloudstack.WithProject(d.Get("project").(string)),
	)
	if err != nil {
		if count == 0 {
			log.Printf("[DEBUG] Load balancer rule %s does no longer exist", d.Get("name").(string))
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("algorithm", lb.Algorithm)
	d.Set("public_port", lb.Publicport)
	d.Set("private_port", lb.Privateport)
	d.Set("ip_address_id", lb.Publicipid)

	// Only set network if user specified it to avoid spurious diffs
	if _, ok := d.GetOk("network_id"); ok {
		d.Set("network_id", lb.Networkid)
	}

	setValueOrID(d, "project", lb.Project, lb.Projectid)

	p := cs.LoadBalancer.NewListLoadBalancerRuleInstancesParams(d.Id())
	l, err := cs.LoadBalancer.ListLoadBalancerRuleInstances(p)
	if err != nil {
		return err
	}

	var mbs []string
	for _, i := range l.LoadBalancerRuleInstances {
		mbs = append(mbs, i.Id)
	}
	d.Set("member_ids", mbs)

	return nil
}

func resourceCloudStackLoadBalancerRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	if d.HasChange("name") || d.HasChange("description") || d.HasChange("algorithm") {
		name := d.Get("name").(string)

		// Create new parameter struct
		p := cs.LoadBalancer.NewUpdateLoadBalancerRuleParams(d.Id())

		if d.HasChange("name") {
			log.Printf("[DEBUG] Name has changed for load balancer rule %s, starting update", name)

			p.SetName(name)
		}

		if d.HasChange("description") {
			log.Printf(
				"[DEBUG] Description has changed for load balancer rule %s, starting update", name)

			p.SetDescription(d.Get("description").(string))
		}

		if d.HasChange("algorithm") {
			algorithm := d.Get("algorithm").(string)

			log.Printf(
				"[DEBUG] Algorithm has changed to %s for load balancer rule %s, starting update",
				algorithm,
				name,
			)

			// Set the new Algorithm
			p.SetAlgorithm(algorithm)
		}

		_, err := cs.LoadBalancer.UpdateLoadBalancerRule(p)
		if err != nil {
			return fmt.Errorf(
				"Error updating load balancer rule %s", name)
		}
	}

	if d.HasChange("member_ids") {
		o, n := d.GetChange("member_ids")
		ombs, nmbs := o.(*schema.Set), n.(*schema.Set)

		setToStringList := func(s *schema.Set) []string {
			l := make([]string, s.Len())
			for i, v := range s.List() {
				l[i] = v.(string)
			}
			return l
		}

		membersToAdd := setToStringList(nmbs.Difference(ombs))
		membersToRemove := setToStringList(ombs.Difference(nmbs))

		log.Printf("[DEBUG] Members to add: %v, remove: %v", membersToAdd, membersToRemove)

		if len(membersToAdd) > 0 {
			p := cs.LoadBalancer.NewAssignToLoadBalancerRuleParams(d.Id())
			p.SetVirtualmachineids(membersToAdd)
			if _, err := cs.LoadBalancer.AssignToLoadBalancerRule(p); err != nil {
				return err
			}
		}

		if len(membersToRemove) > 0 {
			p := cs.LoadBalancer.NewRemoveFromLoadBalancerRuleParams(d.Id())
			p.SetVirtualmachineids(membersToRemove)
			if _, err := cs.LoadBalancer.RemoveFromLoadBalancerRule(p); err != nil {
				return err
			}
		}
	}

	return resourceCloudStackLoadBalancerRuleRead(d, meta)
}

func resourceCloudStackLoadBalancerRuleDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.LoadBalancer.NewDeleteLoadBalancerRuleParams(d.Id())

	log.Printf("[INFO] Deleting load balancer rule: %s", d.Get("name").(string))
	if _, err := cs.LoadBalancer.DeleteLoadBalancerRule(p); err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if !strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return err
		}
	}

	return nil
}
