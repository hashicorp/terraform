package cloudfoundry

import (
	"fmt"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceEvg() *schema.Resource {

	return &schema.Resource{

		Create: resourceEvgCreate,
		Read:   resourceEvgRead,
		Update: resourceEvgUpdate,
		Delete: resourceEvgDelete,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateDefaultRunningStagingName,
			},
			"variables": &schema.Schema{
				Type:     schema.TypeMap,
				Required: true,
			},
		},
	}
}

func resourceEvgCreate(d *schema.ResourceData, meta interface{}) (err error) {

	if err = resourceEvgUpdate(d, meta); err != nil {
		return
	}
	d.SetId(d.Get("name").(string))
	return
}

func resourceEvgRead(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	var variables map[string]interface{}
	if variables, err = session.EVGManager().GetEVG(d.Get("name").(string)); err != nil {
		return
	}
	d.Set("variables", variables)
	return nil
}

func resourceEvgUpdate(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	name := d.Get("name").(string)
	variables := d.Get("variables").(map[string]interface{})

	err = session.EVGManager().SetEVG(name, variables)
	return
}

func resourceEvgDelete(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	err = session.EVGManager().SetEVG(d.Get("name").(string), map[string]interface{}{})
	return nil
}
