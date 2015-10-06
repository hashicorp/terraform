package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEfsMountTarget() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEfsMountTargetCreate,
		Read:   resourceAwsEfsMountTargetRead,
		Update: resourceAwsEfsMountTargetUpdate,
		Delete: resourceAwsEfsMountTargetDelete,

		Schema: map[string]*schema.Schema{
			"file_system_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},

			"security_groups": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				Computed: true,
				Optional: true,
			},

			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network_interface_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsEfsMountTargetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	input := efs.CreateMountTargetInput{
		FileSystemId: aws.String(d.Get("file_system_id").(string)),
		SubnetId:     aws.String(d.Get("subnet_id").(string)),
	}

	if v, ok := d.GetOk("ip_address"); ok {
		input.IpAddress = aws.String(v.(string))
	}
	if v, ok := d.GetOk("security_groups"); ok {
		input.SecurityGroups = expandStringList(v.(*schema.Set).List())
	}

	log.Printf("[DEBUG] Creating EFS mount target: %#v", input)

	mt, err := conn.CreateMountTarget(&input)
	if err != nil {
		return err
	}

	d.SetId(*mt.MountTargetId)

	stateConf := &resource.StateChangeConf{
		Pending: []string{"creating"},
		Target:  "available",
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.DescribeMountTargets(&efs.DescribeMountTargetsInput{
				MountTargetId: aws.String(d.Id()),
			})
			if err != nil {
				return nil, "error", err
			}

			if len(resp.MountTargets) < 1 {
				return nil, "error", fmt.Errorf("EFS mount target %q not found", d.Id())
			}

			mt := resp.MountTargets[0]

			log.Printf("[DEBUG] Current status of %q: %q", *mt.MountTargetId, *mt.LifeCycleState)
			return mt, *mt.LifeCycleState, nil
		},
		Timeout:    10 * time.Minute,
		Delay:      2 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for EFS mount target (%s) to create: %s", d.Id(), err)
	}

	log.Printf("[DEBUG] EFS mount target created: %s", *mt.MountTargetId)

	return resourceAwsEfsMountTargetRead(d, meta)
}

func resourceAwsEfsMountTargetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	if d.HasChange("security_groups") {
		input := efs.ModifyMountTargetSecurityGroupsInput{
			MountTargetId:  aws.String(d.Id()),
			SecurityGroups: expandStringList(d.Get("security_groups").(*schema.Set).List()),
		}
		_, err := conn.ModifyMountTargetSecurityGroups(&input)
		if err != nil {
			return err
		}
	}

	return resourceAwsEfsMountTargetRead(d, meta)
}

func resourceAwsEfsMountTargetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn
	resp, err := conn.DescribeMountTargets(&efs.DescribeMountTargetsInput{
		MountTargetId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	if len(resp.MountTargets) < 1 {
		return fmt.Errorf("EFS mount target %q not found", d.Id())
	}

	mt := resp.MountTargets[0]

	log.Printf("[DEBUG] Found EFS mount target: %#v", mt)

	d.SetId(*mt.MountTargetId)
	d.Set("file_system_id", *mt.FileSystemId)
	d.Set("ip_address", *mt.IpAddress)
	d.Set("subnet_id", *mt.SubnetId)
	d.Set("network_interface_id", *mt.NetworkInterfaceId)

	sgResp, err := conn.DescribeMountTargetSecurityGroups(&efs.DescribeMountTargetSecurityGroupsInput{
		MountTargetId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	d.Set("security_groups", schema.NewSet(schema.HashString, flattenStringList(sgResp.SecurityGroups)))

	return nil
}

func resourceAwsEfsMountTargetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	log.Printf("[DEBUG] Deleting EFS mount target %q", d.Id())
	_, err := conn.DeleteMountTarget(&efs.DeleteMountTargetInput{
		MountTargetId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{"available", "deleting", "deleted"},
		Target:  "",
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.DescribeMountTargets(&efs.DescribeMountTargetsInput{
				MountTargetId: aws.String(d.Id()),
			})
			if err != nil {
				awsErr, ok := err.(awserr.Error)
				if !ok {
					return nil, "error", err
				}

				if awsErr.Code() == "MountTargetNotFound" {
					return nil, "", nil
				}

				return nil, "error", awsErr
			}

			if len(resp.MountTargets) < 1 {
				return nil, "", nil
			}

			mt := resp.MountTargets[0]

			log.Printf("[DEBUG] Current status of %q: %q", *mt.MountTargetId, *mt.LifeCycleState)
			return mt, *mt.LifeCycleState, nil
		},
		Timeout:    10 * time.Minute,
		Delay:      2 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for EFS mount target (%q) to delete: %q",
			d.Id(), err.Error())
	}

	log.Printf("[DEBUG] EFS mount target %q deleted.", d.Id())

	return nil
}
