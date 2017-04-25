package ignition

import (
	"github.com/coreos/ignition/config/types"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceNetworkdUnit() *schema.Resource {
	return &schema.Resource{
		Exists: resourceNetworkdUnitExists,
		Read:   resourceNetworkdUnitRead,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"content": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceNetworkdUnitRead(d *schema.ResourceData, meta interface{}) error {
	id, err := buildNetworkdUnit(d, globalCache)
	if err != nil {
		return err
	}

	d.SetId(id)
	return nil
}

func resourceNetworkdUnitDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

func resourceNetworkdUnitExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	id, err := buildNetworkdUnit(d, globalCache)
	if err != nil {
		return false, err
	}

	return id == d.Id(), nil
}

func buildNetworkdUnit(d *schema.ResourceData, c *cache) (string, error) {
	if err := validateUnitContent(d.Get("content").(string)); err != nil {
		return "", err
	}

	return c.addNetworkdUnit(&types.NetworkdUnit{
		Name:     types.NetworkdUnitName(d.Get("name").(string)),
		Contents: d.Get("content").(string),
	}), nil
}
