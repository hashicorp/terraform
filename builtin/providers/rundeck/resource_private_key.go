package rundeck

import (
	"crypto/sha1"
	"encoding/hex"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/apparentlymart/go-rundeck-api/rundeck"
)

func resourceRundeckPrivateKey() *schema.Resource {
	return &schema.Resource{
		Create: CreateOrUpdatePrivateKey,
		Update: CreateOrUpdatePrivateKey,
		Delete: DeletePrivateKey,
		Exists: PrivateKeyExists,
		Read:   ReadPrivateKey,

		Schema: map[string]*schema.Schema{
			"path": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path to the key within the key store",
				ForceNew:    true,
			},

			"key_material": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The private key material to store, in PEM format",
				StateFunc: func(v interface{}) string {
					switch v.(type) {
					case string:
						hash := sha1.Sum([]byte(v.(string)))
						return hex.EncodeToString(hash[:])
					default:
						return ""
					}
				},
			},
		},
	}
}

func CreateOrUpdatePrivateKey(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.Client)

	path := d.Get("path").(string)
	keyMaterial := d.Get("key_material").(string)

	var err error

	if d.Id() != "" {
		err = client.ReplacePrivateKey(path, keyMaterial)
	} else {
		err = client.CreatePrivateKey(path, keyMaterial)
	}

	if err != nil {
		return err
	}

	d.SetId(path)

	return ReadPrivateKey(d, meta)
}

func DeletePrivateKey(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.Client)

	path := d.Id()

	// The only "delete" call we have is oblivious to key type, but
	// that's okay since our Exists implementation makes sure that we
	// won't try to delete a key of the wrong type since we'll pretend
	// that it's already been deleted.
	err := client.DeleteKey(path)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func ReadPrivateKey(d *schema.ResourceData, meta interface{}) error {
	// Nothing to read for a private key: existence is all we need to
	// worry about, and PrivateKeyExists took care of that.
	return nil
}

func PrivateKeyExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*rundeck.Client)

	path := d.Id()

	key, err := client.GetKeyMeta(path)
	if err != nil {
		if _, ok := err.(rundeck.NotFoundError); ok {
			err = nil
		}
		return false, err
	}

	if key.KeyType != "private" {
		// If the key type isn't public then as far as this resource is
		// concerned it doesn't exist. (We'll fail properly when we try to
		// create a key where one already exists.)
		return false, nil
	}

	return true, nil
}
