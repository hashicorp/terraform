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
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
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
							"%q must be ENCRYPT_DECRYPT or not specified", k))
					}
					return
				},
			},
			"policy": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				StateFunc: normalizeJson,
			},
			"deletion_window_in_days": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(int)
					if value > 30 || value < 7 {
						es = append(es, fmt.Errorf(
							"%q must be between 7 and 30 days inclusive", k))
					}
					return
				},
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

	d.SetId(*resp.KeyMetadata.KeyId)

	return resourceAwsKmsKeyRead(d, meta)
}

func resourceAwsKmsKeyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn

	req := &kms.DescribeKeyInput{
		KeyId: aws.String(d.Id()),
	}
	resp, err := conn.DescribeKey(req)
	if err != nil {
		return err
	}
	metadata := resp.KeyMetadata

	d.SetId(*metadata.KeyId)

	d.Set("arn", metadata.Arn)
	d.Set("key_id", metadata.KeyId)
	d.Set("description", metadata.Description)
	d.Set("key_usage", metadata.KeyUsage)

	p, err := conn.GetKeyPolicy(&kms.GetKeyPolicyInput{
		KeyId:      metadata.KeyId,
		PolicyName: aws.String("default"),
	})
	if err != nil {
		return err
	}

	d.Set("policy", normalizeJson(*p.Policy))

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
		KeyId:      aws.String(keyId),
		Policy:     aws.String(normalizeJson(policy)),
		PolicyName: aws.String("default"),
	}
	_, err := conn.PutKeyPolicy(req)
	return err
}

func resourceAwsKmsKeyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn
	keyId := d.Get("key_id").(string)

	req := &kms.ScheduleKeyDeletionInput{
		KeyId: aws.String(keyId),
	}
	if v, exists := d.GetOk("deletion_window_in_days"); exists {
		req.PendingWindowInDays = aws.Int64(int64(v.(int)))
	}
	_, err := conn.ScheduleKeyDeletion(req)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] KMS Key: %s deactivated.", keyId)
	d.SetId("")
	return nil
}
