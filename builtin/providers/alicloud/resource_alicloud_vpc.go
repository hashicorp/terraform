package alicloud

import (
	"fmt"
	"strings"

	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/resource"
	"time"
)

func resourceAliyunVpc() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunVpcCreate,
		Read:   resourceAliyunVpcRead,
		Update: resourceAliyunVpcUpdate,
		Delete: resourceAliyunVpcDelete,

		Schema: map[string]*schema.Schema{
			"cidr_block": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateCIDRNetworkAddress,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) < 2 || len(value) > 128 {
						errors = append(errors, fmt.Errorf("%q cannot be longer than 128 characters", k))
					}

					if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
						errors = append(errors, fmt.Errorf("%s cannot starts with http:// or https://", k))
					}

					return
				},
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) < 2 || len(value) > 256 {
						errors = append(errors, fmt.Errorf("%q cannot be longer than 256 characters", k))

					}
					return
				},
			},
			"router_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAliyunVpcCreate(d *schema.ResourceData, meta interface{}) error {

	args, err := buildAliyunVpcArgs(d, meta)
	if err != nil {
		return err
	}

	ec2conn := meta.(*AliyunClient).ecsconn

	vpc, err := ec2conn.CreateVpc(args)
	if err != nil {
		return err
	}

	d.SetId(vpc.VpcId)

	err = ec2conn.WaitForVpcAvailable(args.RegionId, vpc.VpcId, 60)
	if err != nil {
		return fmt.Errorf("Timeout when WaitForVpcAvailable")
	}

	return resourceAliyunVpcRead(d, meta)
}

func resourceAliyunVpcRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*AliyunClient)

	vpc, err := client.DescribeVpc(d.Id())
	if err != nil {
		return err
	}

	if vpc == nil {
		d.SetId("")
		return nil
	}

	d.Set("cidr_block", vpc.CidrBlock)
	d.Set("name", vpc.VpcName)
	d.Set("description", vpc.Description)
	d.Set("router_id", vpc.VRouterId)

	return nil
}

func resourceAliyunVpcUpdate(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AliyunClient).ecsconn

	d.Partial(true)

	vpcid := d.Id()

	if d.HasChange("name") {
		val := d.Get("name").(string)
		args := &ecs.ModifyVpcAttributeArgs{
			VpcId:   vpcid,
			VpcName: val,
		}

		if err := conn.ModifyVpcAttribute(args); err != nil {
			return err
		}

		d.SetPartial("name")
	}

	if d.HasChange("description") {
		val := d.Get("description").(string)
		args := &ecs.ModifyVpcAttributeArgs{
			VpcId:       vpcid,
			Description: val,
		}

		if err := conn.ModifyVpcAttribute(args); err != nil {
			return err
		}

		d.SetPartial("description")
	}

	d.Partial(false)

	return nil
}

func resourceAliyunVpcDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	return resource.Retry(5 * time.Minute, func() *resource.RetryError {
		err := conn.DeleteVpc(d.Id())
		if err == nil {
			return nil
		}

		return resource.RetryableError(fmt.Errorf("Vpc in use - trying again while it is deleted."))
	})
}

func buildAliyunVpcArgs(d *schema.ResourceData, meta interface{}) (*ecs.CreateVpcArgs, error) {
	args := &ecs.CreateVpcArgs{
		RegionId:  getRegion(d, meta),
		CidrBlock: d.Get("cidr_block").(string),
	}

	if v := d.Get("name").(string); v != "" {
		args.VpcName = v
	}

	if v := d.Get("description").(string); v != "" {
		args.Description = v
	}

	return args, nil
}
