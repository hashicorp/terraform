package aws

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsDbSnapshot() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsDbSnapshotRead,

		Schema: map[string]*schema.Schema{
			//selection criteria
			"db_instance_identifier": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"db_snapshot_identifier": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"snapshot_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"include_shared": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},

			"include_public": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},
			"most_recent": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			//Computed values returned
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
			"snapshot_create_time": {
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

func dataSourceAwsDbSnapshotRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	instanceIdentifier, instanceIdentifierOk := d.GetOk("db_instance_identifier")
	snapshotIdentifier, snapshotIdentifierOk := d.GetOk("db_snapshot_identifier")

	if !instanceIdentifierOk && !snapshotIdentifierOk {
		return fmt.Errorf("One of db_snapshot_identifier or db_instance_identifier must be assigned")
	}

	params := &rds.DescribeDBSnapshotsInput{
		IncludePublic: aws.Bool(d.Get("include_public").(bool)),
		IncludeShared: aws.Bool(d.Get("include_shared").(bool)),
	}
	if v, ok := d.GetOk("snapshot_type"); ok {
		params.SnapshotType = aws.String(v.(string))
	}
	if instanceIdentifierOk {
		params.DBInstanceIdentifier = aws.String(instanceIdentifier.(string))
	}
	if snapshotIdentifierOk {
		params.DBSnapshotIdentifier = aws.String(snapshotIdentifier.(string))
	}

	log.Printf("[DEBUG] Reading DB Snapshot: %s", params)
	resp, err := conn.DescribeDBSnapshots(params)
	if err != nil {
		return err
	}

	if len(resp.DBSnapshots) < 1 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	var snapshot *rds.DBSnapshot
	if len(resp.DBSnapshots) > 1 {
		recent := d.Get("most_recent").(bool)
		log.Printf("[DEBUG] aws_db_snapshot - multiple results found and `most_recent` is set to: %t", recent)
		if recent {
			snapshot = mostRecentDbSnapshot(resp.DBSnapshots)
		} else {
			return fmt.Errorf("Your query returned more than one result. Please try a more specific search criteria.")
		}
	} else {
		snapshot = resp.DBSnapshots[0]
	}

	return dbSnapshotDescriptionAttributes(d, snapshot)
}

type rdsSnapshotSort []*rds.DBSnapshot

func (a rdsSnapshotSort) Len() int      { return len(a) }
func (a rdsSnapshotSort) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a rdsSnapshotSort) Less(i, j int) bool {
	// Snapshot creation can be in progress
	if a[i].SnapshotCreateTime == nil {
		return true
	}
	if a[j].SnapshotCreateTime == nil {
		return false
	}

	return (*a[i].SnapshotCreateTime).Before(*a[j].SnapshotCreateTime)
}

func mostRecentDbSnapshot(snapshots []*rds.DBSnapshot) *rds.DBSnapshot {
	sortedSnapshots := snapshots
	sort.Sort(rdsSnapshotSort(sortedSnapshots))
	return sortedSnapshots[len(sortedSnapshots)-1]
}

func dbSnapshotDescriptionAttributes(d *schema.ResourceData, snapshot *rds.DBSnapshot) error {
	d.SetId(*snapshot.DBSnapshotIdentifier)
	d.Set("db_instance_identifier", snapshot.DBInstanceIdentifier)
	d.Set("db_snapshot_identifier", snapshot.DBSnapshotIdentifier)
	d.Set("snapshot_type", snapshot.SnapshotType)
	d.Set("storage_type", snapshot.StorageType)
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
	d.Set("status", snapshot.Status)
	d.Set("vpc_id", snapshot.VpcId)
	if snapshot.SnapshotCreateTime != nil {
		d.Set("snapshot_create_time", snapshot.SnapshotCreateTime.Format(time.RFC3339))
	}

	return nil
}
