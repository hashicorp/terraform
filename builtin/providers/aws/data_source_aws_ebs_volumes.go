package aws

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsEbsVolumes() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsEbsVolumesRead,

		Schema: map[string]*schema.Schema{
			"filter": dataSourceFiltersSchema(),
			"ids": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"volumes": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     dataSourceAwsEbsVolume(),
			},
		},
	}
}

func dataSourceAwsEbsVolumesRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[DEBUG] Reading EBS Volumes.")
	d.SetId(time.Now().UTC().String())

	filters, filtersOk := d.GetOk("filter")

	params := &ec2.DescribeVolumesInput{}
	if filtersOk {
		params.Filters = buildAwsDataSourceFilters(filters.(*schema.Set))
	}

	resp, err := conn.DescribeVolumes(params)
	if err != nil {
		return err
	}

	log.Printf("Found These Volumes %s", spew.Sdump(resp.Volumes))

	filteredVolumes := resp.Volumes[:]

	if len(filteredVolumes) < 1 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again")
	}

	log.Printf("[DEBUG] aws_ebs_volume - Volumes found: %s", len(filteredVolumes))

	volumes, ids := volumesList(filteredVolumes)

	if err := d.Set("volumes", volumes); err != nil {
		return fmt.Errorf("[WARN] Error setting EBS Volumes: %s", err)
	}

	if err := d.Set("ids", ids); err != nil {
		return fmt.Errorf("[WARN] Error setting ids: %s", err)
	}

	return nil
}

type volumesSort []*ec2.Volume

func (a volumesSort) Len() int      { return len(a) }
func (a volumesSort) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a volumesSort) Less(i, j int) bool {
	itime := *a[i].CreateTime
	jtime := *a[j].CreateTime
	return itime.Before(jtime)
}

func volumesList(volumes []*ec2.Volume) ([]interface{}, []string) {
	sortedVolumes := volumes
	sort.Sort(volumesSort(sortedVolumes))

	var vols []interface{}
	var ids []string

	for i, v := range sortedVolumes {

		var vol = map[string]interface{}{
			"volume_id":         *v.VolumeId,
			"availability_zone": *v.AvailabilityZone,
			"encrypted":         *v.Encrypted,
			"size":              *v.Size,
			"snapshot_id":       *v.SnapshotId,
			"volume_type":       *v.VolumeType,
			"most_recent":       i == 0,
			"tags":              dataSourceTags(v.Tags),
		}

		if v.Iops != nil {
			vol["iops"] = *v.Iops
		}

		if v.KmsKeyId != nil {
			vol["kms_key_id"] = *v.KmsKeyId
		}

		vols = append(vols, vol)
		ids = append(ids, *v.VolumeId)
	}
	return vols, ids
}
