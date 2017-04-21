package alicloud

import (
	"fmt"
	"strings"

	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"time"
)

func resourceAliyunEipAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunEipAssociationCreate,
		Read:   resourceAliyunEipAssociationRead,
		Delete: resourceAliyunEipAssociationDelete,

		Schema: map[string]*schema.Schema{
			"allocation_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"instance_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAliyunEipAssociationCreate(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AliyunClient).ecsconn

	allocationId := d.Get("allocation_id").(string)
	instanceId := d.Get("instance_id").(string)

	if err := conn.AssociateEipAddress(allocationId, instanceId); err != nil {
		return err
	}

	d.SetId(allocationId + ":" + instanceId)

	return resourceAliyunEipAssociationRead(d, meta)
}

func resourceAliyunEipAssociationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)

	allocationId, instanceId, err := getAllocationIdAndInstanceId(d, meta)
	if err != nil {
		return err
	}

	eip, err := client.DescribeEipAddress(allocationId)

	if err != nil {
		if notFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error Describe Eip Attribute: %#v", err)
	}

	if eip.InstanceId != instanceId {
		d.SetId("")
		return nil
	}

	d.Set("instance_id", eip.InstanceId)
	d.Set("allocation_id", allocationId)
	return nil
}

func resourceAliyunEipAssociationDelete(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AliyunClient).ecsconn

	allocationId, instanceId, err := getAllocationIdAndInstanceId(d, meta)
	if err != nil {
		return err
	}

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		err := conn.UnassociateEipAddress(allocationId, instanceId)

		if err != nil {
			e, _ := err.(*common.Error)
			errCode := e.ErrorResponse.Code
			if errCode == InstanceIncorrectStatus || errCode == HaVipIncorrectStatus {
				return resource.RetryableError(fmt.Errorf("Eip in use - trying again while make it unassociated."))
			}
		}

		args := &ecs.DescribeEipAddressesArgs{
			RegionId:     getRegion(d, meta),
			AllocationId: allocationId,
		}

		eips, _, descErr := conn.DescribeEipAddresses(args)

		if descErr != nil {
			return resource.NonRetryableError(descErr)
		} else if eips == nil || len(eips) < 1 {
			return nil
		}
		for _, eip := range eips {
			if eip.Status != ecs.EipStatusAvailable {
				return resource.RetryableError(fmt.Errorf("Eip in use - trying again while make it unassociated."))
			}
		}

		return nil
	})
}

func getAllocationIdAndInstanceId(d *schema.ResourceData, meta interface{}) (string, string, error) {
	parts := strings.Split(d.Id(), ":")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid resource id")
	}
	return parts[0], parts[1], nil
}
