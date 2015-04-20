package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSecurityGroupCreate,
		Read:   resourceAwsSecurityGroupRead,
		Update: resourceAwsSecurityGroupUpdate,
		Delete: resourceAwsSecurityGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"owner_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsSecurityGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	securityGroupOpts := &ec2.CreateSecurityGroupInput{
		GroupName: aws.String(d.Get("name").(string)),
	}

	if v := d.Get("vpc_id"); v != nil {
		securityGroupOpts.VPCID = aws.String(v.(string))
	}

	if v := d.Get("description"); v != nil {
		securityGroupOpts.Description = aws.String(v.(string))
	}

	log.Printf(
		"[DEBUG] Security Group create configuration: %#v", securityGroupOpts)
	createResp, err := conn.CreateSecurityGroup(securityGroupOpts)
	if err != nil {
		return fmt.Errorf("Error creating Security Group: %s", err)
	}

	d.SetId(*createResp.GroupID)

	log.Printf("[INFO] Security Group ID: %s", d.Id())

	// Wait for the security group to truly exist
	log.Printf(
		"[DEBUG] Waiting for Security Group (%s) to exist",
		d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{""},
		Target:  "exists",
		Refresh: SGStateRefreshFunc(conn, d.Id()),
		Timeout: 1 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for Security Group (%s) to become available: %s",
			d.Id(), err)
	}

	return resourceAwsSecurityGroupUpdate(d, meta)
}

func resourceAwsSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	sgRaw, _, err := SGStateRefreshFunc(conn, d.Id())()
	if err != nil {
		return err
	}
	if sgRaw == nil {
		d.SetId("")
		return nil
	}

	sg := sgRaw.(*ec2.SecurityGroup)

	d.Set("description", sg.Description)
	d.Set("name", sg.GroupName)
	d.Set("vpc_id", sg.VPCID)
	d.Set("owner_id", sg.OwnerID)
	d.Set("tags", tagsToMapSDK(sg.Tags))
	return nil
}

func resourceAwsSecurityGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	sgRaw, _, err := SGStateRefreshFunc(conn, d.Id())()
	if err != nil {
		return err
	}
	if sgRaw == nil {
		d.SetId("")
		return nil
	}

	if err := setTagsSDK(conn, d); err != nil {
		return err
	}

	d.SetPartial("tags")

	return resourceAwsSecurityGroupRead(d, meta)
}

func resourceAwsSecurityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[DEBUG] Security Group destroy: %v", d.Id())

	return resource.Retry(5*time.Minute, func() error {
		_, err := conn.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
			GroupID: aws.String(d.Id()),
		})
		if err != nil {
			ec2err, ok := err.(aws.APIError)
			if !ok {
				return err
			}

			switch ec2err.Code {
			case "InvalidGroup.NotFound":
				return nil
			case "DependencyViolation":
				// If it is a dependency violation, we want to retry
				return err
			default:
				// Any other error, we want to quit the retry loop immediately
				return resource.RetryError{Err: err}
			}
		}

		return nil
	})
}

// SGStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a security group.
func SGStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		req := &ec2.DescribeSecurityGroupsInput{
			GroupIDs: []*string{aws.String(id)},
		}
		resp, err := conn.DescribeSecurityGroups(req)
		if err != nil {
			if ec2err, ok := err.(aws.APIError); ok {
				if ec2err.Code == "InvalidSecurityGroupID.NotFound" ||
					ec2err.Code == "InvalidGroup.NotFound" {
					resp = nil
					err = nil
				}
			}

			if err != nil {
				log.Printf("Error on SGStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			return nil, "", nil
		}

		group := resp.SecurityGroups[0]
		return group, "exists", nil
	}
}
