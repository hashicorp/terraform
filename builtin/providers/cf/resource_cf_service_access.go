package cloudfoundry

import (
	"fmt"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceServiceAccess() *schema.Resource {

	return &schema.Resource{

		Create: resourceServiceAccessCreate,
		Read:   resourceServiceAccessRead,
		Update: resourceServiceAccessUpdate,
		Delete: resourceServiceAccessDelete,

		Schema: map[string]*schema.Schema{

			"plan": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"org": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceServiceAccessCreate(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	plan := d.Get("plan").(string)
	org := d.Get("org").(string)

	sm := session.ServiceManager()

	var id string
	if id, err = sm.CreateServicePlanAccess(plan, org); err != nil {
		return
	}

	d.SetId(id)
	return nil
}

func resourceServiceAccessRead(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	sm := session.ServiceManager()

	var plan, org string
	if plan, org, err = sm.ReadServicePlanAccess(d.Id()); err != nil {
		d.SetId("")
		return
	}

	d.Set("plan", plan)
	d.Set("org", org)

	return nil
}

func resourceServiceAccessUpdate(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	sm := session.ServiceManager()

	id := d.Id()
	plan := d.Get("plan").(string)
	org := d.Get("org").(string)

	if err = sm.UpdateServicePlanAccess(id, plan, org); err != nil {
		return
	}

	d.SetId(id)
	return nil
}

func resourceServiceAccessDelete(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	sm := session.ServiceManager()
	err = sm.DeleteServicePlanAccess(d.Id())
	return nil
}
