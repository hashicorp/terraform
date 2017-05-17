package ignition

import (
	"github.com/coreos/ignition/config/types"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDisk() *schema.Resource {
	return &schema.Resource{
		Exists: resourceDiskExists,
		Read:   resourceDiskRead,
		Schema: map[string]*schema.Schema{
			"device": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"wipe_table": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"partition": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"label": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"number": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"start": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"type_guid": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func resourceDiskRead(d *schema.ResourceData, meta interface{}) error {
	id, err := buildDisk(d, globalCache)
	if err != nil {
		return err
	}

	d.SetId(id)
	return nil
}

func resourceDiskExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	id, err := buildDisk(d, globalCache)
	if err != nil {
		return false, err
	}

	return id == d.Id(), nil
}

func buildDisk(d *schema.ResourceData, c *cache) (string, error) {
	var partitions []types.Partition
	for _, raw := range d.Get("partition").([]interface{}) {
		v := raw.(map[string]interface{})

		partitions = append(partitions, types.Partition{
			Label:    types.PartitionLabel(v["label"].(string)),
			Number:   v["number"].(int),
			Size:     types.PartitionDimension(v["size"].(int)),
			Start:    types.PartitionDimension(v["start"].(int)),
			TypeGUID: types.PartitionTypeGUID(v["type_guid"].(string)),
		})
	}

	return c.addDisk(&types.Disk{
		Device:     types.Path(d.Get("device").(string)),
		WipeTable:  d.Get("wipe_table").(bool),
		Partitions: partitions,
	}), nil
}
