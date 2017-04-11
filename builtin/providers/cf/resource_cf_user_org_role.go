package cloudfoundry

import (
	"fmt"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceUserOrgRole() *schema.Resource {

	return &schema.Resource{

		Create: resourceUserOrgRoleCreate,
		Read:   resourceUserOrgRoleRead,
		Update: resourceUserOrgRoleUpdate,
		Delete: resourceUserOrgRoleDelete,

		Schema: map[string]*schema.Schema{

			"user": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"role": &schema.Schema{
				Type:     schema.TypeSet,
				Set:      orgRoleHash,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "member",
							ValidateFunc: validateOrgRoleType,
						},
						"org": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

var userOrgRoleToTypeMap = map[cfapi.UserRoleInOrg]string{
	cfapi.UserIsOrgManager:        "manager",
	cfapi.UserIsOrgBillingManager: "billing_manager",
	cfapi.UserIsOrgAuditor:        "auditor",
	cfapi.UserIsOrgMember:         "member",
}
var typeToOrgRoleMap = map[string]cfapi.OrgRole{
	"manager":         cfapi.OrgRoleManager,
	"billing_manager": cfapi.OrgRoleBillingManager,
	"auditor":         cfapi.OrgRoleAuditor,
	"member":          cfapi.OrgRoleMember,
}

func orgRoleHash(d interface{}) int {
	t := d.(map[string]interface{})["type"].(string)
	o := d.(map[string]interface{})["org"].(string)
	return hashcode.String(t + o)
}

func validateOrgRoleType(v interface{}, k string) (ws []string, errs []error) {
	value := v.(string)
	if value != "manager" &&
		value != "billing_manager" &&
		value != "auditor" &&
		value != "member" {
		errs = append(errs, fmt.Errorf("%q must be one of 'manager', 'billing_manager', 'auditor' or member", k))
	}
	return
}

func resourceUserOrgRoleCreate(d *schema.ResourceData, meta interface{}) error {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	d.SetId(fmt.Sprintf("uor-%s", d.Get("user").(string)))
	return resourceUserOrgRoleUpdate(d, meta)
}

func resourceUserOrgRoleRead(d *schema.ResourceData, meta interface{}) error {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	id := getUserIDFromUORID(d.Id())

	userOrgRoles := []interface{}{}
	userOrgRoles, err := readUserOrgAssociations(session, id, cfapi.UserIsOrgManager, userOrgRoles)
	if err != nil {
		return err
	}
	userOrgRoles, err = readUserOrgAssociations(session, id, cfapi.UserIsOrgBillingManager, userOrgRoles)
	if err != nil {
		return err
	}
	userOrgRoles, err = readUserOrgAssociations(session, id, cfapi.UserIsOrgAuditor, userOrgRoles)
	if err != nil {
		return err
	}
	userOrgRoles, err = readUserOrgAssociations(session, id, cfapi.UserIsOrgMember, userOrgRoles)
	if err != nil {
		return err
	}
	d.Set("role", userOrgRoles)
	return nil
}

func resourceUserOrgRoleUpdate(d *schema.ResourceData, meta interface{}) error {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	userID := getUserIDFromUORID(d.Id())

	oldOrgRoles, newOrgRoles := d.GetChange("role")
	orgsRolesToDelete, orgsRolesToAdd := getListChangedSchemaLists(oldOrgRoles, newOrgRoles)

	if len(orgsRolesToDelete) > 0 || len(orgsRolesToAdd) > 0 {
		om := session.OrgManager()

		for _, orgRole := range orgsRolesToAdd {

			t := orgRole["type"].(string)
			o := orgRole["org"].(string)

			session.Log.DebugMessage(
				"associating user '%s' with org '%s' with role '%s'", userID, o, t)

			err := om.AddUsers(o, []string{userID}, typeToOrgRoleMap[t])
			if err != nil {
				return err
			}
		}
		for _, orgRole := range orgsRolesToDelete {

			t := orgRole["type"].(string)
			o := orgRole["org"].(string)

			session.Log.DebugMessage(
				"removing user '%s's role '%s' from org '%s'", userID, t, o)

			err := om.RemoveUsers(o, []string{userID}, typeToOrgRoleMap[t])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func resourceUserOrgRoleDelete(d *schema.ResourceData, meta interface{}) error {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	userID := getUserIDFromUORID(d.Id())

	roles, ok := d.GetOk("role")
	if ok {
		om := session.OrgManager()

		for _, o := range roles.(*schema.Set).List() {

			orgRole := o.(map[string]interface{})
			t := orgRole["type"].(string)
			o := orgRole["org"].(string)

			err := om.RemoveUsers(o, []string{userID}, typeToOrgRoleMap[t])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getUserIDFromUORID(uorID string) string {
	return uorID[4:len(uorID)]
}

func readUserOrgAssociations(session *cfapi.Session,
	userID string, role cfapi.UserRoleInOrg,
	userOrgRoles []interface{}) ([]interface{}, error) {

	orgIDs, err := session.UserManager().ListOrgsForUser(userID, role)
	if err != nil {
		return nil, err
	}
	for _, o := range orgIDs {

		userOrgRole := make(map[string]interface{})
		userOrgRole["type"] = userOrgRoleToTypeMap[role]
		userOrgRole["org"] = o

		userOrgRoles = append(userOrgRoles, userOrgRole)
	}
	return userOrgRoles, nil
}
