package aws

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
)

func resourceAwsKmsKeyEncrypt() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsKmsKeyEncryptCreate,
		Read:   resourceAwsKmsKeyEncryptRead,
		Delete: resourceAwsKmsKeyEncryptDelete,

		Schema: map[string]*schema.Schema{
			"key_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"plaintext": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				Sensitive: true,
			},
			"ciphertext_blob": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsKmsKeyEncryptRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsKmsKeyEncryptDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsKmsKeyEncryptCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn

	req := &kms.EncryptInput{
		KeyId:     aws.String(d.Get("key_id").(string)),
		Plaintext: []byte(d.Get("plaintext").(string)),
	}

	log.Printf("[DEBUG] KMS encrypt for key: %s", d.Get("key_id").(string))

	out, err := conn.Encrypt(req)
	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%s-encrypted", d.Get("key_id").(string)))
	d.Set("ciphertext_blob", base64.StdEncoding.EncodeToString((out.CiphertextBlob)))
	return nil
}
