package aws

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamAccessKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamAccessKeyCreate,
		Read:   resourceAwsIamAccessKeyRead,
		Delete: resourceAwsIamAccessKeyDelete,

		Schema: map[string]*schema.Schema{
			"user": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"status": &schema.Schema{
				Type: schema.TypeString,
				// this could be settable, but goamz does not support the
				// UpdateAccessKey API yet.
				Computed: true,
			},
			"secret": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"ses_smtp_password": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsIamAccessKeyCreate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	request := &iam.CreateAccessKeyInput{
		UserName: aws.String(d.Get("user").(string)),
	}

	createResp, err := iamconn.CreateAccessKey(request)
	if err != nil {
		return fmt.Errorf(
			"Error creating access key for user %s: %s",
			*request.UserName,
			err,
		)
	}

	if err := d.Set("secret", createResp.AccessKey.SecretAccessKey); err != nil {
		return err
	}

	d.Set("ses_smtp_password",
		sesSmtpPasswordFromSecretKey(createResp.AccessKey.SecretAccessKey))

	return resourceAwsIamAccessKeyReadResult(d, &iam.AccessKeyMetadata{
		AccessKeyId: createResp.AccessKey.AccessKeyId,
		CreateDate:  createResp.AccessKey.CreateDate,
		Status:      createResp.AccessKey.Status,
		UserName:    createResp.AccessKey.UserName,
	})
}

func resourceAwsIamAccessKeyRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	request := &iam.ListAccessKeysInput{
		UserName: aws.String(d.Get("user").(string)),
	}

	getResp, err := iamconn.ListAccessKeys(request)
	if err != nil {
		if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" { // XXX TEST ME
			// the user does not exist, so the key can't exist.
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IAM acces key: %s", err)
	}

	for _, key := range getResp.AccessKeyMetadata {
		if key.AccessKeyId != nil && *key.AccessKeyId == d.Id() {
			return resourceAwsIamAccessKeyReadResult(d, key)
		}
	}

	// Guess the key isn't around anymore.
	d.SetId("")
	return nil
}

func resourceAwsIamAccessKeyReadResult(d *schema.ResourceData, key *iam.AccessKeyMetadata) error {
	d.SetId(*key.AccessKeyId)
	if err := d.Set("user", key.UserName); err != nil {
		return err
	}
	if err := d.Set("status", key.Status); err != nil {
		return err
	}
	return nil
}

func resourceAwsIamAccessKeyDelete(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	request := &iam.DeleteAccessKeyInput{
		AccessKeyId: aws.String(d.Id()),
		UserName:    aws.String(d.Get("user").(string)),
	}

	if _, err := iamconn.DeleteAccessKey(request); err != nil {
		return fmt.Errorf("Error deleting access key %s: %s", d.Id(), err)
	}
	return nil
}

func sesSmtpPasswordFromSecretKey(key *string) string {
	if key == nil {
		return ""
	}
	version := byte(0x02)
	message := []byte("SendRawEmail")
	hmacKey := []byte(*key)
	h := hmac.New(sha256.New, hmacKey)
	h.Write(message)
	rawSig := h.Sum(nil)
	versionedSig := make([]byte, 0, len(rawSig)+1)
	versionedSig = append(versionedSig, version)
	versionedSig = append(versionedSig, rawSig...)
	return base64.StdEncoding.EncodeToString(versionedSig)
}
