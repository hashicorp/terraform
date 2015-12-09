package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAzureDnsServer returns the *schema.Resource associated
// to an Azure hosted service.
func resourceAzureDnsServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureDnsServerCreateChoiceFunc,
		Read:   resourceAzureDnsServerReadChoiceFunc,
		Update: resourceAzureDnsServerUpdateChoiceFunc,
		Exists: resourceAzureDnsServerExistsChoiceFunc,
		Delete: resourceAzureDnsServerDeleteChoiceFunc,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: parameterDescriptions["name"],
			},
			"use_asm_api": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"dns_address": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["dns_address"],
			},
		},
	}
}
