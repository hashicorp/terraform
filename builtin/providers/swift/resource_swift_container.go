package swift

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/ncw/swift"
	"strings"
)

func resourceSwiftContainer() *schema.Resource {
	return &schema.Resource{
		Create: resourceSwiftContainerCreate,
		Read:   resourceSwiftContainerRead,
		Update: resourceSwiftContainerUpdate,
		Delete: resourceSwiftContainerDelete,
		Exists: resourceSwiftContainerExists,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"read_access": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"write_access": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

var aclHeaderMap = map[string]string{
	"X-Container-Read":  "read_access",
	"X-Container-Write": "write_access",
}

func resourceSwiftContainerCreate(d *schema.ResourceData, meta interface{}) error {
	return containerCreateOrUpdate(true, d, meta)
}

func resourceSwiftContainerRead(d *schema.ResourceData, meta interface{}) error {
	c := obtainConnection(meta)

	name := d.Get("name").(string)
	_, headers, err := c.Container(name)
	if err != nil {
		return fmt.Errorf("swift container resource read: Could not get container %s", name)
	}

	// Read in acls into schema
	for headerName, aclType := range aclHeaderMap {
		if headers[headerName] != "" {
			usernames := headers[headerName]
			if usernames != "" {
				d.Set(aclType, strings.Split(usernames, ","))
			}
		}
	}

	return nil
}

func resourceSwiftContainerUpdate(d *schema.ResourceData, meta interface{}) error {
	return containerCreateOrUpdate(false, d, meta)
}

func resourceSwiftContainerDelete(d *schema.ResourceData, meta interface{}) error {
	c := obtainConnection(meta)

	name := d.Get("name").(string)

	return c.ContainerDelete(name)
}

func resourceSwiftContainerExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	err := resourceSwiftContainerRead(d, meta)

	return err == nil, err
}

// Utility functions

func containerCreateOrUpdate(create bool, d *schema.ResourceData, meta interface{}) error {
	c := obtainConnection(meta)
	action := "creation"
	if create == false {
		action = "update"
	}

	headers := swift.Headers{}
	name := d.Get("name").(string)

	// Get acls into swift headers
	for headerName, aclType := range aclHeaderMap {
		aclList := d.Get(aclType).([]interface{})
		if len(aclList) > 0 {
			usernames := []string{}
			for _, username := range aclList {
				usernames = append(usernames, username.(string))
			}
			headers[headerName] = strings.Join(usernames, ",")
		}
	}

	var err error
	if create {
		err = c.ContainerCreate(name, headers)
	} else {
		err = c.ContainerUpdate(name, headers)
	}

	if err != nil {
		return fmt.Errorf(
			"swift container resource %s: The container %s %s failed",
			action, action,
			name,
		)
	}

	if create {
		d.SetId(name)
	}

	return nil
}
