package aws

import (
	"fmt"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsEbsSnapshotIds() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsEbsSnapshotIdsRead,

		Schema: map[string]*schema.Schema{
			"filter": dataSourceFiltersSchema(),
			"owners": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"restorable_by_user_ids": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"ids": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceAwsEbsSnapshotIdsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	restorableUsers, restorableUsersOk := d.GetOk("restorable_by_user_ids")
	filters, filtersOk := d.GetOk("filter")
	owners, ownersOk := d.GetOk("owners")

	if restorableUsers == false && !filtersOk && !ownersOk {
		return fmt.Errorf("One of filters, restorable_by_user_ids, or owners must be assigned")
	}

	params := &ec2.DescribeSnapshotsInput{}

	if restorableUsersOk {
		params.RestorableByUserIds = expandStringList(restorableUsers.([]interface{}))
	}
	if filtersOk {
		params.Filters = buildAwsDataSourceFilters(filters.(*schema.Set))
	}
	if ownersOk {
		params.OwnerIds = expandStringList(owners.([]interface{}))
	}

	log.Printf("[DEBUG] Reading EBS Snapshot IDs: %s", params)
	resp, err := conn.DescribeSnapshots(params)
	if err != nil {
		return err
	}

	snapshotIds := make([]string, 0)

	sort.Slice(resp.Snapshots, func(i, j int) bool {
		return aws.TimeValue(resp.Snapshots[i].StartTime).Unix() > aws.TimeValue(resp.Snapshots[j].StartTime).Unix()
	})
	for _, snapshot := range resp.Snapshots {
		snapshotIds = append(snapshotIds, *snapshot.SnapshotId)
	}

	d.SetId(fmt.Sprintf("%d", hashcode.String(params.String())))
	d.Set("ids", snapshotIds)

	return nil
}
