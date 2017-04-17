package alicloud

import (
	"fmt"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
)

func dataSourceAlicloudInstanceTypes() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAlicloudInstanceTypesRead,

		Schema: map[string]*schema.Schema{
			"instance_type_family": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"cpu_core_count": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"memory_size": {
				Type:     schema.TypeFloat,
				Optional: true,
				ForceNew: true,
			},
			// Computed values.
			"instance_types": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"cpu_core_count": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"memory_size": {
							Type:     schema.TypeFloat,
							Computed: true,
						},
						"family": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAlicloudInstanceTypesRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	cpu, _ := d.Get("cpu_core_count").(int)
	mem, _ := d.Get("memory_size").(float64)

	args, err := buildAliyunAlicloudInstanceTypesArgs(d, meta)

	if err != nil {
		return err
	}

	resp, err := conn.DescribeInstanceTypesNew(args)
	if err != nil {
		return err
	}

	var instanceTypes []ecs.InstanceTypeItemType
	for _, types := range resp {
		if cpu > 0 && types.CpuCoreCount != cpu {
			continue
		}

		if mem > 0 && types.MemorySize != mem {
			continue
		}
		instanceTypes = append(instanceTypes, types)
	}

	if len(instanceTypes) < 1 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	log.Printf("[DEBUG] alicloud_instance_type - Types found: %#v", instanceTypes)
	return instanceTypesDescriptionAttributes(d, instanceTypes)
}

func instanceTypesDescriptionAttributes(d *schema.ResourceData, types []ecs.InstanceTypeItemType) error {
	var ids []string
	var s []map[string]interface{}
	for _, t := range types {
		mapping := map[string]interface{}{
			"id":             t.InstanceTypeId,
			"cpu_core_count": t.CpuCoreCount,
			"memory_size":    t.MemorySize,
			"family":         t.InstanceTypeFamily,
		}

		log.Printf("[DEBUG] alicloud_instance_type - adding type mapping: %v", mapping)
		ids = append(ids, t.InstanceTypeId)
		s = append(s, mapping)
	}

	d.SetId(dataResourceIdHash(ids))
	if err := d.Set("instance_types", s); err != nil {
		return err
	}
	return nil
}

func buildAliyunAlicloudInstanceTypesArgs(d *schema.ResourceData, meta interface{}) (*ecs.DescribeInstanceTypesArgs, error) {
	args := &ecs.DescribeInstanceTypesArgs{}

	if v := d.Get("instance_type_family").(string); v != "" {
		args.InstanceTypeFamily = v
	}

	return args, nil
}
