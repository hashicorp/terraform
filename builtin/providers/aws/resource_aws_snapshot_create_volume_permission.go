package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
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
		return fmt.Errorf("error creating snapshot volume permission: %s", err)
	}

	d.SetId(fmt.Sprintf("%s-%s", snapshot_id, account_id))
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
		return fmt.Errorf("error removing snapshot volume permission: %s", err)
	}

	return nil
}

func hasCreateVolumePermission(conn *ec2.EC2, snapshot_id string, account_id string) (bool, error) {
	attrs, err := conn.DescribeSnapshotAttribute(&ec2.DescribeSnapshotAttributeInput{
		SnapshotId: aws.String(snapshot_id),
		Attribute:  aws.String("createVolumePermission"),
	})
	if err != nil {
		return false, err
	}

	for _, vp := range attrs.CreateVolumePermissions {
		if *vp.UserId == account_id {
			return true, nil
		}
	}
	return false, nil
}
