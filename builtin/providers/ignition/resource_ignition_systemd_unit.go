package ignition

import (
	"github.com/coreos/ignition/config/types"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceSystemdUnit() *schema.Resource {
	return &schema.Resource{
		Exists: resourceSystemdUnitExists,
		Read:   resourceSystemdUnitRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"enable": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},
			"mask": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"content": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"dropin": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"content": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func resourceSystemdUnitRead(d *schema.ResourceData, meta interface{}) error {
	id, err := buildSystemdUnit(d, globalCache)
	if err != nil {
		return err
	}

	d.SetId(id)
	return nil
}

func resourceSystemdUnitExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	id, err := buildSystemdUnit(d, globalCache)
	if err != nil {
		return false, err
	}

	return id == d.Id(), nil
}

func buildSystemdUnit(d *schema.ResourceData, c *cache) (string, error) {
	var dropins []types.SystemdUnitDropIn
	for _, raw := range d.Get("dropin").([]interface{}) {
		value := raw.(map[string]interface{})

		if err := validateUnitContent(value["content"].(string)); err != nil {
			return "", err
		}

		dropins = append(dropins, types.SystemdUnitDropIn{
			Name:     types.SystemdUnitDropInName(value["name"].(string)),
			Contents: value["content"].(string),
		})
	}

	if err := validateUnitContent(d.Get("content").(string)); err != nil {
		if err != errEmptyUnit {
			return "", err
		}
	}

	return c.addSystemdUnit(&types.SystemdUnit{
		Name:     types.SystemdUnitName(d.Get("name").(string)),
		Contents: d.Get("content").(string),
		Enable:   d.Get("enable").(bool),
		Mask:     d.Get("mask").(bool),
		DropIns:  dropins,
	}), nil
}
