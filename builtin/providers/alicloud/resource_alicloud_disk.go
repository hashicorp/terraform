package alicloud

import (
	"fmt"

	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/resource"
	"time"
	"github.com/denverdino/aliyungo/common"
	"log"
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
				Type:     schema.TypeString,
				Optional: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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
	} else {
		if args.DiskCategory == ecs.DiskCategoryCloud {
			args.Size = 5
		}
		if args.DiskCategory == ecs.DiskCategoryCloudEfficiency ||
			args.DiskCategory == ecs.DiskCategoryCloudSSD {
			args.Size = 20
		}

		d.Set("size", args.Size)
	}

	if v, ok := d.GetOk("snapshot_id"); ok && v.(string) != "" {
		args.SnapshotId = v.(string)
	}

	if v, ok := d.GetOk("name"); ok && v.(string) != "" {
		args.DiskName = v.(string)
	}

	if v, ok := d.GetOk("name"); ok && v.(string) != "" {
		args.DiskName = v.(string)
	}

	diskID, err := conn.CreateDisk(args)
	if err != nil {
		return fmt.Errorf("CreateDisk got a error: %s", err)
	}

	d.SetId(diskID)
	d.SetPartial("name")
	d.SetPartial("availability_zone")
	d.SetPartial("description")
	d.SetPartial("size")
	d.SetPartial("category")
	d.SetPartial("snapshot_id")

	return resourceAliyunDiskRead(d, meta)
}

func resourceAliyunDiskRead(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func resourceAliyunDiskUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	conn := client.ecsconn

	d.Partial(true)

	if err := setTags(client, ecs.TagResourceDisk, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	if d.HasChange("name") {
		val := d.Get("description").(string)
		args := &ecs.ModifyDiskAttributeArgs{
			DiskId:   d.Id(),
			DiskName: val,
		}

		if err := conn.ModifyDiskAttribute(args); err != nil {
			return err
		}

		d.SetPartial("name")
	}

	if d.HasChange("description") {
		val := d.Get("description").(string)
		args := &ecs.ModifyDiskAttributeArgs{
			DiskId:      d.Id(),
			Description: val,
		}

		if err := conn.ModifyDiskAttribute(args); err != nil {
			return err
		}

		d.SetPartial("description")
	}

	return resourceAliyunDiskRead(d, meta)
}

func resourceAliyunDiskDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	return resource.Retry(5 * time.Minute, func() *resource.RetryError {
		err := conn.DeleteDisk(d.Id())
		if err == nil {
			return nil
		}

		e, _ := err.(*common.Error)
		if e.ErrorResponse.Code == "IncorrectDiskStatus" || e.ErrorResponse.Code == "DiskCreatingSnapshot" {
			return resource.RetryableError(fmt.Errorf("Disk in use - trying again while it is deleted."))
		}

		log.Printf("[ERROR] Delete disk is failed.")
		return resource.NonRetryableError(err)
	})
}
