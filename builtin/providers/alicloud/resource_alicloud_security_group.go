package alicloud

import (
	"fmt"

	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"time"
)

func resourceAliyunSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunSecurityGroupCreate,
		Read:   resourceAliyunSecurityGroupRead,
		Update: resourceAliyunSecurityGroupUpdate,
		Delete: resourceAliyunSecurityGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateSecurityGroupName,
			},

			"description": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateSecurityGroupDescription,
			},

			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAliyunSecurityGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	args, err := buildAliyunSecurityGroupArgs(d, meta)
	if err != nil {
		return err
	}

	securityGroupID, err := conn.CreateSecurityGroup(args)
	if err != nil {
		return err
	}

	d.SetId(securityGroupID)

	return resourceAliyunSecurityGroupRead(d, meta)
}

func resourceAliyunSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	args := &ecs.DescribeSecurityGroupAttributeArgs{
		SecurityGroupId: d.Id(),
		RegionId:        getRegion(d, meta),
	}

	sg, err := conn.DescribeSecurityGroupAttribute(args)
	if err != nil {
		if notFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error DescribeSecurityGroupAttribute: %#v", err)
	}

	if sg == nil {
		d.SetId("")
		return nil
	}

	d.Set("name", sg.SecurityGroupName)
	d.Set("description", sg.Description)

	return nil
}

func resourceAliyunSecurityGroupUpdate(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AliyunClient).ecsconn

	d.Partial(true)
	attributeUpdate := false
	args := &ecs.ModifySecurityGroupAttributeArgs{
		SecurityGroupId: d.Id(),
		RegionId:        getRegion(d, meta),
	}

	if d.HasChange("name") {
		d.SetPartial("name")
		args.SecurityGroupName = d.Get("name").(string)

		attributeUpdate = true
	}

	if d.HasChange("description") {
		d.SetPartial("description")
		args.Description = d.Get("description").(string)

		attributeUpdate = true
	}
	if attributeUpdate {
		if err := conn.ModifySecurityGroupAttribute(args); err != nil {
			return err
		}
	}

	return nil
}

func resourceAliyunSecurityGroupDelete(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AliyunClient).ecsconn

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		err := conn.DeleteSecurityGroup(getRegion(d, meta), d.Id())

		if err != nil {
			e, _ := err.(*common.Error)
			if e.ErrorResponse.Code == SgDependencyViolation {
				return resource.RetryableError(fmt.Errorf("Security group in use - trying again while it is deleted."))
			}
		}

		sg, err := conn.DescribeSecurityGroupAttribute(&ecs.DescribeSecurityGroupAttributeArgs{
			RegionId:        getRegion(d, meta),
			SecurityGroupId: d.Id(),
		})

		if err != nil {
			e, _ := err.(*common.Error)
			if e.ErrorResponse.Code == InvalidSecurityGroupIdNotFound {
				return nil
			}
			return resource.NonRetryableError(err)
		} else if sg == nil {
			return nil
		}

		return resource.RetryableError(fmt.Errorf("Security group in use - trying again while it is deleted."))
	})

}

func buildAliyunSecurityGroupArgs(d *schema.ResourceData, meta interface{}) (*ecs.CreateSecurityGroupArgs, error) {

	args := &ecs.CreateSecurityGroupArgs{
		RegionId: getRegion(d, meta),
	}

	if v := d.Get("name").(string); v != "" {
		args.SecurityGroupName = v
	}

	if v := d.Get("description").(string); v != "" {
		args.Description = v
	}

	if v := d.Get("vpc_id").(string); v != "" {
		args.VpcId = v
	}

	return args, nil
}
