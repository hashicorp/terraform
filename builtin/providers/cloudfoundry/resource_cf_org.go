package cloudfoundry

import (
	"fmt"

	"github.com/hashicorp/terraform/builtin/providers/cloudfoundry/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceOrg() *schema.Resource {

	return &schema.Resource{

		Create: resourceOrgCreate,
		Read:   resourceOrgRead,
		Update: resourceOrgUpdate,
		Delete: resourceOrgDelete,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"members": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      resourceStringHash,
			},
			"managers": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      resourceStringHash,
			},
			"billing_managers": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      resourceStringHash,
			},
			"auditors": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      resourceStringHash,
			},
			"quota": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceOrgCreate(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	var (
		name, quota string
		org         cfapi.CCOrg
	)
	name = d.Get("name").(string)
	if v, ok := d.GetOk("quota"); ok {
		quota = v.(string)
	}

	om := session.OrgManager()
	if org, err = om.CreateOrg(name, quota); err != nil {
		return err
	}
	if len(quota) == 0 {
		d.Set("quota", org.QuotaGUID)
	}
	d.SetId(org.ID)
	return resourceOrgUpdate(d, NewResourceMeta{meta})
}

func resourceOrgRead(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	id := d.Id()
	om := session.OrgManager()

	var (
		org cfapi.CCOrg

		members, managers, billingManagers, auditors []interface{}
	)

	if org, err = om.ReadOrg(id); err != nil {
		return
	}
	if members, err = om.ListUsers(id, cfapi.OrgRoleMember); err != nil {
		return
	}
	if managers, err = om.ListUsers(id, cfapi.OrgRoleManager); err != nil {
		return
	}
	if billingManagers, err = om.ListUsers(id, cfapi.OrgRoleBillingManager); err != nil {
		return
	}
	if auditors, err = om.ListUsers(id, cfapi.OrgRoleAuditor); err != nil {
		return
	}

	d.Set("name", org.Name)
	d.Set("quota", org.QuotaGUID)
	d.Set("members", schema.NewSet(resourceStringHash, members))
	d.Set("managers", schema.NewSet(resourceStringHash, managers))
	d.Set("billing_managers", schema.NewSet(resourceStringHash, billingManagers))
	d.Set("auditors", schema.NewSet(resourceStringHash, auditors))
	return
}

func resourceOrgUpdate(d *schema.ResourceData, meta interface{}) (err error) {

	var (
		newResource bool
		session     *cfapi.Session
	)

	if m, ok := meta.(NewResourceMeta); ok {
		session = m.meta.(*cfapi.Session)
		newResource = true
	} else {
		session = meta.(*cfapi.Session)
		if session == nil {
			return fmt.Errorf("client is nil")
		}
		newResource = false
	}

	id := d.Id()
	om := session.OrgManager()

	if !newResource {

		org := cfapi.CCOrg{
			ID:   id,
			Name: d.Get("name").(string),
		}
		if v, ok := d.GetOk("quota"); ok {
			org.QuotaGUID = v.(string)
		}

		if err = om.UpdateOrg(org); err != nil {
			return err
		}
	}

	old, new := d.GetChange("members")
	remove, add := getListChanges(old, new)
	if err = om.RemoveUsers(id, remove, cfapi.OrgRoleMember); err != nil {
		return
	}
	if err = om.AddUsers(id, add, cfapi.OrgRoleMember); err != nil {
		return
	}

	old, new = d.GetChange("managers")
	remove, add = getListChanges(old, new)
	if err = om.RemoveUsers(id, remove, cfapi.OrgRoleManager); err != nil {
		return
	}
	if err = om.AddUsers(id, add, cfapi.OrgRoleManager); err != nil {
		return
	}

	old, new = d.GetChange("billing_managers")
	remove, add = getListChanges(old, new)
	if err = om.RemoveUsers(id, remove, cfapi.OrgRoleBillingManager); err != nil {
		return
	}
	if err = om.AddUsers(id, add, cfapi.OrgRoleBillingManager); err != nil {
		return
	}

	old, new = d.GetChange("auditors")
	remove, add = getListChanges(old, new)
	if err = om.RemoveUsers(id, remove, cfapi.OrgRoleAuditor); err != nil {
		return
	}
	err = om.AddUsers(id, add, cfapi.OrgRoleAuditor)

	return
}

func resourceOrgDelete(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	err = session.OrgManager().DeleteOrg(d.Id())
	return
}
