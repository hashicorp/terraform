package gpg

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceGPG() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGPGRead,

		Schema: map[string]*schema.Schema{
			"encrypted_data": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "GPG encrypted data",
			},
			"key_directory": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Path to GPG key directory",
			},
			"decrypted_data": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "decrypted data",
			},
		},
	}
}

func dataSourceGPGRead(d *schema.ResourceData, meta interface{}) error {
	rendered, err := decryptData(d)
	if err != nil {
		return err
	}
	d.Set("decrypted_data", rendered)
	d.SetId(hash(rendered))
	return nil
}

type templateRenderError error

func decryptData(d *schema.ResourceData) (string, error) {
	var keyRingPath string
	if v, ok := d.GetOk("key_directory"); ok {
		keyRingPath = filepath.Join(v.(string), "secring.gpg")
	} else {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		keyRingPath = filepath.Join(usr.HomeDir, ".gnupg/secring.gpg")
	}

	keyringFileBuffer, err := os.Open(keyRingPath)
	if err != nil {
		return "", err
	}
	defer keyringFileBuffer.Close()

	keyring, err := openpgp.ReadKeyRing(keyringFileBuffer)
	if err != nil {
		return "", err
	}

	r, err := armor.Decode(bytes.NewBufferString(d.Get("encrypted_data").(string)))
	if err != nil {
		return "", err
	}

	md, err := openpgp.ReadMessage(r.Body, keyring, nil, nil)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(md.UnverifiedBody)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func hash(s string) string {
	sha := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sha[:])
}
