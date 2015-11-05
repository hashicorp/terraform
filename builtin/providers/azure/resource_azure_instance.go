package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAzureInstance returns the *schema.Resource corresponding
// to instances on Azure.
func resourceAzureInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureInstanceCreateChoiceFunc,
		Read:   resourceAzureInstanceReadChoiceFunc,
		Update: resourceAzureInstanceUpdateChoiceFunc,
		Delete: resourceAzureInstanceDeleteChoiceFunc,

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

			"hosted_service_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			// in order to prevent an unintentional delete of a containing
			// hosted service in the case the same name are given to both the
			// service and the instance despite their being created separately,
			// we must maintain a flag to definitively denote whether this
			// instance had a hosted service created for it or not:
			"has_dedicated_service": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"image": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"size": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"subnet": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"virtual_network": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"storage_service_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"reverse_dns": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"location": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"automatic_updates": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"time_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"username": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"password": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"ssh_key_thumbprint": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"endpoint": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "tcp",
						},

						"public_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"private_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
				Set: resourceAzureEndpointHash,
			},

			"security_group": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"vip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"domain_username": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"domain_password": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"domain_ou": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}
