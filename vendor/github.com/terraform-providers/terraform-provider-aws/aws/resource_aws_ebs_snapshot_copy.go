package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEbsSnapshotCopy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEbsSnapshotCopyCreate,
		Read:   resourceAwsEbsSnapshotCopyRead,
		Delete: resourceAwsEbsSnapshotCopyDelete,

		Schema: map[string]*schema.Schema{
			"volume_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner_alias": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"encrypted": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"volume_size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"kms_key_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"data_encryption_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"source_region": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"source_snapshot_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsEbsSnapshotCopyCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	request := &ec2.CopySnapshotInput{
		SourceRegion:     aws.String(d.Get("source_region").(string)),
		SourceSnapshotId: aws.String(d.Get("source_snapshot_id").(string)),
	}
	if v, ok := d.GetOk("description"); ok {
		request.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("encrypted"); ok {
		request.Encrypted = aws.Bool(v.(bool))
	}
	if v, ok := d.GetOk("kms_key_id"); ok {
		request.KmsKeyId = aws.String(v.(string))
	}

	res, err := conn.CopySnapshot(request)
	if err != nil {
		return err
	}

	d.SetId(*res.SnapshotId)

	err = resourceAwsEbsSnapshotCopyWaitForAvailable(d.Id(), conn)
	if err != nil {
		return err
	}

	if err := setTags(conn, d); err != nil {
		log.Printf("[WARN] error setting tags: %s", err)
	}

	return resourceAwsEbsSnapshotCopyRead(d, meta)
}

func resourceAwsEbsSnapshotCopyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeSnapshotsInput{
		SnapshotIds: []*string{aws.String(d.Id())},
	}
	res, err := conn.DescribeSnapshots(req)
	if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidSnapshotID.NotFound" {
		log.Printf("Snapshot %q Not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	snapshot := res.Snapshots[0]

	d.Set("description", snapshot.Description)
	d.Set("owner_id", snapshot.OwnerId)
	d.Set("encrypted", snapshot.Encrypted)
	d.Set("owner_alias", snapshot.OwnerAlias)
	d.Set("volume_id", snapshot.VolumeId)
	d.Set("data_encryption_key_id", snapshot.DataEncryptionKeyId)
	d.Set("kms_key_id", snapshot.KmsKeyId)
	d.Set("volume_size", snapshot.VolumeSize)

	if err := d.Set("tags", tagsToMap(snapshot.Tags)); err != nil {
		log.Printf("[WARN] error saving tags to state: %s", err)
	}

	return nil
}

func resourceAwsEbsSnapshotCopyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		request := &ec2.DeleteSnapshotInput{
			SnapshotId: aws.String(d.Id()),
		}
		_, err := conn.DeleteSnapshot(request)
		if err == nil {
			return nil
		}

		ebsErr, ok := err.(awserr.Error)
		if ebsErr.Code() == "SnapshotInUse" {
			return resource.RetryableError(fmt.Errorf("EBS SnapshotInUse - trying again while it detaches"))
		}

		if !ok {
			return resource.NonRetryableError(err)
		}

		return resource.NonRetryableError(err)
	})
}

func resourceAwsEbsSnapshotCopyWaitForAvailable(id string, conn *ec2.EC2) error {
	log.Printf("Waiting for Snapshot %s to become available...", id)

	req := &ec2.DescribeSnapshotsInput{
		SnapshotIds: []*string{aws.String(id)},
	}
	err := conn.WaitUntilSnapshotCompleted(req)
	return err
}
