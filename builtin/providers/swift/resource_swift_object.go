package swift

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceSwiftObject() *schema.Resource {
	return &schema.Resource{
		Create: resourceSwiftObjectCreate,
		Read:   resourceSwiftObjectRead,
		Update: resourceSwiftObjectUpdate,
		Delete: resourceSwiftObjectDelete,
		Exists: resourceSwiftObjectExists,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"container_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"contents": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceSwiftObjectCreate(d *schema.ResourceData, meta interface{}) error {
	return objectCreateOrUpdate(true, d, meta)
}

func resourceSwiftObjectRead(d *schema.ResourceData, meta interface{}) error {
	c := obtainConnection(meta)

	name := d.Get("name").(string)
	containerName := d.Get("container_name").(string)

	contents, err := c.ObjectGetString(containerName, name)
	if err != nil {
		return fmt.Errorf(
			"swift object resource read: %s/%s: %s",
			containerName, name, err.Error())
	}

	d.Set("contents", contents)

	return nil
}

func resourceSwiftObjectUpdate(d *schema.ResourceData, meta interface{}) error {
	return objectCreateOrUpdate(false, d, meta)
}

func resourceSwiftObjectDelete(d *schema.ResourceData, meta interface{}) error {
	c := obtainConnection(meta)

	name := d.Get("name").(string)
	containerName := d.Get("container_name").(string)

	return c.ObjectDelete(containerName, name)
}

func resourceSwiftObjectExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	err := resourceSwiftObjectRead(d, meta)

	return err == nil, err
}

// Utility functions

func objectCreateOrUpdate(create bool, d *schema.ResourceData, meta interface{}) error {
	c := obtainConnection(meta)
	action := "creation"
	if create == false {
		action = "update"
	}

	name := d.Get("name").(string)
	containerName := d.Get("container_name").(string)
	contents := d.Get("contents").(string)
	id := fmt.Sprintf("%s/%s", containerName, name)
	data := make([]byte, 100)

	if contents != "" {
		data = bytes.NewBufferString(contents).Bytes()
	}

	file, err := c.ObjectCreate(containerName, name, true, "", "", nil)
	if err != nil {
		return fmt.Errorf(
			"swift object resource %s: Failed to %s object %s: %s",
			action, action, id, err.Error(),
		)
	}

	// Write contents here to file
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf(
			"swift object resource %s: Error writing object %s: %s",
			action, id, err.Error())
	}

	// Close file; check error on close
	err = file.Close()
	if err != nil {
		return fmt.Errorf(
			"swift object resource %s: Error closing write stream: %s",
			action, err.Error())
	}

	if create {
		d.SetId(id)
	}

	return nil
}
