package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/roles"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceIdentityRoleV3() *schema.Resource {
	return &schema.Resource{
		Create: resourceIdentityRoleV3Create,
		Read:   resourceIdentityRoleV3Read,
		Update: resourceIdentityRoleV3Update,
		Delete: resourceIdentityRoleV3Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"domain_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceIdentityRoleV3Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	identityClient, err := config.identityV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack identity client: %s", err)
	}

	createOpts := roles.CreateOpts{
		DomainID: d.Get("domain_id").(string),
		Name:     d.Get("name").(string),
	}

	log.Printf("[DEBUG] openstack_identity_role_v3 create options: %#v", createOpts)
	role, err := roles.Create(identityClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating openstack_identity_role_v3: %s", err)
	}

	d.SetId(role.ID)

	return resourceIdentityRoleV3Read(d, meta)
}

func resourceIdentityRoleV3Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	identityClient, err := config.identityV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack identity client: %s", err)
	}

	role, err := roles.Get(identityClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "Error retrieving openstack_identity_role_v3")
	}

	log.Printf("[DEBUG] Retrieved openstack_identity_role_v3: %#v", role)

	d.Set("domain_id", role.DomainID)
	d.Set("name", role.Name)
	d.Set("region", GetRegion(d, config))

	return nil
}

func resourceIdentityRoleV3Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	identityClient, err := config.identityV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack identity client: %s", err)
	}

	var hasChange bool
	var updateOpts roles.UpdateOpts

	if d.HasChange("name") {
		hasChange = true
		updateOpts.Name = d.Get("name").(string)
	}

	if hasChange {
		_, err := roles.Update(identityClient, d.Id(), updateOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error updating openstack_identity_role_v3 %s: %s", d.Id(), err)
		}
	}

	return resourceIdentityRoleV3Read(d, meta)
}

func resourceIdentityRoleV3Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	identityClient, err := config.identityV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack identity client: %s", err)
	}

	err = roles.Delete(identityClient, d.Id()).ExtractErr()
	if err != nil {
		return CheckDeleted(d, err, "Error deleting openstack_identity_role_v3")
	}

	return nil
}
