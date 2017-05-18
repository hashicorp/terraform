package alicloud

import (
	"strconv"

	"fmt"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"time"
)

func resourceAliyunEip() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunEipCreate,
		Read:   resourceAliyunEipRead,
		Update: resourceAliyunEipUpdate,
		Delete: resourceAliyunEipDelete,

		Schema: map[string]*schema.Schema{
			"bandwidth": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  5,
			},
			"internet_charge_type": &schema.Schema{
				Type:         schema.TypeString,
				Default:      "PayByBandwidth",
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateInternetChargeType,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"instance": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAliyunEipCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	args, err := buildAliyunEipArgs(d, meta)
	if err != nil {
		return err
	}

	_, allocationID, err := conn.AllocateEipAddress(args)
	if err != nil {
		return err
	}

	d.SetId(allocationID)

	return resourceAliyunEipRead(d, meta)
}

func resourceAliyunEipRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)

	eip, err := client.DescribeEipAddress(d.Id())
	if err != nil {
		if notFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error Describe Eip Attribute: %#v", err)
	}

	if eip.InstanceId != "" {
		d.Set("instance", eip.InstanceId)
	} else {
		d.Set("instance", "")
		return nil
	}

	bandwidth, _ := strconv.Atoi(eip.Bandwidth)
	d.Set("bandwidth", bandwidth)
	d.Set("internet_charge_type", eip.InternetChargeType)
	d.Set("ip_address", eip.IpAddress)
	d.Set("status", eip.Status)

	return nil
}

func resourceAliyunEipUpdate(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AliyunClient).ecsconn

	d.Partial(true)

	if d.HasChange("bandwidth") {
		err := conn.ModifyEipAddressAttribute(d.Id(), d.Get("bandwidth").(int))
		if err != nil {
			return err
		}

		d.SetPartial("bandwidth")
	}

	d.Partial(false)

	return nil
}

func resourceAliyunEipDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		err := conn.ReleaseEipAddress(d.Id())

		if err != nil {
			e, _ := err.(*common.Error)
			if e.ErrorResponse.Code == EipIncorrectStatus {
				return resource.RetryableError(fmt.Errorf("EIP in use - trying again while it is deleted."))
			}
		}

		args := &ecs.DescribeEipAddressesArgs{
			RegionId:     getRegion(d, meta),
			AllocationId: d.Id(),
		}

		eips, _, descErr := conn.DescribeEipAddresses(args)
		if descErr != nil {
			return resource.NonRetryableError(descErr)
		} else if eips == nil || len(eips) < 1 {
			return nil
		}
		return resource.RetryableError(fmt.Errorf("EIP in use - trying again while it is deleted."))
	})
}

func buildAliyunEipArgs(d *schema.ResourceData, meta interface{}) (*ecs.AllocateEipAddressArgs, error) {

	args := &ecs.AllocateEipAddressArgs{
		RegionId:           getRegion(d, meta),
		Bandwidth:          d.Get("bandwidth").(int),
		InternetChargeType: common.InternetChargeType(d.Get("internet_charge_type").(string)),
	}

	return args, nil
}
