package vault

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/vault/api"
)

func resourceVaultPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceVaultPolicyCreate,
		Update: resourceVaultPolicyCreate,
		Read:   resourceVaultPolicyRead,
		Delete: resourceVaultPolicyDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"rules": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceVaultPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	err := client.Sys().PutPolicy(
		d.Get("name").(string),
		d.Get("rules").(string),
	)
	if err != nil {
		return fmt.Errorf("Error creating policy: %s", err)
	}

	d.SetId(d.Get("name").(string))

	return nil
}

func resourceVaultPolicyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	policies, err := client.Sys().ListPolicies()
	if err != nil {
		return fmt.Errorf("Error listing policies: %s", err)
	}

	found := false
	for _, p := range policies {
		if p == d.Id() {
			found = true
		}
	}

	if !found {
		log.Printf("[WARN] Policy not found; removing from state: %s", d.Id())
		d.SetId("")
		return nil
	}

	rules, err := client.Sys().GetPolicy(d.Id())
	if err != nil {
		return fmt.Errorf("Error getting policy: %s", err)
	}
	d.Set("rules", rules)

	return nil
}

func resourceVaultPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	err := client.Sys().DeletePolicy(d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting policy: %s", err)
	}
	return nil
}
