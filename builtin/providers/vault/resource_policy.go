package vault

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/vault/api"
)

func policyResource() *schema.Resource {
	return &schema.Resource{
		Create: policyWrite,
		Update: policyWrite,
		Delete: policyDelete,
		Read:   policyRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the policy",
			},

			"policy": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The policy document",
			},
		},
	}
}

func policyWrite(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	name := d.Get("name").(string)
	policy := d.Get("policy").(string)

	log.Printf("[DEBUG] Writing policy %s to Vault", name)
	err := client.Sys().PutPolicy(name, policy)

	if err != nil {
		return fmt.Errorf("error writing to Vault: %s", err)
	}

	d.SetId(name)

	return nil
}

func policyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	name := d.Id()

	log.Printf("[DEBUG] Deleting policy %s from Vault", name)

	err := client.Sys().DeletePolicy(name)
	if err != nil {
		return fmt.Errorf("error deleting from Vault: %s", err)
	}

	return nil
}

func policyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	name := d.Id()

	policy, err := client.Sys().GetPolicy(name)

	if err != nil {
		return fmt.Errorf("error reading from Vault: %s", err)
	}

	d.Set("policy", policy)

	return nil
}
