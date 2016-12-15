package alicloud

import (
	"strconv"

	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/resource"
	"time"
	"log"
	"fmt"
)

func resourceAliyunEip() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunEipCreate,
		Read:   resourceAliyunEipRead,
		Update: resourceAliyunEipUpdate,
		Delete: resourceAliyunEipDelete,

		Schema: map[string]*schema.Schema{
			"band_width": &schema.Schema{
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
		return err
	}

	bandwidth, _ := strconv.Atoi(eip.Bandwidth)
	d.Set("band_width", bandwidth)
	d.Set("internet_charge_type", eip.InternetChargeType)
	d.Set("ip_address", eip.IpAddress)
	d.Set("status", eip.Status)

	if eip.InstanceId != "" {
		d.Set("instance", eip.InstanceId)
	} else {
		d.Set("instance", "")
	}

	return nil
}

func resourceAliyunEipUpdate(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AliyunClient).ecsconn

	d.Partial(true)

	if d.HasChange("band_width") {
		err := conn.ModifyEipAddressAttribute(d.Id(), d.Get("band_width").(int))
		if err != nil {
			return err
		}

		d.SetPartial("band_width")
	}

	d.Partial(false)

	return nil
}

func resourceAliyunEipDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	return resource.Retry(5 * time.Minute, func() *resource.RetryError {
		err := conn.ReleaseEipAddress(d.Id())
		if err == nil {
			return nil
		}

		e, _ := err.(*common.Error)
		if e.ErrorResponse.Code == "IncorrectEipStatus" {
			return resource.RetryableError(fmt.Errorf("EIP in use - trying again while it is deleted."))
		}

		log.Println("[ERROR] Delete EIP failed.")
		return resource.NonRetryableError(err)
	})
}

func buildAliyunEipArgs(d *schema.ResourceData, meta interface{}) (*ecs.AllocateEipAddressArgs, error) {

	args := &ecs.AllocateEipAddressArgs{
		RegionId:           getRegion(d, meta),
		Bandwidth:          d.Get("band_width").(int),
		InternetChargeType: common.InternetChargeType(d.Get("internet_charge_type").(string)),
	}

	return args, nil
}
