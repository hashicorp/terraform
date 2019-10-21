package openstack

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
)

func dataSourceIdentityAuthScopeV3() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceIdentityAuthScopeV3Read,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			// computed attributes
			"user_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"user_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"user_domain_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"user_domain_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"project_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"project_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"project_domain_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"project_domain_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"roles": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"role_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceIdentityAuthScopeV3Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	identityClient, err := config.identityV3Client(GetRegion(d, config))
	tokenID := config.OsClient.TokenID

	d.SetId(d.Get("name").(string))

	result := tokens.Get(identityClient, tokenID)
	if result.Err != nil {
		return result.Err
	}

	user, err := result.ExtractUser()
	if err != nil {
		return err
	}

	d.Set("user_name", user.Name)
	d.Set("user_id", user.Name)
	d.Set("user_domain_name", user.Domain.Name)
	d.Set("user_domain_id", user.Domain.ID)

	project, err := result.ExtractProject()
	if err != nil {
		return err
	}

	d.Set("project_name", project.Name)
	d.Set("project_id", project.ID)
	d.Set("project_domain_name", project.Domain.Name)
	d.Set("project_domain_id", project.Domain.ID)

	roles, err := result.ExtractRoles()
	if err != nil {
		return err
	}

	allRoles := flattenIdentityAuthScopeV3Roles(roles)
	if err := d.Set("roles", allRoles); err != nil {
		log.Printf("[DEBUG] Unable to set openstack_identity_auth_scope_v3 roles: %s", err)
	}

	d.Set("region", GetRegion(d, config))

	return nil
}
