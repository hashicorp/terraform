package aws

import (
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsKmsSecret() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsKmsSecretRead,

		Schema: map[string]*schema.Schema{
			"secret": {
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"payload": {
							Type:     schema.TypeString,
							Required: true,
						},
						"context": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"grant_tokens": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"__has_dynamic_attributes": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

// dataSourceAwsKmsSecretRead decrypts the specified secrets
func dataSourceAwsKmsSecretRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn
	secrets := d.Get("secret").(*schema.Set)

	d.SetId(time.Now().UTC().String())

	for _, v := range secrets.List() {
		secret := v.(map[string]interface{})

		// base64 decode the payload
		payload, err := base64.StdEncoding.DecodeString(secret["payload"].(string))
		if err != nil {
			return fmt.Errorf("Invalid base64 value for secret '%s': %v", secret["name"].(string), err)
		}

		// build the kms decrypt params
		params := &kms.DecryptInput{
			CiphertextBlob: []byte(payload),
		}
		if context, exists := secret["context"]; exists {
			params.EncryptionContext = make(map[string]*string)
			for k, v := range context.(map[string]interface{}) {
				params.EncryptionContext[k] = aws.String(v.(string))
			}
		}
		if grant_tokens, exists := secret["grant_tokens"]; exists {
			params.GrantTokens = make([]*string, 0)
			for _, v := range grant_tokens.([]interface{}) {
				params.GrantTokens = append(params.GrantTokens, aws.String(v.(string)))
			}
		}

		// decrypt
		resp, err := conn.Decrypt(params)
		if err != nil {
			return fmt.Errorf("Failed to decrypt '%s': %s", secret["name"].(string), err)
		}

		// Set the secret via the name
		log.Printf("[DEBUG] aws_kms_secret - successfully decrypted secret: %s", secret["name"].(string))
		d.UnsafeSetFieldRaw(secret["name"].(string), string(resp.Plaintext))
	}

	return nil
}
