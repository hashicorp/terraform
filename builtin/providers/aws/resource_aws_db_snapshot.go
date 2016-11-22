package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDbSnapshot() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbSnapshotCreate,
		Read:   resourceAwsDbSnapshotRead,
		Delete: resourceAwsDbSnapshotDelete,

		Schema: map[string]*schema.Schema{
			"db_snapshot_identifier": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"db_instance_identifier": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"allocated_storage": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"availability_zone": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"db_snapshot_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"encrypted": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"engine": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"engine_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"iops": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"kms_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"license_model": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"option_group_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"port": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"source_db_snapshot_identifier": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"source_region": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"snapshot_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"storage_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsDbSnapshotCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	params := &rds.CreateDBSnapshotInput{
		DBInstanceIdentifier: aws.String(d.Get("db_instance_identifier").(string)),
		DBSnapshotIdentifier: aws.String(d.Get("db_snapshot_identifier").(string)),
	}

	_, err := conn.CreateDBSnapshot(params)
	if err != nil {
		return err
	}
	d.SetId(d.Get("db_snapshot_identifier").(string))

	_, err = resourceAwsDbSnapshotWaitForAvailable(d.Id(), conn)
	if err != nil {
		return err
	}

	return resourceAwsDbSnapshotRead(d, meta)
}

func resourceAwsDbSnapshotRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	params := &rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: aws.String(d.Id()),
	}
	resp, err := conn.DescribeDBSnapshots(params)
	if err != nil {
		return err
	}

	snapshot := resp.DBSnapshots[0]

	d.Set("allocated_storage", snapshot.AllocatedStorage)
	d.Set("availability_zone", snapshot.AvailabilityZone)
	d.Set("db_snapshot_arn", snapshot.DBSnapshotArn)
	d.Set("encrypted", snapshot.Encrypted)
	d.Set("engine", snapshot.Engine)
	d.Set("engine_version", snapshot.EngineVersion)
	d.Set("iops", snapshot.Iops)
	d.Set("kms_key_id", snapshot.KmsKeyId)
	d.Set("license_model", snapshot.LicenseModel)
	d.Set("option_group_name", snapshot.OptionGroupName)
	d.Set("port", snapshot.Port)
	d.Set("source_db_snapshot_identifier", snapshot.SourceDBSnapshotIdentifier)
	d.Set("source_region", snapshot.SourceRegion)
	d.Set("snapshot_type", snapshot.SnapshotType)
	d.Set("status", snapshot.Status)
	d.Set("vpc_id", snapshot.VpcId)

	return nil
}

func resourceAwsDbSnapshotDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	params := &rds.DeleteDBSnapshotInput{
		DBSnapshotIdentifier: aws.String(d.Id()),
	}
	_, err := conn.DeleteDBSnapshot(params)
	if err != nil {
		return err
	}

	return nil
}

func resourceAwsDbSnapshotWaitForAvailable(id string, conn *rds.RDS) (*rds.DBSnapshot, error) {
	log.Printf("Waiting for Snapshot %s to become available...", id)

	req := &rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: aws.String(id),
	}
	pollsWhereNotFound := 0
	for {
		res, err := conn.DescribeDBSnapshots(req)
		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "DBSnapshotNotFound" {
				pollsWhereNotFound++
				// We arbitrarily stop polling after getting a "not found" error five times,
				// assuming that the Snapshot has been deleted by something other than Terraform.
				if pollsWhereNotFound > 100 {
					return nil, fmt.Errorf("gave up waiting for Snapshot to be created: %s", err)
				}
				time.Sleep(30 * time.Second)
				continue
			}
			return nil, fmt.Errorf("error reading DB Snapshot: %s", err)
		}

		if len(res.DBSnapshots) != 1 {
			return nil, fmt.Errorf("new Snapshot vanished while pending")
		}

		state := *res.DBSnapshots[0].Status

		if state == "creating" {
			// Give it a few seconds before we poll again.
			time.Sleep(4 * time.Second)
			continue
		}

		if state == "available" {
			// We're done!
			return res.DBSnapshots[0], nil
		}

		return nil, fmt.Errorf("new Snapshot became %s while pending", state)
	}
}
