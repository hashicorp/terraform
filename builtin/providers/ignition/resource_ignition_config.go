package ignition

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/coreos/ignition/config/types"
)

var ignitionResource = &schema.Resource{
	Create: resourceIgnitionFileCreate,
	Delete: resourceIgnitionFileDelete,
	Exists: resourceIgnitionFileExists,
	Read:   resourceIgnitionFileRead,
	Schema: map[string]*schema.Schema{
		"config": &schema.Schema{
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 1,
			ForceNew: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"replace": &schema.Schema{
						Type:     schema.TypeList,
						Optional: true,
						MaxItems: 1,
						Elem:     configReferenceResource,
					},
					"append": &schema.Schema{
						Type:     schema.TypeList,
						Optional: true,
						Elem:     configReferenceResource,
					},
				},
			},
		},
	},
}
var configReferenceResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"source": &schema.Schema{
			Type:        schema.TypeString,
			Required:    true,
			Description: "The URL of the config. Supported schemes are http. Note: When using http, it is advisable to use the verification option to ensure the contents haven’t been modified.",
		},
		"verification": &schema.Schema{
			Type:        schema.TypeString,
			Required:    true,
			Description: "The hash of the config (SHA512)",
		},
	},
}

func resourceConfig() *schema.Resource {
	return &schema.Resource{
		Create: resourceIgnitionFileCreate,
		Delete: resourceIgnitionFileDelete,
		Exists: resourceIgnitionFileExists,
		Read:   resourceIgnitionFileRead,
		Schema: map[string]*schema.Schema{
			"ignition": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				ForceNew: true,
				Elem:     ignitionResource,
			},
			"disks": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"arrays": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"users": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"groups": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"rendered": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceIgnitionFileCreate(d *schema.ResourceData, meta interface{}) error {
	rendered, err := renderConfig(d, meta.(*cache))
	if err != nil {
		return err
	}

	if err := d.Set("rendered", rendered); err != nil {
		return err
	}

	d.SetId(hash(rendered))
	return nil
}

func resourceIgnitionFileDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

func resourceIgnitionFileExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	rendered, err := renderConfig(d, meta.(*cache))
	if err != nil {
		return false, err
	}

	return hash(rendered) == d.Id(), nil
}

func resourceIgnitionFileRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func renderConfig(d *schema.ResourceData, c *cache) (string, error) {
	i, err := buildConfig(d, c)
	if err != nil {
		return "", err
	}

	bytes, err := json.MarshalIndent(i, "  ", "  ")

	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func buildConfig(d *schema.ResourceData, c *cache) (*types.Config, error) {
	var err error
	config := &types.Config{}
	config.Ignition, err = buildIgnition(d)
	if err != nil {
		return nil, err
	}

	config.Passwd, err = buildPasswd(d, c)
	if err != nil {
		return nil, err
	}

	config.Storage, err = buildStorage(d, c)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func buildIgnition(d *schema.ResourceData) (types.Ignition, error) {
	var err error

	i := types.Ignition{}
	i.Version.UnmarshalJSON([]byte(`"2.0.0"`))

	rr := d.Get("ignition.0.config.0.replace.0").(map[string]interface{})
	if len(rr) != 0 {
		i.Config.Replace, err = buildConfigReference(rr)
		if err != nil {
			return i, err
		}
	}

	ar := d.Get("ignition.0.config.0.append").([]interface{})
	if len(ar) != 0 {
		for _, rr := range ar {
			r, err := buildConfigReference(rr.(map[string]interface{}))
			if err != nil {
				return i, err
			}

			i.Config.Append = append(i.Config.Append, *r)
		}
	}

	return i, nil
}

func buildConfigReference(raw map[string]interface{}) (*types.ConfigReference, error) {
	r := &types.ConfigReference{}

	src, err := buildURL(raw["source"].(string))
	if err != nil {
		return nil, err
	}

	r.Source = src

	hash, err := buildHash(raw["verification"].(string))
	if err != nil {
		return nil, err
	}

	r.Verification.Hash = &hash

	return r, nil
}

func buildPasswd(d *schema.ResourceData, c *cache) (types.Passwd, error) {
	passwd := types.Passwd{}

	for _, id := range d.Get("users").([]interface{}) {
		u, ok := c.users[id.(string)]
		if !ok {
			return passwd, fmt.Errorf("invalid user %q, unknown user id", id)
		}

		passwd.Users = append(passwd.Users, *u)
	}

	for _, id := range d.Get("groups").([]interface{}) {
		g, ok := c.groups[id.(string)]
		if !ok {
			return passwd, fmt.Errorf("invalid group %q, unknown group id", id)
		}

		passwd.Groups = append(passwd.Groups, *g)
	}

	return passwd, nil

}

func buildStorage(d *schema.ResourceData, c *cache) (types.Storage, error) {
	storage := types.Storage{}

	for _, id := range d.Get("disks").([]interface{}) {
		d, ok := c.disks[id.(string)]
		if !ok {
			return storage, fmt.Errorf("invalid disk %q, unknown disk id", id)
		}

		storage.Disks = append(storage.Disks, *d)
	}

	for _, id := range d.Get("arrays").([]interface{}) {
		d, ok := c.arrays[id.(string)]
		if !ok {
			return storage, fmt.Errorf("invalid raid %q, unknown raid id", id)
		}

		storage.Arrays = append(storage.Arrays, *d)
	}

	return storage, nil

}

func buildURL(raw string) (types.Url, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return types.Url{}, err
	}

	return types.Url(*u), nil
}

func buildHash(raw string) (types.Hash, error) {
	h := types.Hash{}
	err := h.UnmarshalJSON([]byte(fmt.Sprintf("%q", raw)))

	return h, err
}
