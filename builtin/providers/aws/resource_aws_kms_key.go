package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
)

func resourceAwsKmsKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsKmsKeyCreate,
		Read:   resourceAwsKmsKeyRead,
		Update: resourceAwsKmsKeyUpdate,
		Delete: resourceAwsKmsKeyDelete,

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"key_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},
			"key_usage": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if !(value == "ENCRYPT_DECRYPT" || value == "") {
						es = append(es, fmt.Errorf(
							"key_usage must be ENCRYPT_DECRYPT or not specified"))
					}
					return
				},
			},
			"policy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},
		},
	}
}

func resourceAwsKmsKeyCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn

	// Allow aws to chose default values if we don't pass them
	var req kms.CreateKeyInput
	if v, exists := d.GetOk("description"); exists {
		req.Description = aws.String(v.(string))
	}
	if v, exists := d.GetOk("key_usage"); exists {
		req.KeyUsage = aws.String(v.(string))
	}
	if v, exists := d.GetOk("policy"); exists {
		req.Policy = aws.String(v.(string))
	}

	resp, err := conn.CreateKey(&req)
	if err != nil {
		return err
	}
	return resourceAwsKmsKeyReadResult(d, resp.KeyMetadata)
}

func resourceAwsKmsKeyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn
	keyId := d.Get("key_id").(string)

	req := &kms.DescribeKeyInput{
		KeyId:       aws.String(keyId),
	}
	resp, err := conn.DescribeKey(req)
	if err != nil {
		return err
	}
	return resourceAwsKmsKeyReadResult(d, resp.KeyMetadata)
}

func resourceAwsKmsKeyReadResult(d *schema.ResourceData, metadata *kms.KeyMetadata) error {
	d.SetId(*metadata.KeyId)

	if err := d.Set("arn", metadata.Arn); err != nil {
		return err
	}
	if err := d.Set("key_id", metadata.KeyId); err != nil {
		return err
	}
	if err := d.Set("enabled", metadata.Enabled); err != nil {
		return err
	}
	if err := d.Set("description", metadata.Description); err != nil {
		return err
	}
	if err := d.Set("key_usage", metadata.KeyUsage); err != nil {
		return err
	}
	return nil
}

func resourceAwsKmsKeyUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn

	if d.HasChange("description") {
		if err := resourceAwsKmsKeyDescriptionUpdate(conn, d); err != nil {
			return err
		}
	}
	if d.HasChange("policy") {
		if err := resourceAwsKmsKeyPolicyUpdate(conn, d); err != nil {
			return err
		}
	}
	return resourceAwsKmsKeyRead(d, meta)
}

func resourceAwsKmsKeyDescriptionUpdate(conn *kms.KMS, d *schema.ResourceData) error {
	description := d.Get("description").(string)
	keyId := d.Get("key_id").(string)

	log.Printf("[DEBUG] KMS key: %s, update description: %s", keyId, description)

	req := &kms.UpdateKeyDescriptionInput{
		Description: aws.String(description),
		KeyId:       aws.String(keyId),
	}
	_, err := conn.UpdateKeyDescription(req)
	return err
}

func resourceAwsKmsKeyPolicyUpdate(conn *kms.KMS, d *schema.ResourceData) error {
	policy := d.Get("policy").(string)
	keyId := d.Get("key_id").(string)

	log.Printf("[DEBUG] KMS key: %s, update policy: %s", keyId, policy)

	req := &kms.PutKeyPolicyInput{
		KeyId:       aws.String(keyId),
		Policy:      aws.String(policy),
		PolicyName:  aws.String("default"),
	}
	_, err := conn.PutKeyPolicy(req)
	return err
}

func resourceAwsKmsKeyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn
	keyId := d.Get("key_id").(string)

	req := &kms.DisableKeyInput{
		KeyId:       aws.String(keyId),
	}
	_, err := conn.DisableKey(req)

	log.Printf("[DEBUG] KMS Key: %s deactivated.", keyId)
	d.SetId("")
	return err
}
