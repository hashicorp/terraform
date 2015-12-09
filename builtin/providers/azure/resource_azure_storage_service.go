package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAzureStorageService returns the *schema.Resource associated
// to an Azure hosted service.
func resourceAzureStorageService() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureStorageServiceCreateChoiceFunc,
		Read:   resourceAzureStorageServiceReadChoiceFunc,
		Exists: resourceAzureStorageServiceExistsChoiceFunc,
		Delete: resourceAzureStorageServiceDeleteChoiceFunc,

		Schema: map[string]*schema.Schema{
			// General attributes:
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				// TODO(aznashwan): constrain name in description
				Description: parameterDescriptions["name"],
			},
			"use_asm_api": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"location": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["location"],
			},
			"label": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "Made by Terraform.",
				Description: parameterDescriptions["label"],
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: parameterDescriptions["description"],
			},
			// Functional attributes:
			"account_type": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["account_type"],
			},
			"affinity_group": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: parameterDescriptions["affinity_group"],
			},
			"properties": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem:     schema.TypeString,
			},
			// Computed attributes:
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"primary_key": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"secondary_key": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
