package rundeck

import (
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/apparentlymart/go-rundeck-api/rundeck"
)

func resourceRundeckPublicKey() *schema.Resource {
	return &schema.Resource{
		Create: CreatePublicKey,
		Update: UpdatePublicKey,
		Delete: DeletePublicKey,
		Exists: PublicKeyExists,
		Read:   ReadPublicKey,

		Schema: map[string]*schema.Schema{
			"path": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path to the key within the key store",
				ForceNew:    true,
			},

			"key_material": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The public key data to store, in the usual OpenSSH public key file format",
			},

			"url": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL at which the key content can be retrieved",
			},

			"delete": &schema.Schema{
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True if the key should be deleted when the resource is deleted. Defaults to true if key_material is provided in the configuration.",
			},
		},
	}
}

func CreatePublicKey(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.Client)

	path := d.Get("path").(string)
	keyMaterial := d.Get("key_material").(string)

	if keyMaterial != "" {
		err := client.CreatePublicKey(path, keyMaterial)
		if err != nil {
			return err
		}
		d.Set("delete", true)
	}

	d.SetId(path)

	return ReadPublicKey(d, meta)
}

func UpdatePublicKey(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.Client)

	if d.HasChange("key_material") {
		path := d.Get("path").(string)
		keyMaterial := d.Get("key_material").(string)

		err := client.ReplacePublicKey(path, keyMaterial)
		if err != nil {
			return err
		}
	}

	return ReadPublicKey(d, meta)
}

func DeletePublicKey(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.Client)

	path := d.Id()

	// Since this resource can be used both to create and to read existing
	// public keys, we'll only actually delete the key if we remember that
	// we created the key in the first place, or if the user explicitly
	// opted in to have an existing key deleted.
	if d.Get("delete").(bool) {
		// The only "delete" call we have is oblivious to key type, but
		// that's okay since our Exists implementation makes sure that we
		// won't try to delete a key of the wrong type since we'll pretend
		// that it's already been deleted.
		err := client.DeleteKey(path)
		if err != nil {
			return err
		}
	}

	d.SetId("")
	return nil
}

func ReadPublicKey(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.Client)

	path := d.Id()

	key, err := client.GetKeyMeta(path)
	if err != nil {
		return err
	}

	keyMaterial, err := client.GetKeyContent(path)
	if err != nil {
		return err
	}

	d.Set("key_material", keyMaterial)
	d.Set("url", key.URL)

	return nil
}

func PublicKeyExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*rundeck.Client)

	path := d.Id()

	key, err := client.GetKeyMeta(path)
	if err != nil {
		if _, ok := err.(rundeck.NotFoundError); ok {
			err = nil
		}
		return false, err
	}

	if key.KeyType != "public" {
		// If the key type isn't public then as far as this resource is
		// concerned it doesn't exist. (We'll fail properly when we try to
		// create a key where one already exists.)
		return false, nil
	}

	return true, nil
}
