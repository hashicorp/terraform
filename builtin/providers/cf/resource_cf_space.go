package cloudfoundry

import (
	"fmt"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceSpace() *schema.Resource {

	return &schema.Resource{

		Create: resourceSpaceCreate,
		Read:   resourceSpaceRead,
		Update: resourceSpaceUpdate,
		Delete: resourceSpaceDelete,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"org": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"quota": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"asgs": &schema.Schema{
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
			"developers": &schema.Schema{
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
			"allow_ssh": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func resourceSpaceCreate(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	var (
		name, org, quota string
		allowSSH         bool
		asgs             []interface{}
	)
	name = d.Get("name").(string)
	org = d.Get("org").(string)
	if v, ok := d.GetOk("quota"); ok {
		quota = v.(string)
	}
	if v, ok := d.GetOk("asgs"); ok {
		asgs = v.(*schema.Set).List()
	}
	allowSSH = d.Get("allow_ssh").(bool)

	var id string

	sm := session.SpaceManager()
	if id, err = sm.CreateSpace(name, org, quota, allowSSH, asgs); err != nil {
		return err
	}
	d.SetId(id)
	return resourceSpaceUpdate(d, NewResourceMeta{meta})
}

func resourceSpaceRead(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	id := d.Id()
	sm := session.SpaceManager()

	var (
		space cfapi.CCSpace

		runningAsgs                    []string
		spaceAsgs, asgs                []interface{}
		managers, developers, auditors []interface{}
	)

	if space, err = sm.ReadSpace(id); err != nil {
		return
	}
	if managers, err = sm.ListUsers(id, cfapi.SpaceRoleManager); err != nil {
		return
	}
	if developers, err = sm.ListUsers(id, cfapi.SpaceRoleDeveloper); err != nil {
		return
	}
	if auditors, err = sm.ListUsers(id, cfapi.SpaceRoleAuditor); err != nil {
		return
	}

	if runningAsgs, err = session.ASGManager().Running(); err != nil {
		return err
	}
	if spaceAsgs, err = sm.ListASGs(id); err != nil {
		return
	}
	for _, a := range spaceAsgs {
		if !isStringInList(runningAsgs, a.(string)) {
			asgs = append(asgs, a)
		}
	}

	d.Set("name", space.Name)
	d.Set("org", space.OrgGUID)
	d.Set("quota", space.QuotaGUID)
	d.Set("asgs", schema.NewSet(resourceStringHash, asgs))
	d.Set("managers", schema.NewSet(resourceStringHash, managers))
	d.Set("developers", schema.NewSet(resourceStringHash, developers))
	d.Set("auditors", schema.NewSet(resourceStringHash, auditors))
	d.Set("allow_ssh", space.AllowSSH)
	return
}

func resourceSpaceUpdate(d *schema.ResourceData, meta interface{}) (err error) {

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
	sm := session.SpaceManager()

	if !newResource {

		var asgs []interface{}

		space := cfapi.CCSpace{
			ID:      d.Id(),
			Name:    d.Get("name").(string),
			OrgGUID: d.Get("org").(string),
		}
		if v, ok := d.GetOk("quota"); ok {
			space.QuotaGUID = v.(string)
		}
		if v, ok := d.GetOk("asgs"); ok {
			asgs = v.(*schema.Set).List()
		}

		if err = sm.UpdateSpace(space, asgs); err != nil {
			return err
		}
	}

	old, new := d.GetChange("managers")
	remove, add := getListChanges(old, new)
	if err = sm.RemoveUsers(id, remove, cfapi.SpaceRoleManager); err != nil {
		return
	}
	if err = sm.AddUsers(id, add, cfapi.SpaceRoleManager); err != nil {
		return
	}

	old, new = d.GetChange("developers")
	remove, add = getListChanges(old, new)
	if err = sm.RemoveUsers(id, remove, cfapi.SpaceRoleDeveloper); err != nil {
		return
	}
	if err = sm.AddUsers(id, add, cfapi.SpaceRoleDeveloper); err != nil {
		return
	}

	old, new = d.GetChange("auditors")
	remove, add = getListChanges(old, new)
	if err = sm.RemoveUsers(id, remove, cfapi.SpaceRoleAuditor); err != nil {
		return
	}
	err = sm.AddUsers(id, add, cfapi.SpaceRoleAuditor)

	return
}

func resourceSpaceDelete(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	err = session.SpaceManager().DeleteSpace(d.Id())
	return
}
