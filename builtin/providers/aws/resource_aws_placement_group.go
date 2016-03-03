package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsPlacementGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsPlacementGroupCreate,
		Read:   resourceAwsPlacementGroupRead,
		Delete: resourceAwsPlacementGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"strategy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsPlacementGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	name := d.Get("name").(string)
	input := ec2.CreatePlacementGroupInput{
		GroupName: aws.String(name),
		Strategy:  aws.String(d.Get("strategy").(string)),
	}
	log.Printf("[DEBUG] Creating EC2 Placement group: %s", input)
	_, err := conn.CreatePlacementGroup(&input)
	if err != nil {
		return err
	}

	wait := resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     []string{"available"},
		Timeout:    5 * time.Minute,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.DescribePlacementGroups(&ec2.DescribePlacementGroupsInput{
				GroupNames: []*string{aws.String(name)},
			})

			if err != nil {
				return out, "", err
			}

			if len(out.PlacementGroups) == 0 {
				return out, "", fmt.Errorf("Placement group not found (%q)", name)
			}
			pg := out.PlacementGroups[0]

			return out, *pg.State, nil
		},
	}

	_, err = wait.WaitForState()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] EC2 Placement group created: %q", name)

	d.SetId(name)

	return resourceAwsPlacementGroupRead(d, meta)
}

func resourceAwsPlacementGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	input := ec2.DescribePlacementGroupsInput{
		GroupNames: []*string{aws.String(d.Get("name").(string))},
	}
	out, err := conn.DescribePlacementGroups(&input)
	if err != nil {
		return err
	}
	pg := out.PlacementGroups[0]

	log.Printf("[DEBUG] Received EC2 Placement Group: %s", pg)

	d.Set("name", pg.GroupName)
	d.Set("strategy", pg.Strategy)

	return nil
}

func resourceAwsPlacementGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[DEBUG] Deleting EC2 Placement Group %q", d.Id())
	_, err := conn.DeletePlacementGroup(&ec2.DeletePlacementGroupInput{
		GroupName: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	wait := resource.StateChangeConf{
		Pending:    []string{"deleting"},
		Target:     []string{"deleted"},
		Timeout:    5 * time.Minute,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.DescribePlacementGroups(&ec2.DescribePlacementGroupsInput{
				GroupNames: []*string{aws.String(d.Id())},
			})

			if err != nil {
				awsErr := err.(awserr.Error)
				if awsErr.Code() == "InvalidPlacementGroup.Unknown" {
					return out, "deleted", nil
				}
				return out, "", awsErr
			}

			if len(out.PlacementGroups) == 0 {
				return out, "deleted", nil
			}

			pg := out.PlacementGroups[0]

			return out, *pg.State, nil
		},
	}

	_, err = wait.WaitForState()
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
