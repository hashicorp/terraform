package cloudfoundry

import "github.com/hashicorp/terraform/helper/schema"

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
				Required: false,
			},
			"tags": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      resourceStringHash,
			},
		},
	}
}

func resourceServiceInstanceCreate(d *schema.ResourceData, meta interface{}) (err error) {
	/*
		session := meta.(*cfapi.Session)
		if session == nil {
			return fmt.Errorf("client is nil")
		}

		name := d.Get("name").(string)
		servicePlan := d.Get("servicePlan").(string)
		space := d.Get("space").(string)
		jsonParameters := d.Get("jsonParameters").(string)
		tags := d.Get("tags").(string)

		sm := session.ServiceManager()

		var (
			serviceInstance cfapi.CCServiceInstance
		)

		if serviceInstance, err = sm.CreateServiceInstance(name, servicePlan, space); err != nil {
			return
		}

		d.SetId(serviceInstance.ID)
	*/
	return
}

func resourceServiceInstanceRead(d *schema.ResourceData, meta interface{}) (err error) {
	/*
		session := meta.(*cfapi.Session)
		if session == nil {
			return fmt.Errorf("client is nil")
		}
		sm := session.ServiceManager()

		var serviceInstance cfapi.CCServiceInstance
	*/
	return
}

func resourceServiceInstanceUpdate(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func resourceServiceInstanceDelete(d *schema.ResourceData, meta interface{}) error {

	return nil
}
