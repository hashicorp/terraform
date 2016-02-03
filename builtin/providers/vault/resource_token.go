package vault

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/vault/api"
)

func resourceVaultToken() *schema.Resource {
	return &schema.Resource{
		Create: resourceVaultTokenCreate,
		Read:   resourceVaultTokenRead,
		Delete: resourceVaultTokenDelete,

		Schema: map[string]*schema.Schema{
			"display_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "token",
			},

			"ttl": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"num_uses": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"policies": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"no_default_policy": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},

			"meta": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceVaultTokenCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	policyList := d.Get("policies").(*schema.Set).List()
	policies := make([]string, 0, len(policyList))
	for _, policyName := range policyList {
		policies = append(policies, policyName.(string))
	}

	metadata := make(map[string]string)
	for k, v := range d.Get("meta").(map[string]interface{}) {
		metadata[k] = v.(string)
	}

	opts := &api.TokenCreateRequest{
		DisplayName:     d.Get("display_name").(string),
		TTL:             d.Get("ttl").(string),
		NumUses:         d.Get("num_uses").(int),
		Policies:        policies,
		NoDefaultPolicy: d.Get("no_default_policy").(bool),
		Metadata:        metadata,
	}
	token, err := client.Auth().Token().Create(opts)
	if err != nil {
		return fmt.Errorf("Error creating policy: %s", err)
	}

	if token.Auth == nil {
		return fmt.Errorf("Got response with no auth information: %#v", token)
	}

	d.SetId(token.Auth.ClientToken)
	if err := d.Set("policies", token.Auth.Policies); err != nil {
		return err
	}

	return nil
}

func resourceVaultTokenRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	_, err := client.Auth().Token().Lookup(d.Id())
	if err != nil {
		if isTokenNotFoundError(err) {
			log.Printf("[WARN] %q seems to be gone, removing from state.", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	return nil
}

func isTokenNotFoundError(err error) bool {
	return strings.Contains(err.Error(), "bad token")
}

func resourceVaultTokenDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	return client.Auth().Token().RevokeTree(d.Id())
}
