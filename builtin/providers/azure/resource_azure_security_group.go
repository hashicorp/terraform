package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAzureSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureSecurityGroupCreateChoiceFunc,
		Read:   resourceAzureSecurityGroupReadChoiceFunc,
		Delete: resourceAzureSecurityGroupDeleteChoiceFunc,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"use_asm_api": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"label": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"location": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}
