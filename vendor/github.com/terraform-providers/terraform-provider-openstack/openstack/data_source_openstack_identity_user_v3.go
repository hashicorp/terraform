package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceIdentityUserV3() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceIdentityUserV3Read,

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"default_project_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"domain_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"idp_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"password_expires_at": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validatePasswordExpiresAtQuery,
			},

			"protocol_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"unique_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

// dataSourceIdentityUserV3Read performs the user lookup.
func dataSourceIdentityUserV3Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	identityClient, err := config.identityV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack identity client: %s", err)
	}

	enabled := d.Get("enabled").(bool)
	listOpts := users.ListOpts{
		DomainID:          d.Get("domain_id").(string),
		Enabled:           &enabled,
		IdPID:             d.Get("idp_id").(string),
		Name:              d.Get("name").(string),
		PasswordExpiresAt: d.Get("password_expires_at").(string),
		ProtocolID:        d.Get("protocol_id").(string),
		UniqueID:          d.Get("unique_id").(string),
	}

	log.Printf("[DEBUG] openstack_identity_user_v3 list options: %#v", listOpts)

	var user users.User
	allPages, err := users.List(identityClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("Unable to query openstack_identity_user_v3: %s", err)
	}

	allUsers, err := users.ExtractUsers(allPages)
	if err != nil {
		return fmt.Errorf("Unable to retrieve openstack_identity_user_v3: %s", err)
	}

	if len(allUsers) < 1 {
		return fmt.Errorf("Your openstack_identity_user_v3 query returned no results. " +
			"Please change your search criteria and try again.")
	}

	if len(allUsers) > 1 {
		return fmt.Errorf("Your openstack_identity_user_v3 query returned more than one result.")
	}

	user = allUsers[0]

	return dataSourceIdentityUserV3Attributes(d, &user)
}

// dataSourceIdentityUserV3Attributes populates the fields of an User resource.
func dataSourceIdentityUserV3Attributes(d *schema.ResourceData, user *users.User) error {
	log.Printf("[DEBUG] openstack_identity_user_v3 details: %#v", user)

	d.SetId(user.ID)
	d.Set("default_project_id", user.DefaultProjectID)
	d.Set("description", user.Description)
	d.Set("domain_id", user.DomainID)
	d.Set("enabled", user.Enabled)
	d.Set("name", user.Name)
	d.Set("password_expires_at", user.PasswordExpiresAt.Format(time.RFC3339))

	return nil
}
