package google

import (
	"bytes"
	"fmt"
	"log"
	"sort"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeFirewall() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeFirewallCreate,
		Read:   resourceComputeFirewallRead,
		Update: resourceComputeFirewallUpdate,
		Delete: resourceComputeFirewallDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"allow": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"ports": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
				Set: resourceComputeFirewallAllowHash,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"source_ranges": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"source_tags": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"target_tags": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceComputeFirewallAllowHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["protocol"].(string)))

	// We need to make sure to sort the strings below so that we always
	// generate the same hash code no matter what is in the set.
	if v, ok := m["ports"]; ok {
		vs := v.(*schema.Set).List()
		s := make([]string, len(vs))
		for i, raw := range vs {
			s[i] = raw.(string)
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}

	return hashcode.String(buf.String())
}

func resourceComputeFirewallCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	firewall, err := resourceFirewall(d, meta)
	if err != nil {
		return err
	}

	op, err := config.clientCompute.Firewalls.Insert(
		project, firewall).Do()
	if err != nil {
		return fmt.Errorf("Error creating firewall: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(firewall.Name)

	err = computeOperationWaitGlobal(config, op, "Creating Firewall")
	if err != nil {
		return err
	}

	return resourceComputeFirewallRead(d, meta)
}

func resourceComputeFirewallRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	firewall, err := config.clientCompute.Firewalls.Get(
		project, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore
			log.Printf("[WARN] Removing Firewall %q because it's gone", d.Get("name").(string))
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading firewall: %s", err)
	}

	d.Set("self_link", firewall.SelfLink)

	return nil
}

func resourceComputeFirewallUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	d.Partial(true)

	firewall, err := resourceFirewall(d, meta)
	if err != nil {
		return err
	}

	op, err := config.clientCompute.Firewalls.Update(
		project, d.Id(), firewall).Do()
	if err != nil {
		return fmt.Errorf("Error updating firewall: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, "Updating Firewall")
	if err != nil {
		return err
	}

	d.Partial(false)

	return resourceComputeFirewallRead(d, meta)
}

func resourceComputeFirewallDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Delete the firewall
	op, err := config.clientCompute.Firewalls.Delete(
		project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting firewall: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, "Deleting Firewall")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func resourceFirewall(
	d *schema.ResourceData,
	meta interface{}) (*compute.Firewall, error) {
	config := meta.(*Config)

	project, _ := getProject(d, config)

	// Look up the network to attach the firewall to
	network, err := config.clientCompute.Networks.Get(
		project, d.Get("network").(string)).Do()
	if err != nil {
		return nil, fmt.Errorf("Error reading network: %s", err)
	}

	// Build up the list of allowed entries
	var allowed []*compute.FirewallAllowed
	if v := d.Get("allow").(*schema.Set); v.Len() > 0 {
		allowed = make([]*compute.FirewallAllowed, 0, v.Len())
		for _, v := range v.List() {
			m := v.(map[string]interface{})

			var ports []string
			if v := m["ports"].(*schema.Set); v.Len() > 0 {
				ports = make([]string, v.Len())
				for i, v := range v.List() {
					ports[i] = v.(string)
				}
			}

			allowed = append(allowed, &compute.FirewallAllowed{
				IPProtocol: m["protocol"].(string),
				Ports:      ports,
			})
		}
	}

	// Build up the list of sources
	var sourceRanges, sourceTags []string
	if v := d.Get("source_ranges").(*schema.Set); v.Len() > 0 {
		sourceRanges = make([]string, v.Len())
		for i, v := range v.List() {
			sourceRanges[i] = v.(string)
		}
	}
	if v := d.Get("source_tags").(*schema.Set); v.Len() > 0 {
		sourceTags = make([]string, v.Len())
		for i, v := range v.List() {
			sourceTags[i] = v.(string)
		}
	}

	// Build up the list of targets
	var targetTags []string
	if v := d.Get("target_tags").(*schema.Set); v.Len() > 0 {
		targetTags = make([]string, v.Len())
		for i, v := range v.List() {
			targetTags[i] = v.(string)
		}
	}

	// Build the firewall parameter
	return &compute.Firewall{
		Name:         d.Get("name").(string),
		Description:  d.Get("description").(string),
		Network:      network.SelfLink,
		Allowed:      allowed,
		SourceRanges: sourceRanges,
		SourceTags:   sourceTags,
		TargetTags:   targetTags,
	}, nil
}
