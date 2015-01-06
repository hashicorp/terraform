package aws

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsKeyPair() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsKeyPairCreate,
		Read:   resourceAwsKeyPairRead,
		Update: nil,
		Delete: resourceAwsKeyPairDelete,

		Schema: map[string]*schema.Schema{
			"key_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"public_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsKeyPairCreate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	keyName := d.Get("key_name").(string)
	publicKey := d.Get("public_key").(string)
	resp, err := ec2conn.ImportKeyPair(keyName, publicKey)
	if err != nil {
		return fmt.Errorf("Error import KeyPair: %s", err)
	}

	d.SetId(resp.KeyName)

	return nil
}

func resourceAwsKeyPairRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	resp, err := ec2conn.KeyPairs([]string{d.Id()}, nil)
	if err != nil {
		return fmt.Errorf("Error retrieving KeyPair: %s", err)
	}

	for _, keyPair := range resp.Keys {
		if keyPair.Name == d.Id() {
			d.Set("key_name", keyPair.Name)
			d.Set("fingerprint", keyPair.Fingerprint)
			return nil
		}
	}

	return fmt.Errorf("Unable to find key pair within: %#v", resp.Keys)
}

func resourceAwsKeyPairDelete(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	_, err := ec2conn.DeleteKeyPair(d.Id())
	return err
}
