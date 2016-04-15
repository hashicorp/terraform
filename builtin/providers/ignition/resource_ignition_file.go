package ignition

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/coreos/ignition/config/types"
)

var configReference = &schema.Resource{
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

func resourceFile() *schema.Resource {
	return &schema.Resource{
		Create: resourceIgnitionFileCreate,
		Delete: resourceIgnitionFileDelete,
		Exists: resourceIgnitionFileExists,
		Read:   resourceIgnitionFileRead,

		Schema: map[string]*schema.Schema{
			"version": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "2.0.0",
				Description: "The semantic version number of the spec",
				ForceNew:    true,
			},
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
							Elem:     configReference,
						},
						"append": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     configReference,
						},
					},
				},
			},
			"rendered": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "rendered template",
			},
		},
	}
}

func resourceIgnitionFileCreate(d *schema.ResourceData, meta interface{}) error {
	rendered, err := renderIgnition(d)
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
	rendered, err := renderIgnition(d)
	if err != nil {
		return false, err
	}

	return hash(rendered) == d.Id(), nil
}

func resourceIgnitionFileRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func renderIgnition(d *schema.ResourceData) (string, error) {
	i, err := buildIgnition(d)
	if err != nil {
		return "", err
	}

	bytes, err := json.MarshalIndent(i, "  ", "  ")
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func buildIgnition(d *schema.ResourceData) (*types.Ignition, error) {
	var err error

	i := &types.Ignition{}
	i.Version.UnmarshalJSON([]byte(`"2.0.0"`))

	rr := d.Get("config.0.replace.0").(map[string]interface{})
	if len(rr) != 0 {
		i.Config.Replace, err = buildConfigReference(rr)
		if err != nil {
			return nil, err
		}
	}

	ar := d.Get("config.0.append").([]interface{})
	if len(ar) != 0 {
		for _, rr := range ar {
			r, err := buildConfigReference(rr.(map[string]interface{}))
			if err != nil {
				return nil, err
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

func hash(s string) string {
	sha := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sha[:])
}
