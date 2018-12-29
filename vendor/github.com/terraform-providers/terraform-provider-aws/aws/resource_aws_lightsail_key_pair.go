package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/hashicorp/terraform/helper/encryption"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLightsailKeyPair() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLightsailKeyPairCreate,
		Read:   resourceAwsLightsailKeyPairRead,
		Delete: resourceAwsLightsailKeyPairDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
			},
			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
			},

			// optional fields
			"pgp_key": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			// additional info returned from the API
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// fields returned from CreateKey
			"fingerprint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"public_key": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},
			"private_key": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// encrypted fields if pgp_key is given
			"encrypted_fingerprint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"encrypted_private_key": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsLightsailKeyPairCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lightsailconn

	var kName string
	if v, ok := d.GetOk("name"); ok {
		kName = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		kName = resource.PrefixedUniqueId(v.(string))
	} else {
		kName = resource.UniqueId()
	}

	var pubKey string
	var op *lightsail.Operation
	if pubKeyInterface, ok := d.GetOk("public_key"); ok {
		pubKey = pubKeyInterface.(string)
	}

	if pubKey == "" {
		// creating new key
		resp, err := conn.CreateKeyPair(&lightsail.CreateKeyPairInput{
			KeyPairName: aws.String(kName),
		})
		if err != nil {
			return err
		}
		if resp.Operation == nil {
			return fmt.Errorf("No operation found for CreateKeyPair response")
		}
		if resp.KeyPair == nil {
			return fmt.Errorf("No KeyPair information found for CreateKeyPair response")
		}
		d.SetId(kName)

		// private_key and public_key are only available in the response from
		// CreateKey pair. Here we set the public_key, and encrypt the private_key
		// if a pgp_key is given, else we store the private_key in state
		d.Set("public_key", resp.PublicKeyBase64)

		// encrypt private key if pgp_key is given
		pgpKey, err := encryption.RetrieveGPGKey(d.Get("pgp_key").(string))
		if err != nil {
			return err
		}
		if pgpKey != "" {
			fingerprint, encrypted, err := encryption.EncryptValue(pgpKey, *resp.PrivateKeyBase64, "Lightsail Private Key")
			if err != nil {
				return err
			}

			d.Set("encrypted_fingerprint", fingerprint)
			d.Set("encrypted_private_key", encrypted)
		} else {
			d.Set("private_key", resp.PrivateKeyBase64)
		}

		op = resp.Operation
	} else {
		// importing key
		resp, err := conn.ImportKeyPair(&lightsail.ImportKeyPairInput{
			KeyPairName:     aws.String(kName),
			PublicKeyBase64: aws.String(pubKey),
		})

		if err != nil {
			log.Printf("[ERR] Error importing key: %s", err)
			return err
		}
		d.SetId(kName)

		op = resp.Operation
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Started"},
		Target:     []string{"Completed", "Succeeded"},
		Refresh:    resourceAwsLightsailOperationRefreshFunc(op.Id, meta),
		Timeout:    10 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		// We don't return an error here because the Create call succeeded
		log.Printf("[ERR] Error waiting for KeyPair (%s) to become ready: %s", d.Id(), err)
	}

	return resourceAwsLightsailKeyPairRead(d, meta)
}

func resourceAwsLightsailKeyPairRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lightsailconn

	resp, err := conn.GetKeyPair(&lightsail.GetKeyPairInput{
		KeyPairName: aws.String(d.Id()),
	})

	if err != nil {
		log.Printf("[WARN] Error getting KeyPair (%s): %s", d.Id(), err)
		// check for known not found error
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NotFoundException" {
				log.Printf("[WARN] Lightsail KeyPair (%s) not found, removing from state", d.Id())
				d.SetId("")
				return nil
			}
		}
		return err
	}

	d.Set("arn", resp.KeyPair.Arn)
	d.Set("name", resp.KeyPair.Name)
	d.Set("fingerprint", resp.KeyPair.Fingerprint)

	return nil
}

func resourceAwsLightsailKeyPairDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lightsailconn
	resp, err := conn.DeleteKeyPair(&lightsail.DeleteKeyPairInput{
		KeyPairName: aws.String(d.Id()),
	})

	if err != nil {
		return err
	}

	op := resp.Operation
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Started"},
		Target:     []string{"Completed", "Succeeded"},
		Refresh:    resourceAwsLightsailOperationRefreshFunc(op.Id, meta),
		Timeout:    10 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for KeyPair (%s) to become destroyed: %s",
			d.Id(), err)
	}

	return nil
}
