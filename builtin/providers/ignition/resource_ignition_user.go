package ignition

import (
	"github.com/coreos/ignition/config/types"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceUser() *schema.Resource {
	return &schema.Resource{
		Create: resourceUserCreate,
		Delete: resourceUserDelete,
		Exists: resourceUserExists,
		Read:   resourceUserRead,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"password_hash": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceUserCreate(d *schema.ResourceData, meta interface{}) error {
	id, err := buildUser(d, meta.(*cache))
	if err != nil {
		return err
	}

	d.SetId(id)
	d.Set("id", id)
	return nil
}

func resourceUserDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

func resourceUserExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	id, err := buildUser(d, meta.(*cache))
	if err != nil {
		return false, err
	}

	return id == d.Id(), nil
}

func resourceUserRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func buildUser(d *schema.ResourceData, c *cache) (string, error) {
	u := &types.User{}
	u.Name = d.Get("name").(string)
	u.PasswordHash = d.Get("password_hash").(string)

	return c.addUser(u), nil
}
