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

func resourceAliyunSubnet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunSwitchCreate,
		Read:   resourceAliyunSwitchRead,
		Update: resourceAliyunSwitchUpdate,
		Delete: resourceAliyunSwitchDelete,

		Schema: map[string]*schema.Schema{
			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"cidr_block": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateSwitchCIDRNetworkAddress,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAliyunSwitchCreate(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AliyunClient).ecsconn

	args, err := buildAliyunSwitchArgs(d, meta)
	if err != nil {
		return err
	}

	vswitchID, err := conn.CreateVSwitch(args)
	if err != nil {
		return fmt.Errorf("Create subnet got a error :%s", err)
	}

	d.SetId(vswitchID)

	err = conn.WaitForVSwitchAvailable(args.VpcId, vswitchID, 60)
	if err != nil {
		return fmt.Errorf("WaitForVSwitchAvailable got a error: %s", err)
	}

	return resourceAliyunSwitchRead(d, meta)
}

func resourceAliyunSwitchRead(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AliyunClient).ecsconn

	args := &ecs.DescribeVSwitchesArgs{
		VpcId:     d.Get("vpc_id").(string),
		VSwitchId: d.Id(),
	}

	vswitches, _, err := conn.DescribeVSwitches(args)

	if err != nil {
		if notFoundError(err) {
			d.SetId("")
			return nil
		}
		return err
	}

	if len(vswitches) == 0 {
		d.SetId("")
		return nil
	}

	vswitch := vswitches[0]

	d.Set("availability_zone", vswitch.ZoneId)
	d.Set("vpc_id", vswitch.VpcId)
	d.Set("cidr_block", vswitch.CidrBlock)
	d.Set("name", vswitch.VSwitchName)
	d.Set("description", vswitch.Description)

	return nil
}

func resourceAliyunSwitchUpdate(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AliyunClient).ecsconn

	d.Partial(true)

	if d.HasChange("name") {
		val := d.Get("name").(string)
		args := &ecs.ModifyVSwitchAttributeArgs{
			VSwitchId:   d.Id(),
			VSwitchName: val,
		}

		if err := conn.ModifyVSwitchAttribute(args); err != nil {
			return err
		}

		d.SetPartial("name")
	}

	if d.HasChange("description") {
		val := d.Get("description").(string)
		args := &ecs.ModifyVSwitchAttributeArgs{
			VSwitchId:   d.Id(),
			Description: val,
		}

		if err := conn.ModifyVSwitchAttribute(args); err != nil {
			return err
		}

		d.SetPartial("description")
	}

	d.Partial(false)

	return nil
}

func resourceAliyunSwitchDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	return resource.Retry(5 * time.Minute, func() *resource.RetryError {
		err := conn.DeleteVSwitch(d.Id())

		if err == nil {
			return nil
		}

		e, _ := err.(*common.Error)
		if e.ErrorResponse.Code == "InvalidRegionId.NotFound" {
			log.Printf("[ERROR] Delete Switch is failed.")
			return resource.NonRetryableError(err)
		}

		return resource.RetryableError(fmt.Errorf("Switch in use. -- trying again while it is deleted."))
	})
}

func buildAliyunSwitchArgs(d *schema.ResourceData, meta interface{}) (*ecs.CreateVSwitchArgs, error) {

	client := meta.(*AliyunClient)

	vpcID := d.Get("vpc_id").(string)

	vpc, err := client.DescribeVpc(vpcID)
	if err != nil {
		return nil, err
	}

	if vpc == nil {
		return nil, fmt.Errorf("vpc_id not found")
	}

	zoneID := d.Get("availability_zone").(string)

	zone, err := client.DescribeZone(zoneID)
	if err != nil {
		return nil, err
	}

	err = client.ResourceAvailable(zone, ecs.ResourceTypeVSwitch)
	if err != nil {
		return nil, err
	}

	cidrBlock := d.Get("cidr_block").(string)

	args := &ecs.CreateVSwitchArgs{
		VpcId:     vpcID,
		ZoneId:    zoneID,
		CidrBlock: cidrBlock,
	}

	if v, ok := d.GetOk("name"); ok && v != "" {
		args.VSwitchName = v.(string)
	}

	if v, ok := d.GetOk("description"); ok && v != "" {
		args.Description = v.(string)
	}

	return args, nil
}
