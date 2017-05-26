package alicloud

import (
	"fmt"

	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"time"
)

func resourceAliyunDisk() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunDiskCreate,
		Read:   resourceAliyunDiskRead,
		Update: resourceAliyunDiskUpdate,
		Delete: resourceAliyunDiskDelete,

		Schema: map[string]*schema.Schema{
			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateDiskName,
			},

			"description": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateDiskDescription,
			},

			"category": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateDiskCategory,
				Default:      "cloud",
			},

			"size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"snapshot_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAliyunDiskCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)

	conn := client.ecsconn

	availabilityZone, err := client.DescribeZone(d.Get("availability_zone").(string))
	if err != nil {
		return err
	}

	args := &ecs.CreateDiskArgs{
		RegionId: getRegion(d, meta),
		ZoneId:   availabilityZone.ZoneId,
	}

	if v, ok := d.GetOk("category"); ok && v.(string) != "" {
		category := ecs.DiskCategory(v.(string))
		if err := client.DiskAvailable(availabilityZone, category); err != nil {
			return err
		}
		args.DiskCategory = category
	}

	if v, ok := d.GetOk("size"); ok {
		size := v.(int)
		if args.DiskCategory == ecs.DiskCategoryCloud && (size < 5 || size > 2000) {
			return fmt.Errorf("the size of cloud disk must between 5 to 2000")
		}

		if (args.DiskCategory == ecs.DiskCategoryCloudEfficiency ||
			args.DiskCategory == ecs.DiskCategoryCloudSSD) && (size < 20 || size > 32768) {
			return fmt.Errorf("the size of %s disk must between 20 to 32768", args.DiskCategory)
		}
		args.Size = size

		d.Set("size", args.Size)
	}

	if v, ok := d.GetOk("snapshot_id"); ok && v.(string) != "" {
		args.SnapshotId = v.(string)
	}

	if args.Size <= 0 && args.SnapshotId == "" {
		return fmt.Errorf("One of size or snapshot_id is required when specifying an ECS disk.")
	}

	if v, ok := d.GetOk("name"); ok && v.(string) != "" {
		args.DiskName = v.(string)
	}

	if v, ok := d.GetOk("description"); ok && v.(string) != "" {
		args.Description = v.(string)
	}

	diskID, err := conn.CreateDisk(args)
	if err != nil {
		return fmt.Errorf("CreateDisk got a error: %#v", err)
	}

	d.SetId(diskID)

	return resourceAliyunDiskUpdate(d, meta)
}

func resourceAliyunDiskRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	disks, _, err := conn.DescribeDisks(&ecs.DescribeDisksArgs{
		RegionId: getRegion(d, meta),
		DiskIds:  []string{d.Id()},
	})

	if err != nil {
		if notFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error DescribeDiskAttribute: %#v", err)
	}

	log.Printf("[DEBUG] DescribeDiskAttribute for instance: %#v", disks)

	if disks == nil || len(disks) <= 0 {
		return fmt.Errorf("No disks found.")
	}

	disk := disks[0]
	d.Set("availability_zone", disk.ZoneId)
	d.Set("category", disk.Category)
	d.Set("size", disk.Size)
	d.Set("status", disk.Status)
	d.Set("name", disk.DiskName)
	d.Set("description", disk.Description)
	d.Set("snapshot_id", disk.SourceSnapshotId)

	tags, _, err := conn.DescribeTags(&ecs.DescribeTagsArgs{
		RegionId:     getRegion(d, meta),
		ResourceType: ecs.TagResourceDisk,
		ResourceId:   d.Id(),
	})

	if err != nil {
		log.Printf("[DEBUG] DescribeTags for disk got error: %#v", err)
	}

	d.Set("tags", tagsToMap(tags))

	return nil
}

func resourceAliyunDiskUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	conn := client.ecsconn

	d.Partial(true)

	if err := setTags(client, ecs.TagResourceDisk, d); err != nil {
		log.Printf("[DEBUG] Set tags for instance got error: %#v", err)
		return fmt.Errorf("Set tags for instance got error: %#v", err)
	} else {
		d.SetPartial("tags")
	}
	attributeUpdate := false
	args := &ecs.ModifyDiskAttributeArgs{
		DiskId: d.Id(),
	}

	if d.HasChange("name") {
		d.SetPartial("name")
		val := d.Get("name").(string)
		args.DiskName = val

		attributeUpdate = true
	}

	if d.HasChange("description") {
		d.SetPartial("description")
		val := d.Get("description").(string)
		args.Description = val

		attributeUpdate = true
	}
	if attributeUpdate {
		if err := conn.ModifyDiskAttribute(args); err != nil {
			return err
		}
	}

	d.Partial(false)

	return resourceAliyunDiskRead(d, meta)
}

func resourceAliyunDiskDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		err := conn.DeleteDisk(d.Id())
		if err != nil {
			e, _ := err.(*common.Error)
			if e.ErrorResponse.Code == DiskIncorrectStatus || e.ErrorResponse.Code == DiskCreatingSnapshot {
				return resource.RetryableError(fmt.Errorf("Disk in use - trying again while it is deleted."))
			}
		}

		disks, _, descErr := conn.DescribeDisks(&ecs.DescribeDisksArgs{
			RegionId: getRegion(d, meta),
			DiskIds:  []string{d.Id()},
		})

		if descErr != nil {
			log.Printf("[ERROR] Delete disk is failed.")
			return resource.NonRetryableError(descErr)
		}
		if disks == nil || len(disks) < 1 {
			return nil
		}

		return resource.RetryableError(fmt.Errorf("Disk in use - trying again while it is deleted."))
	})
}
