package alicloud

import (
	"fmt"
	"strings"

	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"time"
)

func resourceAliyunDiskAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunDiskAttachmentCreate,
		Read:   resourceAliyunDiskAttachmentRead,
		Delete: resourceAliyunDiskAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"instance_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"disk_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"device_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAliyunDiskAttachmentCreate(d *schema.ResourceData, meta interface{}) error {

	err := diskAttachment(d, meta)
	if err != nil {
		return err
	}

	d.SetId(d.Get("disk_id").(string) + ":" + d.Get("instance_id").(string))

	return resourceAliyunDiskAttachmentRead(d, meta)
}

func resourceAliyunDiskAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	diskId, instanceId, err := getDiskIDAndInstanceID(d, meta)
	if err != nil {
		return err
	}

	conn := meta.(*AliyunClient).ecsconn
	disks, _, err := conn.DescribeDisks(&ecs.DescribeDisksArgs{
		RegionId:   getRegion(d, meta),
		InstanceId: instanceId,
		DiskIds:    []string{diskId},
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
		return fmt.Errorf("No Disks Found.")
	}

	disk := disks[0]
	d.Set("instance_id", disk.InstanceId)
	d.Set("disk_id", disk.DiskId)
	d.Set("device_name", disk.Device)

	return nil
}

func resourceAliyunDiskAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn
	diskID, instanceID, err := getDiskIDAndInstanceID(d, meta)
	if err != nil {
		return err
	}

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		err := conn.DetachDisk(instanceID, diskID)
		if err != nil {
			e, _ := err.(*common.Error)
			if e.ErrorResponse.Code == DiskIncorrectStatus || e.ErrorResponse.Code == InstanceLockedForSecurity {
				return resource.RetryableError(fmt.Errorf("Disk in use - trying again while it detaches"))
			}
		}

		disks, _, descErr := conn.DescribeDisks(&ecs.DescribeDisksArgs{
			RegionId: getRegion(d, meta),
			DiskIds:  []string{diskID},
		})

		if descErr != nil {
			log.Printf("[ERROR] Disk %s is not detached.", diskID)
			return resource.NonRetryableError(err)
		}

		for _, disk := range disks {
			if disk.Status != ecs.DiskStatusAvailable {
				return resource.RetryableError(fmt.Errorf("Disk in use - trying again while it is deleted."))
			}
		}
		return nil
	})
}

func getDiskIDAndInstanceID(d *schema.ResourceData, meta interface{}) (string, string, error) {
	parts := strings.Split(d.Id(), ":")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid resource id")
	}
	return parts[0], parts[1], nil
}
func diskAttachment(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	diskID := d.Get("disk_id").(string)
	instanceID := d.Get("instance_id").(string)

	deviceName := d.Get("device_name").(string)

	args := &ecs.AttachDiskArgs{
		InstanceId: instanceID,
		DiskId:     diskID,
		Device:     deviceName,
	}

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		err := conn.AttachDisk(args)
		log.Printf("error : %s", err)

		if err != nil {
			e, _ := err.(*common.Error)
			if e.ErrorResponse.Code == DiskIncorrectStatus || e.ErrorResponse.Code == InstanceIncorrectStatus {
				return resource.RetryableError(fmt.Errorf("Disk or Instance status is incorrect - trying again while it attaches"))
			}
			return resource.NonRetryableError(err)
		}

		disks, _, descErr := conn.DescribeDisks(&ecs.DescribeDisksArgs{
			RegionId:   getRegion(d, meta),
			InstanceId: instanceID,
			DiskIds:    []string{diskID},
		})

		if descErr != nil {
			log.Printf("[ERROR] Disk %s is not attached.", diskID)
			return resource.NonRetryableError(err)
		}

		if disks == nil || len(disks) <= 0 {
			return resource.RetryableError(fmt.Errorf("Disk in attaching - trying again while it is attached."))
		}

		return nil

	})
}
