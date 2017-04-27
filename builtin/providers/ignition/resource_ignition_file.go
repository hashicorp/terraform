package ignition

import (
	"encoding/base64"
	"fmt"

	"github.com/coreos/ignition/config/types"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceFile() *schema.Resource {
	return &schema.Resource{
		Exists: resourceFileExists,
		Read:   resourceFileRead,
		Schema: map[string]*schema.Schema{
			"filesystem": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"path": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"content": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mime": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Default:  "text/plain",
						},

						"content": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
			"source": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"compression": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"verification": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
			"mode": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"uid": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"gid": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceFileRead(d *schema.ResourceData, meta interface{}) error {
	id, err := buildFile(d, globalCache)
	if err != nil {
		return err
	}

	d.SetId(id)
	return nil
}

func resourceFileExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	id, err := buildFile(d, globalCache)
	if err != nil {
		return false, err
	}

	return id == d.Id(), nil
}

func buildFile(d *schema.ResourceData, c *cache) (string, error) {
	_, hasContent := d.GetOk("content")
	_, hasSource := d.GetOk("source")
	if hasContent && hasSource {
		return "", fmt.Errorf("content and source options are incompatible")
	}

	if !hasContent && !hasSource {
		return "", fmt.Errorf("content or source options must be present")
	}

	var compression types.Compression
	var source types.Url
	var hash *types.Hash
	var err error

	if hasContent {
		source, err = encodeDataURL(
			d.Get("content.0.mime").(string),
			d.Get("content.0.content").(string),
		)

		if err != nil {
			return "", err
		}
	}

	if hasSource {
		source, err = buildURL(d.Get("source.0.source").(string))
		if err != nil {
			return "", err
		}

		compression = types.Compression(d.Get("source.0.compression").(string))
		h, err := buildHash(d.Get("source.0.verification").(string))
		if err != nil {
			return "", err
		}

		hash = &h
	}

	return c.addFile(&types.File{
		Filesystem: d.Get("filesystem").(string),
		Path:       types.Path(d.Get("path").(string)),
		Contents: types.FileContents{
			Compression: compression,
			Source:      source,
			Verification: types.Verification{
				Hash: hash,
			},
		},
		User: types.FileUser{
			Id: d.Get("uid").(int),
		},
		Group: types.FileGroup{
			Id: d.Get("gid").(int),
		},
		Mode: types.FileMode(d.Get("mode").(int)),
	}), nil
}

func encodeDataURL(mime, content string) (types.Url, error) {
	base64 := base64.StdEncoding.EncodeToString([]byte(content))
	return buildURL(
		fmt.Sprintf("data:%s;charset=utf-8;base64,%s", mime, base64),
	)
}
