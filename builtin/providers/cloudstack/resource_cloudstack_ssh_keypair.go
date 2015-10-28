package cloudstack

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/go-homedir"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackSSHKeyPair() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackSSHKeyPairCreate,
		Read:   resourceCloudStackSSHKeyPairRead,
		Delete: resourceCloudStackSSHKeyPairDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"public_key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"private_key": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCloudStackSSHKeyPairCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	name := d.Get("name").(string)
	publicKey := d.Get("public_key").(string)

	if publicKey != "" {
		// Register supplied key
		keyPath, err := homedir.Expand(publicKey)
		if err != nil {
			return fmt.Errorf("Error expanding the public key path: %v", err)
		}

		key, err := ioutil.ReadFile(keyPath)
		if err != nil {
			return fmt.Errorf("Error reading the public key: %v", err)
		}

		p := cs.SSH.NewRegisterSSHKeyPairParams(name, string(key))
		_, err = cs.SSH.RegisterSSHKeyPair(p)
		if err != nil {
			return err
		}
	} else {
		// No key supplied, must create one and return the private key
		p := cs.SSH.NewCreateSSHKeyPairParams(name)
		r, err := cs.SSH.CreateSSHKeyPair(p)
		if err != nil {
			return err
		}
		d.Set("private_key", r.Privatekey)
	}

	log.Printf("[DEBUG] Key pair successfully generated at Cloudstack")
	d.SetId(name)

	return resourceCloudStackSSHKeyPairRead(d, meta)
}

func resourceCloudStackSSHKeyPairRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	log.Printf("[DEBUG] looking for key pair with name %s", d.Id())

	p := cs.SSH.NewListSSHKeyPairsParams()
	p.SetName(d.Id())

	r, err := cs.SSH.ListSSHKeyPairs(p)
	if err != nil {
		return err
	}
	if r.Count == 0 {
		log.Printf("[DEBUG] Key pair %s does not exist", d.Id())
		d.SetId("")
		return nil
	}

	//SSHKeyPair name is unique in a cloudstack account so dont need to check for multiple
	d.Set("name", r.SSHKeyPairs[0].Name)
	d.Set("fingerprint", r.SSHKeyPairs[0].Fingerprint)

	return nil
}

func resourceCloudStackSSHKeyPairDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.SSH.NewDeleteSSHKeyPairParams(d.Id())

	// Remove the SSH Keypair
	_, err := cs.SSH.DeleteSSHKeyPair(p)
	if err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"A key pair with name '%s' does not exist for account", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error deleting key pair: %s", err)
	}

	return nil
}
