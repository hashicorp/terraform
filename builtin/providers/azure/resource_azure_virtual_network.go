package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAzureVirtualNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureVirtualNetworkCreateChoiceFunc,
		Read:   resourceAzureVirtualNetworkReadChoiceFunc,
		Update: resourceAzureVirtualNetworkUpdateChoiceFunc,
		Delete: resourceAzureVirtualNetworkDeleteChoiceFunc,

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

			"address_space": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"dns_servers_names": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"subnet": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"address_prefix": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"security_group": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceAzureSubnetHash,
			},

			"location": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			// ARM-specific fields:
			"resource_group_name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"use_asm_api"},
			},
		},
	}
}
