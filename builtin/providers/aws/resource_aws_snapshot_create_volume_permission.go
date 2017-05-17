package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSnapshotCreateVolumePermission() *schema.Resource {
	return &schema.Resource{
		Exists: resourceAwsSnapshotCreateVolumePermissionExists,
		Create: resourceAwsSnapshotCreateVolumePermissionCreate,
		Read:   resourceAwsSnapshotCreateVolumePermissionRead,
		Delete: resourceAwsSnapshotCreateVolumePermissionDelete,

		Schema: map[string]*schema.Schema{
			"snapshot_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"account_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsSnapshotCreateVolumePermissionExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*AWSClient).ec2conn

	snapshot_id := d.Get("snapshot_id").(string)
	account_id := d.Get("account_id").(string)
	return hasCreateVolumePermission(conn, snapshot_id, account_id)
}

func resourceAwsSnapshotCreateVolumePermissionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	snapshot_id := d.Get("snapshot_id").(string)
	account_id := d.Get("account_id").(string)

	_, err := conn.ModifySnapshotAttribute(&ec2.ModifySnapshotAttributeInput{
		SnapshotId: aws.String(snapshot_id),
		Attribute:  aws.String("createVolumePermission"),
		CreateVolumePermission: &ec2.CreateVolumePermissionModifications{
			Add: []*ec2.CreateVolumePermission{
				&ec2.CreateVolumePermission{UserId: aws.String(account_id)},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("Error adding snapshot createVolumePermission: %s", err)
	}

	d.SetId(fmt.Sprintf("%s-%s", snapshot_id, account_id))

	// Wait for the account to appear in the permission list
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"denied"},
		Target:     []string{"granted"},
		Refresh:    resourceAwsSnapshotCreateVolumePermissionStateRefreshFunc(conn, snapshot_id, account_id),
		Timeout:    5 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 10 * time.Second,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for snapshot createVolumePermission (%s) to be added: %s",
			d.Id(), err)
	}

	return nil
}

func resourceAwsSnapshotCreateVolumePermissionRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsSnapshotCreateVolumePermissionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	snapshot_id := d.Get("snapshot_id").(string)
	account_id := d.Get("account_id").(string)

	_, err := conn.ModifySnapshotAttribute(&ec2.ModifySnapshotAttributeInput{
		SnapshotId: aws.String(snapshot_id),
		Attribute:  aws.String("createVolumePermission"),
		CreateVolumePermission: &ec2.CreateVolumePermissionModifications{
			Remove: []*ec2.CreateVolumePermission{
				&ec2.CreateVolumePermission{UserId: aws.String(account_id)},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("Error removing snapshot createVolumePermission: %s", err)
	}

	// Wait for the account to disappear from the permission list
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"granted"},
		Target:     []string{"denied"},
		Refresh:    resourceAwsSnapshotCreateVolumePermissionStateRefreshFunc(conn, snapshot_id, account_id),
		Timeout:    5 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 10 * time.Second,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for snapshot createVolumePermission (%s) to be removed: %s",
			d.Id(), err)
	}

	return nil
}

func hasCreateVolumePermission(conn *ec2.EC2, snapshot_id string, account_id string) (bool, error) {
	_, state, err := resourceAwsSnapshotCreateVolumePermissionStateRefreshFunc(conn, snapshot_id, account_id)()
	if err != nil {
		return false, err
	}
	if state == "granted" {
		return true, nil
	} else {
		return false, nil
	}
}

func resourceAwsSnapshotCreateVolumePermissionStateRefreshFunc(conn *ec2.EC2, snapshot_id string, account_id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		attrs, err := conn.DescribeSnapshotAttribute(&ec2.DescribeSnapshotAttributeInput{
			SnapshotId: aws.String(snapshot_id),
			Attribute:  aws.String("createVolumePermission"),
		})
		if err != nil {
			return nil, "", fmt.Errorf("Error refreshing snapshot createVolumePermission state: %s", err)
		}

		for _, vp := range attrs.CreateVolumePermissions {
			if *vp.UserId == account_id {
				return attrs, "granted", nil
			}
		}
		return attrs, "denied", nil
	}
}
