package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/fwaas/policies"
)

func resourceFWPolicyV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceFirewallPolicyCreate,
		Read:   resourceFirewallPolicyRead,
		Update: resourceFirewallPolicyUpdate,
		Delete: resourceFirewallPolicyDelete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFunc("OS_REGION_NAME"),
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"audited": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"shared": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"rules": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
		},
	}
}

func resourceFirewallPolicyCreate(d *schema.ResourceData, meta interface{}) error {

	// TODO To remove
	time.Sleep(time.Second * 5)

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	v := d.Get("rules").(*schema.Set)

	log.Printf("[DEBUG] Rules found : %#v", v)
	log.Printf("[DEBUG] Rules count : %d", v.Len())

	rules := make([]string, v.Len())
	for i, v := range v.List() {
		rules[i] = v.(string)
	}

	audited := d.Get("audited").(bool)
	shared := d.Get("shared").(bool)

	opts := policies.CreateOpts{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Audited:     &audited,
		Shared:      &shared,
		TenantID:    d.Get("tenant_id").(string),
		Rules:       rules,
	}

	log.Printf("[DEBUG] Create firewall policy: %#v", opts)

	policy, err := policies.Create(networkingClient, opts).Extract()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Firewall policy craeted: %#v", policy)

	d.SetId(policy.ID)

	return nil
}

func resourceFirewallPolicyRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Retrieve information about firewall policy: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	policy, err := policies.Get(networkingClient, d.Id()).Extract()

	if err != nil {
		httpError, ok := err.(*perigee.UnexpectedResponseCodeError)
		if !ok {
			return err
		}
		if httpError.Actual == 404 {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", policy.Name)
	d.Set("description", policy.Description)
	d.Set("shared", policy.Shared)
	d.Set("audited", policy.Audited)
	d.Set("tenant_id", policy.TenantID)

	return nil
}

func resourceFirewallPolicyUpdate(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	opts := policies.UpdateOpts{}

	if d.HasChange("name") {
		name := d.Get("name").(string)
		opts.Name = &name
	}

	if d.HasChange("description") {
		description := d.Get("description").(string)
		opts.Description = &description
	}

	if d.HasChange("rules") {
		v := d.Get("rules").(*schema.Set)

		log.Printf("[DEBUG] Rules found : %#v", v)
		log.Printf("[DEBUG] Rules count : %d", v.Len())

		rules := make([]string, v.Len())
		for i, v := range v.List() {
			rules[i] = v.(string)
		}
		opts.Rules = rules
	}

	log.Printf("[DEBUG] Updating firewall policy with id %s: %#v", d.Id(), opts)

	return policies.Update(networkingClient, d.Id(), opts).Err
}

func resourceFirewallPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Destroy firewall policy: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	for i := 0; i < 15; i++ {

		err = policies.Delete(networkingClient, d.Id()).Err
		if err == nil {
			break
		}

		httpError, ok := err.(*perigee.UnexpectedResponseCodeError)
		if !ok || httpError.Actual != 409 {
			return err
		}

		// This error usualy means that the policy is attached
		// to a firewall. At this point, the firewall is probably
		// being delete. So, we retry a few times.

		time.Sleep(time.Second * 2)
	}

	return err
}
