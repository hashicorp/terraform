package cloudfoundry

import (
	"fmt"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceUserProvidedService() *schema.Resource {

	return &schema.Resource{

		Create: resourceUserProvidedServiceCreate,
		Read:   resourceUserProvidedServiceRead,
		Update: resourceUserProvidedServiceUpdate,
		Delete: resourceUserProvidedServiceDelete,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"space": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"syslogDrainURL": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"routeServiceURL": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"credentials": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
		},
	}
}

func resourceUserProvidedServiceCreate(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	var (
		id          string
		credentials map[string]interface{}
	)

	name := d.Get("name").(string)
	space := d.Get("space").(string)
	syslogDrainURL := d.Get("syslogDrainURL").(string)
	routeServiceURL := d.Get("routeServiceURL").(string)

	credentials = make(map[string]interface{})
	for k, v := range d.Get("credentials").(map[string]interface{}) {
		credentials[k] = v.(string)
	}

	sm := session.ServiceManager()

	if id, err = sm.CreateUserProvidedService(name, space, credentials, syslogDrainURL, routeServiceURL); err != nil {
		return
	}
	session.Log.DebugMessage("New User Provided Service : %# v", id)

	d.SetId(id)

	return
}

func resourceUserProvidedServiceRead(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}
	session.Log.DebugMessage("Reading User Provided Service : %s", d.Id())

	sm := session.ServiceManager()
	var ups cfapi.CCUserProvidedService

	ups, err = sm.ReadUserProvidedService(d.Id())
	if err != nil {
		return
	}

	d.Set("name", ups.Name)
	d.Set("space", ups.SpaceGUID)
	d.Set("syslogDrainURL", ups.SyslogDrainURL)
	d.Set("routeServiceURL", ups.RouteServiceURL)
	d.Set("credentials", ups.Credentials)

	session.Log.DebugMessage("Read User Provided Service : %# v", ups)

	return
}

func resourceUserProvidedServiceUpdate(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}
	sm := session.ServiceManager()

	session.Log.DebugMessage("Updating User Provided service %s ", d.Id())

	var (
		credentials map[string]interface{}
	)

	id := d.Id()
	name := d.Get("name").(string)
	syslogDrainURL := d.Get("syslogDrainURL").(string)
	routeServiceURL := d.Get("routeServiceURL").(string)
	credentials = make(map[string]interface{})
	for k, v := range d.Get("credentials").(map[string]interface{}) {
		credentials[k] = v.(string)
	}

	if _, err = sm.UpdateUserProvidedService(id, name, credentials, syslogDrainURL, routeServiceURL); err != nil {
		return
	}
	if err != nil {
		return
	}

	return
}

func resourceUserProvidedServiceDelete(d *schema.ResourceData, meta interface{}) (err error) {

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
