package nsx

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sky-uk/gonsx"
	"github.com/sky-uk/gonsx/api/service"
	"log"
)

func getSingleService(scopeid, name string, nsxclient *gonsx.NSXClient) (*service.ApplicationService, error) {
	getAllAPI := service.NewGetAll(scopeid)
	err := nsxclient.Do(getAllAPI)

	if err != nil {
		return nil, err
	}

	if getAllAPI.StatusCode() != 200 {
		return nil, fmt.Errorf("Status code: %d, Response: %s", getAllAPI.StatusCode(), getAllAPI.ResponseObject())
	}

	service := getAllAPI.GetResponse().FilterByName(name)

	if service.ObjectID == "" {
		return nil, fmt.Errorf("Not found %s", name)
	}

	return service, nil
}

func resourceService() *schema.Resource {
	return &schema.Resource{
		Create: resourceServiceCreate,
		Read:   resourceServiceRead,
		Delete: resourceServiceDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"scopeid": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"desc": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"proto": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ports": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceServiceCreate(d *schema.ResourceData, m interface{}) error {
	nsxclient := m.(*gonsx.NSXClient)
	var name, scopeid, desc, proto, ports string

	// Gather the attributes for the resource.
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else {
		return fmt.Errorf("name argument is required")
	}

	if v, ok := d.GetOk("scopeid"); ok {
		scopeid = v.(string)
	} else {
		return fmt.Errorf("scopeid argument is required")
	}

	if v, ok := d.GetOk("desc"); ok {
		desc = v.(string)
	} else {
		return fmt.Errorf("desc argument is required")
	}

	if v, ok := d.GetOk("proto"); ok {
		proto = v.(string)
	} else {
		return fmt.Errorf("proto argument is required")
	}

	if v, ok := d.GetOk("ports"); ok {
		ports = v.(string)
	} else {
		return fmt.Errorf("ports argument is required")
	}

	// Create the API, use it and check for errors.
	log.Printf(fmt.Sprintf("[DEBUG] service.NewCreate(%s, %s, %s, %s, %s)", scopeid, name, desc, proto, ports))
	createAPI := service.NewCreate(scopeid, name, desc, proto, ports)
	err := nsxclient.Do(createAPI)

	if err != nil {
		return fmt.Errorf("Error: %v", err)
	}

	if createAPI.StatusCode() != 201 {
		return fmt.Errorf("%s", createAPI.ResponseObject())
	}

	// If we get here, everything is OK.  Set the ID for the Terraform state
	// and return the response from the READ method.
	d.SetId(createAPI.GetResponse())
	return resourceServiceRead(d, m)
}

func resourceServiceRead(d *schema.ResourceData, m interface{}) error {
	nsxclient := m.(*gonsx.NSXClient)
	var scopeid, name string

	// Gather the attributes for the resource.
	if v, ok := d.GetOk("scopeid"); ok {
		scopeid = v.(string)
	} else {
		return fmt.Errorf("scopeid argument is required")
	}

	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else {
		return fmt.Errorf("name argument is required")
	}

	// Gather all the resources that are associated with the specified
	// scopeid.
	log.Printf(fmt.Sprintf("[DEBUG] service.NewGetAll(%s)", scopeid))
	api := service.NewGetAll(scopeid)
	err := nsxclient.Do(api)

	if err != nil {
		return err
	}

	// See if we can find our specifically named resource within the list of
	// resources associated with the scopeid.
	log.Printf(fmt.Sprintf("[DEBUG] api.GetResponse().FilterByName(\"%s\").ObjectID", name))
	serviceObject, err := getSingleService(scopeid, name, nsxclient)
	id := serviceObject.ObjectID
	log.Printf(fmt.Sprintf("[DEBUG] id := %s", id))

	// If the resource has been removed manually, notify Terraform of this fact.
	if id == "" {
		d.SetId("")
	}

	return nil
}

func resourceServiceDelete(d *schema.ResourceData, m interface{}) error {
	nsxclient := m.(*gonsx.NSXClient)
	var name, scopeid string

	// Gather the attributes for the resource.
	if v, ok := d.GetOk("scopeid"); ok {
		scopeid = v.(string)
	} else {
		return fmt.Errorf("scopeid argument is required")
	}

	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else {
		return fmt.Errorf("name argument is required")
	}

	// Gather all the resources that are associated with the specified
	// scopeid.
	log.Printf(fmt.Sprintf("[DEBUG] service.NewGetAll(%s)", scopeid))
	api := service.NewGetAll(scopeid)
	err := nsxclient.Do(api)

	if err != nil {
		return err
	}

	// See if we can find our specifically named resource within the list of
	// resources associated with the scopeid.
	log.Printf(fmt.Sprintf("[DEBUG] api.GetResponse().FilterByName(\"%s\").ObjectID", name))
	serviceObject, err := getSingleService(scopeid, name, nsxclient)
	id := serviceObject.ObjectID
	log.Printf(fmt.Sprintf("[DEBUG] id := %s", id))

	// If the resource has been removed manually, notify Terraform of this fact.
	if id == "" {
		d.SetId("")
		return nil
	}

	// If we got here, the resource exists, so we attempt to delete it.
	deleteAPI := service.NewDelete(id)
	err = nsxclient.Do(deleteAPI)

	if err != nil {
		return err
	}

	// If we got here, the resource had existed, we deleted it and there was
	// no error.  Notify Terraform of this fact and return successful
	// completion.
	d.SetId("")
	log.Printf(fmt.Sprintf("[DEBUG] id %s deleted.", id))

	return nil
}
