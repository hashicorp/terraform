package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamUserSshKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamUserSshKeyCreate,
		Read:   resourceAwsIamUserSshKeyRead,
		Update: resourceAwsIamUserSshKeyUpdate,
		Delete: resourceAwsIamUserSshKeyDelete,

		Schema: map[string]*schema.Schema{
			"ssh_public_key_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"username": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"public_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"encoding": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateIamUserSSHKeyEncoding,
			},

			"status": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsIamUserSshKeyCreate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn
	username := d.Get("username").(string)
	publicKey := d.Get("public_key").(string)

	request := &iam.UploadSSHPublicKeyInput{
		UserName:         aws.String(username),
		SSHPublicKeyBody: aws.String(publicKey),
	}

	log.Println("[DEBUG] Create IAM User SSH Key Request:", request)
	createResp, err := iamconn.UploadSSHPublicKey(request)
	if err != nil {
		return fmt.Errorf("Error creating IAM User SSH Key %s: %s", username, err)
	}

	d.Set("ssh_public_key_id", createResp.SSHPublicKey.SSHPublicKeyId)
	d.SetId(*createResp.SSHPublicKey.SSHPublicKeyId)

	return resourceAwsIamUserSshKeyRead(d, meta)
}

func resourceAwsIamUserSshKeyRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn
	username := d.Get("username").(string)
	request := &iam.GetSSHPublicKeyInput{
		UserName:       aws.String(username),
		SSHPublicKeyId: aws.String(d.Id()),
		Encoding:       aws.String(d.Get("encoding").(string)),
	}

	getResp, err := iamconn.GetSSHPublicKey(request)
	if err != nil {
		if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" { // XXX test me
			log.Printf("[WARN] No IAM user ssh key (%s) found", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IAM User SSH Key %s: %s", d.Id(), err)
	}

	d.Set("fingerprint", getResp.SSHPublicKey.Fingerprint)
	d.Set("status", getResp.SSHPublicKey.Status)

	return nil
}

func resourceAwsIamUserSshKeyUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("status") {
		iamconn := meta.(*AWSClient).iamconn

		request := &iam.UpdateSSHPublicKeyInput{
			UserName:       aws.String(d.Get("username").(string)),
			SSHPublicKeyId: aws.String(d.Id()),
			Status:         aws.String(d.Get("status").(string)),
		}

		log.Println("[DEBUG] Update IAM User SSH Key request:", request)
		_, err := iamconn.UpdateSSHPublicKey(request)
		if err != nil {
			if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" {
				log.Printf("[WARN] No IAM user ssh key by ID (%s) found", d.Id())
				d.SetId("")
				return nil
			}
			return fmt.Errorf("Error updating IAM User SSH Key %s: %s", d.Id(), err)
		}
		return resourceAwsIamUserRead(d, meta)
	}
	return nil
}

func resourceAwsIamUserSshKeyDelete(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	request := &iam.DeleteSSHPublicKeyInput{
		UserName:       aws.String(d.Get("username").(string)),
		SSHPublicKeyId: aws.String(d.Id()),
	}

	log.Println("[DEBUG] Delete IAM User SSH Key request:", request)
	if _, err := iamconn.DeleteSSHPublicKey(request); err != nil {
		return fmt.Errorf("Error deleting IAM User SSH Key %s: %s", d.Id(), err)
	}
	return nil
}

func validateIamUserSSHKeyEncoding(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	encodingTypes := map[string]bool{
		"PEM": true,
		"SSH": true,
	}

	if !encodingTypes[value] {
		errors = append(errors, fmt.Errorf("IAM User SSH Key Encoding can only be PEM or SSH"))
	}
	return
}
