package triton

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/joyent/triton-go"
)

func resourceKey() *schema.Resource {
	return &schema.Resource{
		Create:   resourceKeyCreate,
		Exists:   resourceKeyExists,
		Read:     resourceKeyRead,
		Delete:   resourceKeyDelete,
		Timeouts: fastResourceTimeout,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Name of the key (generated from the key comment if not set)",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
			"key": {
				Description: "Content of public key from disk in OpenSSH format",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourceKeyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*triton.Client)

	if keyName := d.Get("name").(string); keyName == "" {
		parts := strings.SplitN(d.Get("key").(string), " ", 3)
		if len(parts) == 3 {
			d.Set("name", parts[2])
		} else {
			return errors.New("No key name specified, and key material has no comment")
		}
	}

	_, err := client.Keys().CreateKey(context.Background(), &triton.CreateKeyInput{
		Name: d.Get("name").(string),
		Key:  d.Get("key").(string),
	})
	if err != nil {
		return err
	}

	d.SetId(d.Get("name").(string))

	return resourceKeyRead(d, meta)
}

func resourceKeyExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*triton.Client)

	_, err := client.Keys().GetKey(context.Background(), &triton.GetKeyInput{
		KeyName: d.Id(),
	})
	if err != nil {
		return false, err
	}

	return true, nil
}

func resourceKeyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*triton.Client)

	key, err := client.Keys().GetKey(context.Background(), &triton.GetKeyInput{
		KeyName: d.Id(),
	})
	if err != nil {
		return err
	}

	d.Set("name", key.Name)
	d.Set("key", key.Key)

	return nil
}

func resourceKeyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*triton.Client)

	return client.Keys().DeleteKey(context.Background(), &triton.DeleteKeyInput{
		KeyName: d.Id(),
	})
}
