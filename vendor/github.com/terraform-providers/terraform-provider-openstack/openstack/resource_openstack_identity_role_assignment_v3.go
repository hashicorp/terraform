package openstack

import (
	"fmt"
	"log"

	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/roles"
	"github.com/gophercloud/gophercloud/pagination"
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
			"domain_id": &schema.Schema{
				Type:          schema.TypeString,
				ConflictsWith: []string{"project_id"},
				Optional:      true,
				ForceNew:      true,
			},

			"group_id": &schema.Schema{
				Type:          schema.TypeString,
				ConflictsWith: []string{"user_id"},
				Optional:      true,
				ForceNew:      true,
			},

			"project_id": &schema.Schema{
				Type:          schema.TypeString,
				ConflictsWith: []string{"domain_id"},
				Optional:      true,
				ForceNew:      true,
			},

			"role_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"user_id": &schema.Schema{
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

	domainID := d.Get("domain_id").(string)
	groupID := d.Get("group_id").(string)
	projectID := d.Get("project_id").(string)
	roleID := d.Get("role_id").(string)
	userID := d.Get("user_id").(string)
	opts := roles.AssignOpts{
		DomainID:  domainID,
		GroupID:   groupID,
		ProjectID: projectID,
		UserID:    userID,
	}

	err = roles.Assign(identityClient, roleID, opts).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error assigning role: %s", err)
	}

	d.SetId(buildRoleAssignmentID(domainID, projectID, groupID, userID, roleID))

	return resourceIdentityRoleAssignmentV3Read(d, meta)
}

func resourceIdentityRoleAssignmentV3Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	identityClient, err := config.identityV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack identity client: %s", err)
	}

	roleAssignment, err := getRoleAssignment(identityClient, d)
	if err != nil {
		return fmt.Errorf("Error getting role assignment: %s", err)
	}

	log.Printf("[DEBUG] Retrieved OpenStack role assignment: %#v", roleAssignment)
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

	domainID, projectID, groupID, userID, roleID := extractRoleAssignmentID(d.Id())
	var opts roles.UnassignOpts
	opts = roles.UnassignOpts{
		DomainID:  domainID,
		GroupID:   groupID,
		ProjectID: projectID,
		UserID:    userID,
	}
	roles.Unassign(identityClient, roleID, opts).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error unassigning role: %s", err)
	}

	return nil
}

func getRoleAssignment(identityClient *gophercloud.ServiceClient, d *schema.ResourceData) (roles.RoleAssignment, error) {
	domainID, projectID, groupID, userID, roleID := extractRoleAssignmentID(d.Id())

	var opts roles.ListAssignmentsOpts
	opts = roles.ListAssignmentsOpts{
		GroupID:        groupID,
		ScopeDomainID:  domainID,
		ScopeProjectID: projectID,
		UserID:         userID,
	}

	pager := roles.ListAssignments(identityClient, opts)
	var assignment roles.RoleAssignment

	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		assignmentList, err := roles.ExtractRoleAssignments(page)
		if err != nil {
			return false, err
		}

		for _, a := range assignmentList {
			if a.Role.ID == roleID {
				assignment = a
				return false, nil
			}
		}

		return true, nil
	})

	return assignment, err
}

// Role assignments have no ID in OpenStack. Build an ID out of the IDs that make up the role assignment
func buildRoleAssignmentID(domainID, projectID, groupID, userID, roleID string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", domainID, projectID, groupID, userID, roleID)
}

func extractRoleAssignmentID(roleAssignmentID string) (string, string, string, string, string) {
	split := strings.Split(roleAssignmentID, "/")
	return split[0], split[1], split[2], split[3], split[4]
}
