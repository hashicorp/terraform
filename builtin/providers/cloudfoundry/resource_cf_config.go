package cloudfoundry

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceConfig() *schema.Resource {

	return &schema.Resource{

		Create: resourceConfigCreate,
		Read:   resourceConfigRead,
		Update: resourceConfigUpdate,
		Delete: resourceConfigDelete,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			// Feature flags - valid only when name = 'feature_flags'
			"user_org_creation": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"private_domain_creation": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"app_bits_upload": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"app_scaling": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"route_creation": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"service_instance_creation": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"diego_docker": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"set_roles_by_username": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"unset_roles_by_username": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"task_creation": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"env_var_visibility": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"space_scoped_private_broker_creation": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"space_developer_env_var_visibility": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceConfigCreate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceConfigRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceConfigUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceConfigDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
