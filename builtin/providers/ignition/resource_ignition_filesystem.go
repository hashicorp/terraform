package ignition

import (
	"fmt"

	"github.com/coreos/ignition/config/types"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceFilesystem() *schema.Resource {
	return &schema.Resource{
		Create: resourceFilesystemCreate,
		Delete: resourceFilesystemDelete,
		Exists: resourceFilesystemExists,
		Read:   resourceFilesystemRead,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"mount": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"format": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"force": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"options": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceFilesystemCreate(d *schema.ResourceData, meta interface{}) error {
	id, err := buildFilesystem(d, meta.(*cache))
	if err != nil {
		return err
	}

	d.SetId(id)
	return nil
}

func resourceFilesystemDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

func resourceFilesystemExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	id, err := buildFilesystem(d, meta.(*cache))
	if err != nil {
		return false, err
	}

	return id == d.Id(), nil
}

func resourceFilesystemRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func buildFilesystem(d *schema.ResourceData, c *cache) (string, error) {
	var mount *types.FilesystemMount
	if _, ok := d.GetOk("mount"); ok {
		mount = &types.FilesystemMount{
			Device: types.Path(d.Get("mount.0.device").(string)),
			Format: types.FilesystemFormat(d.Get("mount.0.format").(string)),
		}

		force, hasForce := d.GetOk("mount.0.force")
		options, hasOptions := d.GetOk("mount.0.options")
		if hasOptions || hasForce {
			mount.Create = &types.FilesystemCreate{
				Force:   force.(bool),
				Options: castSliceInterface(options.([]interface{})),
			}
		}
	}

	var path *types.Path
	if p, ok := d.GetOk("path"); ok {
		tp := types.Path(p.(string))
		path = &tp
	}

	if mount != nil && path != nil {
		return "", fmt.Errorf("mount and path are mutually exclusive")
	}

	return c.addFilesystem(&types.Filesystem{
		Name:  d.Get("name").(string),
		Mount: mount,
		Path:  path,
	}), nil
}
