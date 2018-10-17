package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDbClusterSnapshot() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbClusterSnapshotCreate,
		Read:   resourceAwsDbClusterSnapshotRead,
		Delete: resourceAwsDbClusterSnapshotDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"db_cluster_snapshot_identifier": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"db_cluster_identifier": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"allocated_storage": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"availability_zones": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
			"db_cluster_snapshot_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"storage_encrypted": {
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
			"kms_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"license_model": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"port": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"source_db_cluster_snapshot_arn": {
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
			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsDbClusterSnapshotCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	params := &rds.CreateDBClusterSnapshotInput{
		DBClusterIdentifier:         aws.String(d.Get("db_cluster_identifier").(string)),
		DBClusterSnapshotIdentifier: aws.String(d.Get("db_cluster_snapshot_identifier").(string)),
	}

	_, err := conn.CreateDBClusterSnapshot(params)
	if err != nil {
		return fmt.Errorf("error creating RDS DB Cluster Snapshot: %s", err)
	}
	d.SetId(d.Get("db_cluster_snapshot_identifier").(string))

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating"},
		Target:     []string{"available"},
		Refresh:    resourceAwsDbClusterSnapshotStateRefreshFunc(d.Id(), conn),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      5 * time.Second,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("error waiting for RDS DB Cluster Snapshot %q to create: %s", d.Id(), err)
	}

	return resourceAwsDbClusterSnapshotRead(d, meta)
}

func resourceAwsDbClusterSnapshotRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	params := &rds.DescribeDBClusterSnapshotsInput{
		DBClusterSnapshotIdentifier: aws.String(d.Id()),
	}
	resp, err := conn.DescribeDBClusterSnapshots(params)
	if err != nil {
		if isAWSErr(err, rds.ErrCodeDBClusterSnapshotNotFoundFault, "") {
			log.Printf("[WARN] RDS DB Cluster Snapshot %q not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading RDS DB Cluster Snapshot %q: %s", d.Id(), err)
	}

	if resp == nil || len(resp.DBClusterSnapshots) == 0 || resp.DBClusterSnapshots[0] == nil || aws.StringValue(resp.DBClusterSnapshots[0].DBClusterSnapshotIdentifier) != d.Id() {
		log.Printf("[WARN] RDS DB Cluster Snapshot %q not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	snapshot := resp.DBClusterSnapshots[0]

	d.Set("allocated_storage", snapshot.AllocatedStorage)
	if err := d.Set("availability_zones", flattenStringList(snapshot.AvailabilityZones)); err != nil {
		return fmt.Errorf("error setting availability_zones: %s", err)
	}
	d.Set("db_cluster_identifier", snapshot.DBClusterIdentifier)
	d.Set("db_cluster_snapshot_arn", snapshot.DBClusterSnapshotArn)
	d.Set("db_cluster_snapshot_identifier", snapshot.DBClusterSnapshotIdentifier)
	d.Set("engine_version", snapshot.EngineVersion)
	d.Set("engine", snapshot.Engine)
	d.Set("kms_key_id", snapshot.KmsKeyId)
	d.Set("license_model", snapshot.LicenseModel)
	d.Set("port", snapshot.Port)
	d.Set("snapshot_type", snapshot.SnapshotType)
	d.Set("source_db_cluster_snapshot_arn", snapshot.SourceDBClusterSnapshotArn)
	d.Set("status", snapshot.Status)
	d.Set("storage_encrypted", snapshot.StorageEncrypted)
	d.Set("vpc_id", snapshot.VpcId)

	return nil
}

func resourceAwsDbClusterSnapshotDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	params := &rds.DeleteDBClusterSnapshotInput{
		DBClusterSnapshotIdentifier: aws.String(d.Id()),
	}
	_, err := conn.DeleteDBClusterSnapshot(params)
	if err != nil {
		if isAWSErr(err, rds.ErrCodeDBClusterSnapshotNotFoundFault, "") {
			return nil
		}
		return fmt.Errorf("error deleting RDS DB Cluster Snapshot %q: %s", d.Id(), err)
	}

	return nil
}

func resourceAwsDbClusterSnapshotStateRefreshFunc(dbClusterSnapshotIdentifier string, conn *rds.RDS) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		opts := &rds.DescribeDBClusterSnapshotsInput{
			DBClusterSnapshotIdentifier: aws.String(dbClusterSnapshotIdentifier),
		}

		log.Printf("[DEBUG] DB Cluster Snapshot describe configuration: %#v", opts)

		resp, err := conn.DescribeDBClusterSnapshots(opts)
		if err != nil {
			if isAWSErr(err, rds.ErrCodeDBClusterSnapshotNotFoundFault, "") {
				return nil, "", nil
			}
			return nil, "", fmt.Errorf("Error retrieving DB Cluster Snapshots: %s", err)
		}

		if resp == nil || len(resp.DBClusterSnapshots) == 0 || resp.DBClusterSnapshots[0] == nil {
			return nil, "", fmt.Errorf("No snapshots returned for %s", dbClusterSnapshotIdentifier)
		}

		snapshot := resp.DBClusterSnapshots[0]

		return resp, aws.StringValue(snapshot.Status), nil
	}
}
