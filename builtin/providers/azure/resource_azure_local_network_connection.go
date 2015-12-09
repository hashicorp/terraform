package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAzureLocalNetworkConnection returns the schema.Resource
// associated to an Azure local network connection.
func resourceAzureLocalNetworkConnection() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureLocalNetworkConnectionCreateChoiceFunc,
		Read:   resourceAzureLocalNetworkConnectionReadChoiceFunc,
		Update: resourceAzureLocalNetworkConnectionUpdateChoiceFunc,
		Exists: resourceAzureLocalNetworkConnectionExistsChoiceFunc,
		Delete: resourceAzureLocalNetworkConnectionDeleteChoiceFunc,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["name"],
			},

			"location": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"use_asm_api"},
			},

			"resource_group_name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"use_asm_api"},
			},

			"use_asm_api": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"vpn_gateway_address": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["vpn_gateway_address"],
			},

			"address_space_prefixes": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: parameterDescriptions["address_space_prefixes"],
			},
		},
	}
}
