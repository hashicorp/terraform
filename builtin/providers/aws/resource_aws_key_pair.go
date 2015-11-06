package aws

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func resourceAwsKeyPair() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsKeyPairCreate,
		Read:   resourceAwsKeyPairRead,
		Update: nil,
		Delete: resourceAwsKeyPairDelete,

		SchemaVersion: 1,
		MigrateState:  resourceAwsKeyPairMigrateState,

		Schema: map[string]*schema.Schema{
			"key_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"public_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					switch v.(type) {
					case string:
						return strings.TrimSpace(v.(string))
					default:
						return ""
					}
				},
			},
			"fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsKeyPairCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	keyName := d.Get("key_name").(string)
	if keyName == "" {
		keyName = resource.UniqueId()
	}

	publicKey := d.Get("public_key").(string)
	req := &ec2.ImportKeyPairInput{
		KeyName:           aws.String(keyName),
		PublicKeyMaterial: []byte(publicKey),
	}
	resp, err := conn.ImportKeyPair(req)
	if err != nil {
		return fmt.Errorf("Error import KeyPair: %s", err)
	}

	d.SetId(*resp.KeyName)
	return nil
}

func resourceAwsKeyPairRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	req := &ec2.DescribeKeyPairsInput{
		KeyNames: []*string{aws.String(d.Id())},
	}
	resp, err := conn.DescribeKeyPairs(req)
	if err != nil {
		awsErr, ok := err.(awserr.Error)
		if ok && awsErr.Code() == "InvalidKeyPair.NotFound" {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving KeyPair: %s", err)
	}

	for _, keyPair := range resp.KeyPairs {
		if *keyPair.KeyName == d.Id() {
			d.Set("key_name", keyPair.KeyName)
			d.Set("fingerprint", keyPair.KeyFingerprint)
			return nil
		}
	}

	return fmt.Errorf("Unable to find key pair within: %#v", resp.KeyPairs)
}

func resourceAwsKeyPairDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	_, err := conn.DeleteKeyPair(&ec2.DeleteKeyPairInput{
		KeyName: aws.String(d.Id()),
	})
	return err
}
