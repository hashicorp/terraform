package cloudfoundry

import (
	"fmt"

	"encoding/json"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceServiceInstance() *schema.Resource {

	return &schema.Resource{

		Create: resourceServiceInstanceCreate,
		Read:   resourceServiceInstanceRead,
		Update: resourceServiceInstanceUpdate,
		Delete: resourceServiceInstanceDelete,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"servicePlan": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"space": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"jsonParameters": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"tags": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceServiceInstanceCreate(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	var (
		id     string
		tags   []string
		params map[string]interface{}
	)
	name := d.Get("name").(string)
	servicePlan := d.Get("servicePlan").(string)
	space := d.Get("space").(string)
	jsonParameters := d.Get("jsonParameters").(string)

	for _, v := range d.Get("tags").([]interface{}) {
		tags = append(tags, v.(string))
	}

	if len(jsonParameters) > 0 {
		if err = json.Unmarshal([]byte(jsonParameters), &params); err != nil {
			return
		}
	}

	sm := session.ServiceManager()

	if id, err = sm.CreateServiceInstance(name, servicePlan, space, params, tags); err != nil {
		return
	}
	session.Log.DebugMessage("New Service Instance : %# v", id)

	// TODO deal with asynchronous responses

	d.SetId(id)

	return
}

func resourceServiceInstanceRead(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}
	session.Log.DebugMessage("Reading Service Instance : %s", d.Id())

	sm := session.ServiceManager()
	var serviceInstance cfapi.CCServiceInstance

	serviceInstance, err = sm.ReadServiceInstance(d.Id())
	if err != nil {
		return
	}

	d.Set("name", serviceInstance.Name)
	d.Set("servicePlan", serviceInstance.ServicePlanGUID)
	d.Set("space", serviceInstance.SpaceGUID)

	if serviceInstance.Tags != nil {
		tags := make([]interface{}, len(serviceInstance.Tags))
		for i, v := range serviceInstance.Tags {
			tags[i] = v
		}
		d.Set("tags", tags)
	} else {
		d.Set("tags", nil)
	}

	session.Log.DebugMessage("Read Service Instance : %# v", serviceInstance)

	return
}

func resourceServiceInstanceUpdate(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}
	sm := session.ServiceManager()

	session.Log.DebugMessage("begin resourceServiceInstanceUpdate")

	var (
		id, name string
		tags     []string
		params   map[string]interface{}
	)

	id = d.Id()
	name = d.Get("name").(string)
	servicePlan := d.Get("servicePlan").(string)
	jsonParameters := d.Get("jsonParameters").(string)

	if len(jsonParameters) > 0 {
		if err = json.Unmarshal([]byte(jsonParameters), &params); err != nil {
			return
		}
	}

	for _, v := range d.Get("tags").([]interface{}) {
		tags = append(tags, v.(string))
	}

	if _, err = sm.UpdateServiceInstance(id, name, servicePlan, params, tags); err != nil {
		return
	}
	if err != nil {
		return
	}

	return
}

func resourceServiceInstanceDelete(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}
	session.Log.DebugMessage("begin resourceServiceInstanceDelete")

	sm := session.ServiceManager()

	err = sm.DeleteServiceInstance(d.Id())
	if err != nil {
		return
	}

	session.Log.DebugMessage("Deleted Service Instance : %s", d.Id())

	return
}
