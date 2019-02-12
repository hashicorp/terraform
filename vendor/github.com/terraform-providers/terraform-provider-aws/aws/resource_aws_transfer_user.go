package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/transfer"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsTransferUser() *schema.Resource {

	return &schema.Resource{
		Create: resourceAwsTransferUserCreate,
		Read:   resourceAwsTransferUserRead,
		Update: resourceAwsTransferUserUpdate,
		Delete: resourceAwsTransferUserDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"home_directory": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(0, 1024),
			},

			"policy": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateFunc:     validateIAMPolicyJson,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
			},

			"role": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateArn,
			},

			"server_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateTransferServerID,
			},

			"tags": tagsSchema(),

			"user_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateTransferUserName,
			},
		},
	}
}

func resourceAwsTransferUserCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).transferconn
	userName := d.Get("user_name").(string)
	serverID := d.Get("server_id").(string)

	createOpts := &transfer.CreateUserInput{
		ServerId: aws.String(serverID),
		UserName: aws.String(userName),
		Role:     aws.String(d.Get("role").(string)),
	}

	if attr, ok := d.GetOk("home_directory"); ok {
		createOpts.HomeDirectory = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("policy"); ok {
		createOpts.Policy = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("tags"); ok {
		createOpts.Tags = tagsFromMapTransfer(attr.(map[string]interface{}))
	}

	log.Printf("[DEBUG] Create Transfer User Option: %#v", createOpts)

	_, err := conn.CreateUser(createOpts)
	if err != nil {
		return fmt.Errorf("error creating Transfer User: %s", err)
	}

	d.SetId(fmt.Sprintf("%s/%s", serverID, userName))

	return resourceAwsTransferUserRead(d, meta)
}

func resourceAwsTransferUserRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).transferconn
	serverID, userName, err := decodeTransferUserId(d.Id())
	if err != nil {
		return fmt.Errorf("error parsing Transfer User ID: %s", err)
	}

	descOpts := &transfer.DescribeUserInput{
		UserName: aws.String(userName),
		ServerId: aws.String(serverID),
	}

	log.Printf("[DEBUG] Describe Transfer User Option: %#v", descOpts)

	resp, err := conn.DescribeUser(descOpts)
	if err != nil {
		if isAWSErr(err, transfer.ErrCodeResourceNotFoundException, "") {
			log.Printf("[WARN] Transfer User (%s) for Server (%s) not found, removing from state", userName, serverID)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading Transfer User (%s): %s", d.Id(), err)
	}

	d.Set("server_id", resp.ServerId)
	d.Set("user_name", resp.User.UserName)
	d.Set("arn", resp.User.Arn)
	d.Set("home_directory", resp.User.HomeDirectory)
	d.Set("policy", resp.User.Policy)
	d.Set("role", resp.User.Role)

	if err := d.Set("tags", tagsToMapTransfer(resp.User.Tags)); err != nil {
		return fmt.Errorf("Error setting tags: %s", err)
	}
	return nil
}

func resourceAwsTransferUserUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).transferconn
	updateFlag := false
	serverID, userName, err := decodeTransferUserId(d.Id())
	if err != nil {
		return fmt.Errorf("error parsing Transfer User ID: %s", err)
	}

	updateOpts := &transfer.UpdateUserInput{
		UserName: aws.String(userName),
		ServerId: aws.String(serverID),
	}

	if d.HasChange("home_directory") {
		updateOpts.HomeDirectory = aws.String(d.Get("home_directory").(string))
		updateFlag = true
	}

	if d.HasChange("policy") {
		updateOpts.Policy = aws.String(d.Get("policy").(string))
		updateFlag = true
	}

	if d.HasChange("role") {
		updateOpts.Role = aws.String(d.Get("role").(string))
		updateFlag = true
	}

	if updateFlag {
		_, err := conn.UpdateUser(updateOpts)
		if err != nil {
			if isAWSErr(err, transfer.ErrCodeResourceNotFoundException, "") {
				log.Printf("[WARN] Transfer User (%s) for Server (%s) not found, removing from state", userName, serverID)
				d.SetId("")
				return nil
			}
			return fmt.Errorf("error updating Transfer User (%s): %s", d.Id(), err)
		}
	}

	if err := setTagsTransfer(conn, d); err != nil {
		return fmt.Errorf("Error update tags: %s", err)
	}

	return resourceAwsTransferUserRead(d, meta)
}

func resourceAwsTransferUserDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).transferconn
	serverID, userName, err := decodeTransferUserId(d.Id())
	if err != nil {
		return fmt.Errorf("error parsing Transfer User ID: %s", err)
	}

	delOpts := &transfer.DeleteUserInput{
		UserName: aws.String(userName),
		ServerId: aws.String(serverID),
	}

	log.Printf("[DEBUG] Delete Transfer User Option: %#v", delOpts)

	_, err = conn.DeleteUser(delOpts)
	if err != nil {
		if isAWSErr(err, transfer.ErrCodeResourceNotFoundException, "") {
			return nil
		}
		return fmt.Errorf("error deleting Transfer User (%s) for Server(%s): %s", userName, serverID, err)
	}

	if err := waitForTransferUserDeletion(conn, serverID, userName); err != nil {
		return fmt.Errorf("error waiting for Transfer User (%s) for Server (%s): %s", userName, serverID, err)
	}

	return nil
}

func decodeTransferUserId(id string) (string, string, error) {
	idParts := strings.SplitN(id, "/", 2)
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		return "", "", fmt.Errorf("unexpected format of ID (%s), expected SERVERID/USERNAME", id)
	}
	return idParts[0], idParts[1], nil
}

func waitForTransferUserDeletion(conn *transfer.Transfer, serverID, userName string) error {
	params := &transfer.DescribeUserInput{
		ServerId: aws.String(serverID),
		UserName: aws.String(userName),
	}

	return resource.Retry(10*time.Minute, func() *resource.RetryError {
		_, err := conn.DescribeUser(params)

		if isAWSErr(err, transfer.ErrCodeResourceNotFoundException, "") {
			return nil
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return resource.RetryableError(fmt.Errorf("Transfer User (%s) for Server (%s) still exists", userName, serverID))
	})
}
