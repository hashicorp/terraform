package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsIamUserSshKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamUserSshKeyCreate,
		Read:   resourceAwsIamUserSshKeyRead,
		Update: resourceAwsIamUserSshKeyUpdate,
		Delete: resourceAwsIamUserSshKeyDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsIamUserSshKeyImport,
		},

		Schema: map[string]*schema.Schema{
			"ssh_public_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"fingerprint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"username": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"public_key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if d.Get("encoding").(string) == "SSH" {
						old = cleanSshKey(old)
						new = cleanSshKey(new)
					}
					return strings.Trim(old, "\n") == strings.Trim(new, "\n")
				},
			},

			"encoding": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					iam.EncodingTypeSsh,
					iam.EncodingTypePem,
				}, false),
			},

			"status": {
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

	d.SetId(*createResp.SSHPublicKey.SSHPublicKeyId)

	return resourceAwsIamUserSshKeyUpdate(d, meta)
}

func resourceAwsIamUserSshKeyRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn
	username := d.Get("username").(string)
	encoding := d.Get("encoding").(string)
	request := &iam.GetSSHPublicKeyInput{
		UserName:       aws.String(username),
		SSHPublicKeyId: aws.String(d.Id()),
		Encoding:       aws.String(encoding),
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

	publicKey := *getResp.SSHPublicKey.SSHPublicKeyBody
	if encoding == "SSH" {
		publicKey = cleanSshKey(publicKey)
	}

	d.Set("fingerprint", getResp.SSHPublicKey.Fingerprint)
	d.Set("status", getResp.SSHPublicKey.Status)
	d.Set("ssh_public_key_id", getResp.SSHPublicKey.SSHPublicKeyId)
	d.Set("public_key", publicKey)
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
	}
	return resourceAwsIamUserSshKeyRead(d, meta)
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

func resourceAwsIamUserSshKeyImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.SplitN(d.Id(), ":", 3)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		return nil, fmt.Errorf("unexpected format of ID (%q), UserName:SSHPublicKeyId:Encoding", d.Id())
	}

	username := idParts[0]
	sshPublicKeyId := idParts[1]
	encoding := idParts[2]

	d.Set("username", username)
	d.Set("ssh_public_key_id", sshPublicKeyId)
	d.Set("encoding", encoding)
	d.SetId(sshPublicKeyId)

	return []*schema.ResourceData{d}, nil
}

func cleanSshKey(key string) string {
	// Remove comments from SSH Keys
	// Comments are anything after "ssh-rsa XXXX" where XXXX is the key.
	parts := strings.Split(key, " ")
	if len(parts) > 2 {
		parts = parts[0:2]
	}
	return strings.Join(parts, " ")
}
