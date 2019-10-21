package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/roles"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceIdentityRoleAssignmentV3() *schema.Resource {
	return &schema.Resource{
		Create: resourceIdentityRoleAssignmentV3Create,
		Read:   resourceIdentityRoleAssignmentV3Read,
		Delete: resourceIdentityRoleAssignmentV3Delete,
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
				Type:          schema.TypeString,
				ConflictsWith: []string{"project_id"},
				Optional:      true,
				ForceNew:      true,
			},

			"group_id": {
				Type:          schema.TypeString,
				ConflictsWith: []string{"user_id"},
				Optional:      true,
				ForceNew:      true,
			},

			"project_id": {
				Type:          schema.TypeString,
				ConflictsWith: []string{"domain_id"},
				Optional:      true,
				ForceNew:      true,
			},

			"role_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"user_id": {
				Type:          schema.TypeString,
				ConflictsWith: []string{"group_id"},
				Optional:      true,
				ForceNew:      true,
			},
		},
	}
}

func resourceIdentityRoleAssignmentV3Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	identityClient, err := config.identityV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack identity client: %s", err)
	}

	roleID := d.Get("role_id").(string)
	domainID := d.Get("domain_id").(string)
	groupID := d.Get("group_id").(string)
	projectID := d.Get("project_id").(string)
	userID := d.Get("user_id").(string)

	opts := roles.AssignOpts{
		DomainID:  domainID,
		GroupID:   groupID,
		ProjectID: projectID,
		UserID:    userID,
	}

	err = roles.Assign(identityClient, roleID, opts).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error creating openstack_identity_role_assignment_v3: %s", err)
	}

	id := identityRoleAssignmentV3ID(domainID, projectID, groupID, userID, roleID)
	d.SetId(id)

	return resourceIdentityRoleAssignmentV3Read(d, meta)
}

func resourceIdentityRoleAssignmentV3Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	identityClient, err := config.identityV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack identity client: %s", err)
	}

	roleAssignment, err := identityRoleAssignmentV3FindAssignment(identityClient, d.Id())
	if err != nil {
		return CheckDeleted(d, err, "Error retrieving openstack_identity_role_assignment_v3")
	}

	log.Printf("[DEBUG] Retrieved openstack_identity_role_assignment_v3 %s: %#v", d.Id(), roleAssignment)
	d.Set("domain_id", roleAssignment.Scope.Domain.ID)
	d.Set("project_id", roleAssignment.Scope.Project.ID)
	d.Set("group_id", roleAssignment.Group.ID)
	d.Set("user_id", roleAssignment.User.ID)
	d.Set("role_id", roleAssignment.Role.ID)
	d.Set("region", GetRegion(d, config))

	return nil
}

func resourceIdentityRoleAssignmentV3Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	identityClient, err := config.identityV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack identity client: %s", err)
	}

	domainID, projectID, groupID, userID, roleID, err := identityRoleAssignmentV3ParseID(d.Id())
	if err != nil {
		return fmt.Errorf("Error determining openstack_identity_role_assignment_v3 ID: %s", err)
	}

	opts := roles.UnassignOpts{
		DomainID:  domainID,
		GroupID:   groupID,
		ProjectID: projectID,
		UserID:    userID,
	}

	roles.Unassign(identityClient, roleID, opts).ExtractErr()
	if err != nil {
		return CheckDeleted(d, err, "Error unassigning openstack_identity_role_assignment_v3")
	}

	return nil
}
