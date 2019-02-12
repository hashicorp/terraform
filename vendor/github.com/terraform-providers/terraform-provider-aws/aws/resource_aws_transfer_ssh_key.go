package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/transfer"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsTransferSshKey() *schema.Resource {

	return &schema.Resource{
		Create: resourceAwsTransferSshKeyCreate,
		Read:   resourceAwsTransferSshKeyRead,
		Delete: resourceAwsTransferSshKeyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"body": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					old = cleanSshKey(old)
					new = cleanSshKey(new)
					return strings.Trim(old, "\n") == strings.Trim(new, "\n")
				},
			},

			"server_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateTransferServerID,
			},

			"user_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateTransferUserName,
			},
		},
	}
}

func resourceAwsTransferSshKeyCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).transferconn
	userName := d.Get("user_name").(string)
	serverID := d.Get("server_id").(string)

	createOpts := &transfer.ImportSshPublicKeyInput{
		ServerId:         aws.String(serverID),
		UserName:         aws.String(userName),
		SshPublicKeyBody: aws.String(d.Get("body").(string)),
	}

	log.Printf("[DEBUG] Create Transfer SSH Public Key Option: %#v", createOpts)

	resp, err := conn.ImportSshPublicKey(createOpts)
	if err != nil {
		return fmt.Errorf("Error importing ssh public key: %s", err)
	}

	d.SetId(fmt.Sprintf("%s/%s/%s", serverID, userName, *resp.SshPublicKeyId))

	return nil
}

func resourceAwsTransferSshKeyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).transferconn
	serverID, userName, sshKeyID, err := decodeTransferSshKeyId(d.Id())
	if err != nil {
		return fmt.Errorf("error parsing Transfer SSH Public Key ID: %s", err)
	}

	descOpts := &transfer.DescribeUserInput{
		UserName: aws.String(userName),
		ServerId: aws.String(serverID),
	}

	log.Printf("[DEBUG] Describe Transfer User Option: %#v", descOpts)

	resp, err := conn.DescribeUser(descOpts)
	if err != nil {
		if isAWSErr(err, transfer.ErrCodeResourceNotFoundException, "") {
			log.Printf("[WARN] Transfer User (%s) for Server (%s) not found, removing ssh public key (%s) from state", userName, serverID, sshKeyID)
			d.SetId("")
			return nil
		}
		return err
	}

	var body string
	for _, s := range resp.User.SshPublicKeys {
		if sshKeyID == *s.SshPublicKeyId {
			body = *s.SshPublicKeyBody
		}
	}

	if body == "" {
		log.Printf("[WARN] No such ssh public key found for User (%s) in Server (%s)", userName, serverID)
		d.SetId("")
	}

	d.Set("server_id", resp.ServerId)
	d.Set("user_name", resp.User.UserName)
	d.Set("body", body)

	return nil
}

func resourceAwsTransferSshKeyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).transferconn
	serverID, userName, sshKeyID, err := decodeTransferSshKeyId(d.Id())
	if err != nil {
		return fmt.Errorf("error parsing Transfer SSH Public Key ID: %s", err)
	}

	delOpts := &transfer.DeleteSshPublicKeyInput{
		UserName:       aws.String(userName),
		ServerId:       aws.String(serverID),
		SshPublicKeyId: aws.String(sshKeyID),
	}

	log.Printf("[DEBUG] Delete Transfer SSH Public Key Option: %#v", delOpts)

	_, err = conn.DeleteSshPublicKey(delOpts)
	if err != nil {
		if isAWSErr(err, transfer.ErrCodeResourceNotFoundException, "") {
			return nil
		}
		return fmt.Errorf("error deleting Transfer User Ssh Key (%s): %s", d.Id(), err)
	}

	return nil
}

func decodeTransferSshKeyId(id string) (string, string, string, error) {
	idParts := strings.SplitN(id, "/", 3)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		return "", "", "", fmt.Errorf("unexpected format of ID (%s), expected SERVERID/USERNAME/SSHKEYID", id)
	}
	return idParts[0], idParts[1], idParts[2], nil
}
