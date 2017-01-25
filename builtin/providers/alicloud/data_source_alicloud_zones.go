package alicloud

import (
	"fmt"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"reflect"
)

func dataSourceAlicloudZones() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAlicloudZonesRead,

		Schema: map[string]*schema.Schema{
			"available_instance_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"available_resource_creation": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"available_disk_category": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			// Computed values.
			"zones": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"local_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"available_instance_types": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"available_resource_creation": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"available_disk_categories": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func dataSourceAlicloudZonesRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	insType, _ := d.Get("available_instance_type").(string)
	resType, _ := d.Get("available_resource_creation").(string)
	diskType, _ := d.Get("available_disk_category").(string)

	resp, err := conn.DescribeZones(getRegion(d, meta))
	if err != nil {
		return err
	}

	var zoneTypes []ecs.ZoneType
	for _, types := range resp {
		if insType != "" && !constraints(types.AvailableInstanceTypes.InstanceTypes, insType) {
			continue
		}

		if resType != "" && !constraints(types.AvailableResourceCreation.ResourceTypes, resType) {
			continue
		}

		if diskType != "" && !constraints(types.AvailableDiskCategories.DiskCategories, diskType) {
			continue
		}
		zoneTypes = append(zoneTypes, types)
	}

	if len(zoneTypes) < 1 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	log.Printf("[DEBUG] alicloud_zones - Zones found: %#v", zoneTypes)
	return zonesDescriptionAttributes(d, zoneTypes)
}

// check array constraints str
func constraints(arr interface{}, v string) bool {
	arrs := reflect.ValueOf(arr)
	len := arrs.Len()
	for i := 0; i < len; i++ {
		if arrs.Index(i).String() == v {
			return true
		}
	}
	return false
}

func zonesDescriptionAttributes(d *schema.ResourceData, types []ecs.ZoneType) error {
	var ids []string
	var s []map[string]interface{}
	for _, t := range types {
		mapping := map[string]interface{}{
			"id":                          t.ZoneId,
			"local_name":                  t.LocalName,
			"available_instance_types":    t.AvailableInstanceTypes.InstanceTypes,
			"available_resource_creation": t.AvailableResourceCreation.ResourceTypes,
			"available_disk_categories":   t.AvailableDiskCategories.DiskCategories,
		}

		log.Printf("[DEBUG] alicloud_zones - adding zone mapping: %v", mapping)
		ids = append(ids, t.ZoneId)
		s = append(s, mapping)
	}

	d.SetId(dataResourceIdHash(ids))
	if err := d.Set("zones", s); err != nil {
		return err
	}
	return nil
}
