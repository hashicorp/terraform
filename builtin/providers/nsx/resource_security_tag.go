package nsx

import (
	"errors"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sky-uk/gonsx"
	"github.com/sky-uk/gonsx/api/securitytag"
	"log"
)

func getSingleSecurityTag(name string, nsxclient *gonsx.NSXClient) (*securitytag.SecurityTag, error) {
	getAllAPI := securitytag.NewGetAll()
	err := nsxclient.Do(getAllAPI)

	if err != nil {
		return nil, err
	}

	if getAllAPI.StatusCode() != 200 {
		return nil, fmt.Errorf("Status code: %d, Response: %s", getAllAPI.StatusCode(), getAllAPI.ResponseObject())
	}

	securitytag := getAllAPI.GetResponse().FilterByName(name)

	if securitytag.ObjectID == "" {
		return nil, fmt.Errorf("Not found %s", name)
	}

	return securitytag, nil
}

func resourceSecurityTag() *schema.Resource {
	return &schema.Resource{
		Create: resourceSecurityTagCreate,
		Read:   resourceSecurityTagRead,
		Delete: resourceSecurityTagDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"desc": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceSecurityTagCreate(d *schema.ResourceData, m interface{}) error {
	nsxclient := m.(*gonsx.NSXClient)
	var name, desc string //, singleoperation string

	// Gather the attributes for the resource.
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else {
		return fmt.Errorf("name argument is required")
	}

	if v, ok := d.GetOk("desc"); ok {
		desc = v.(string)
	} else {
		return fmt.Errorf("desc argument is required")
	}

	log.Printf(fmt.Sprintf("[DEBUG] securitytag.NewCreate(%s, %s)", name, desc))
	createAPI := securitytag.NewCreate(name, desc)
	nsxclient.Do(createAPI)

	if createAPI.StatusCode() != 201 {
		return fmt.Errorf("Failed to create security tag %s", name)
	}

	// If we get to here creation was successful. Set the ID for the Terraform state file
	id := createAPI.GetResponse()

	if id != "" {
		d.SetId(id)
	} else {
		return errors.New("Can not establish the id of the created resource")
	}

	return resourceSecurityTagRead(d, m)
}

func resourceSecurityTagRead(d *schema.ResourceData, m interface{}) error {
	nsxclient := m.(*gonsx.NSXClient)
	var name string

	// Gather the attributes for the resource.
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else {
		return fmt.Errorf("name argument is required")
	}

	// Gather all the resources that are associated with the specified
	// edgeid.
	log.Printf(fmt.Sprintf("[DEBUG] securitytag.NewGetAll()"))
	api := securitytag.NewGetAll()
	err := nsxclient.Do(api)

	if err != nil {
		return err
	}

	// See if we can find our specifically named resource within the list of
	// resources associated with the edgeid.
	log.Printf(fmt.Sprintf("[DEBUG] api.GetResponse().FilterByName(\"%s\").ObjectID", name))
	securityTagObject, err := getSingleSecurityTag(name, nsxclient)

	if err != nil {
		return err
	}

	id := securityTagObject.ObjectID
	log.Printf(fmt.Sprintf("[DEBUG] id := %s", id))

	// If the resource has been removed manually, notify Terraform of this fact.
	if id == "" {
		d.SetId("")
	}

	return nil
}

func resourceSecurityTagDelete(d *schema.ResourceData, m interface{}) error {
	nsxclient := m.(*gonsx.NSXClient)
	var name string //, singleoperation string

	// Gather the attributes for the resource.
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else {
		return fmt.Errorf("name argument is required")
	}

	// Gather all the resources that are associated with the specified
	// edgeid.
	log.Printf(fmt.Sprintf("[DEBUG] securitytag.NewGetAll()"))
	api := securitytag.NewGetAll()
	err := nsxclient.Do(api)

	if err != nil {
		return err
	}

	// See if we can find our specifically named resource within the list of
	// resources associated with the edgeid.
	log.Printf(fmt.Sprintf("[DEBUG] api.GetResponse().FilterByName(\"%s\").ObjectID", name))
	securityTagObject, err := getSingleSecurityTag(name, nsxclient)

	if err != nil {
		return err
	}

	id := securityTagObject.ObjectID
	log.Printf(fmt.Sprintf("[DEBUG] security tag id := %s", id))

	// If we got here, the resource exists, so we attempt to delete it.
	deleteAPI := securitytag.NewDelete(id)
	err = nsxclient.Do(deleteAPI)

	if err != nil {
		return err
	}

	if deleteAPI.StatusCode() != 200 {
		//log.Printf(fmt.Sprintf("[DEBUG] id %s deleted.", id))
		return fmt.Errorf("[DEBUG] response object: %s", deleteAPI.ResponseObject())
	}

	// If we got here, the resource had existed, we deleted it and there was
	// no error.  Notify Terraform of this fact and return successful
	// completion.
	d.SetId("")
	log.Printf(fmt.Sprintf("[DEBUG] id %s deleted.", id))
	return nil
}
