package aws

import (
	"bytes"
	"fmt"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsEbsVolume() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsEbsVolumeRead,

		Schema: map[string]*schema.Schema{
			"filter": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"values": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"most_recent": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
			"availability_zone": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"encrypted": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"iops": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"volume_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"snapshot_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"kms_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"volume_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": {
				Type:     schema.TypeSet,
				Computed: true,
				Set:      volumeTagsHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"value": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAwsEbsVolumeRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	filters, filtersOk := d.GetOk("filter")

	params := &ec2.DescribeVolumesInput{}
	if filtersOk {
		params.Filters = buildVolumeFilters(filters.(*schema.Set))
	}

	resp, err := conn.DescribeVolumes(params)
	if err != nil {
		return err
	}

	log.Printf("Found These Volumes %s", spew.Sdump(resp.Volumes))

	filteredVolumes := resp.Volumes[:]

	var volume *ec2.Volume
	if len(filteredVolumes) < 1 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(filteredVolumes) > 1 {
		recent := d.Get("most_recent").(bool)
		log.Printf("[DEBUG] aws_ebs_volume - multiple results found and `most_recent` is set to: %t", recent)
		if recent {
			volume = mostRecentVolume(filteredVolumes)
		} else {
			return fmt.Errorf("Your query returned more than one result. Please try a more " +
				"specific search criteria, or set `most_recent` attribute to true.")
		}
	} else {
		// Query returned single result.
		volume = filteredVolumes[0]
	}

	log.Printf("[DEBUG] aws_ebs_volume - Single Volume found: %s", *volume.VolumeId)
	return volumeDescriptionAttributes(d, volume)
}

func buildVolumeFilters(set *schema.Set) []*ec2.Filter {
	var filters []*ec2.Filter
	for _, v := range set.List() {
		m := v.(map[string]interface{})
		var filterValues []*string
		for _, e := range m["values"].([]interface{}) {
			filterValues = append(filterValues, aws.String(e.(string)))
		}
		filters = append(filters, &ec2.Filter{
			Name:   aws.String(m["name"].(string)),
			Values: filterValues,
		})
	}
	return filters
}

type volumeSort []*ec2.Volume

func (a volumeSort) Len() int      { return len(a) }
func (a volumeSort) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a volumeSort) Less(i, j int) bool {
	itime := *a[i].CreateTime
	jtime := *a[j].CreateTime
	return itime.Unix() < jtime.Unix()
}

func mostRecentVolume(volumes []*ec2.Volume) *ec2.Volume {
	sortedVolumes := volumes
	sort.Sort(volumeSort(sortedVolumes))
	return sortedVolumes[len(sortedVolumes)-1]
}

func volumeDescriptionAttributes(d *schema.ResourceData, volume *ec2.Volume) error {
	d.SetId(*volume.VolumeId)
	d.Set("volume_id", volume.VolumeId)
	d.Set("availability_zone", volume.AvailabilityZone)
	d.Set("encrypted", volume.Encrypted)
	d.Set("iops", volume.Iops)
	d.Set("kms_key_id", volume.KmsKeyId)
	d.Set("size", volume.Size)
	d.Set("snapshot_id", volume.SnapshotId)
	d.Set("volume_type", volume.VolumeType)

	if err := d.Set("tags", volumeTags(volume.Tags)); err != nil {
		return err
	}

	return nil
}

func volumeTags(m []*ec2.Tag) *schema.Set {
	s := &schema.Set{
		F: volumeTagsHash,
	}
	for _, v := range m {
		tag := map[string]interface{}{
			"key":   *v.Key,
			"value": *v.Value,
		}
		s.Add(tag)
	}
	return s
}

func volumeTagsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["key"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["value"].(string)))
	return hashcode.String(buf.String())
}
