package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAzureSecurityGroupRule returns the *schema.Resource for
// a network security group rule on Azure.
func resourceAzureSecurityGroupRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureSecurityGroupRuleCreateChoiceFunc,
		Read:   resourceAzureSecurityGroupRuleReadChoiceFunc,
		Update: resourceAzureSecurityGroupRuleUpdateChoiceFunc,
		Delete: resourceAzureSecurityGroupRuleDeleteChoiceFunc,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["name"],
			},
			"use_asm_api": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"security_group_names": &schema.Schema{
				Type:        schema.TypeSet,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["netsecgroup_secgroup_names"],
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Set: schema.HashString,
			},
			"type": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["netsecgroup_type"],
			},
			"priority": &schema.Schema{
				Type:        schema.TypeInt,
				Required:    true,
				Description: parameterDescriptions["netsecgroup_priority"],
			},
			"action": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["netsecgroup_action"],
			},
			"source_address_prefix": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["netsecgroup_src_addr_prefix"],
			},
			"source_port_range": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["netsecgroup_src_port_range"],
			},
			"destination_address_prefix": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["netsecgroup_dest_addr_prefix"],
			},
			"destination_port_range": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["netsecgroup_dest_port_range"],
			},
			"protocol": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["netsecgroup_protocol"],
			},
		},
	}
}
