package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/rds"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDbSubnetGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbSubnetGroupCreate,
		Read:   resourceAwsDbSubnetGroupRead,
		Delete: resourceAwsDbSubnetGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"subnet_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
		},
	}
}

func resourceAwsDbSubnetGroupCreate(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn

	subnetIdsSet := d.Get("subnet_ids").(*schema.Set)
	subnetIds := make([]string, subnetIdsSet.Len())
	for i, subnetId := range subnetIdsSet.List() {
		subnetIds[i] = subnetId.(string)
	}

	createOpts := rds.CreateDBSubnetGroupMessage{
		DBSubnetGroupName:        aws.String(d.Get("name").(string)),
		DBSubnetGroupDescription: aws.String(d.Get("description").(string)),
		SubnetIDs:                subnetIds,
	}

	log.Printf("[DEBUG] Create DB Subnet Group: %#v", createOpts)
	_, err := rdsconn.CreateDBSubnetGroup(&createOpts)
	if err != nil {
		return fmt.Errorf("Error creating DB Subnet Group: %s", err)
	}

	d.SetId(*createOpts.DBSubnetGroupName)
	log.Printf("[INFO] DB Subnet Group ID: %s", d.Id())
	return resourceAwsDbSubnetGroupRead(d, meta)
}

func resourceAwsDbSubnetGroupRead(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn

	describeOpts := rds.DescribeDBSubnetGroupsMessage{
		DBSubnetGroupName: aws.String(d.Id()),
	}

	describeResp, err := rdsconn.DescribeDBSubnetGroups(&describeOpts)
	if err != nil {
		if ec2err, ok := err.(aws.APIError); ok && ec2err.Code == "DBSubnetGroupNotFoundFault" {
			// Update state to indicate the db subnet no longer exists.
			d.SetId("")
			return nil
		}
		return err
	}

	if len(describeResp.DBSubnetGroups) != 1 ||
		*describeResp.DBSubnetGroups[0].DBSubnetGroupName != d.Id() {
		return fmt.Errorf("Unable to find DB Subnet Group: %#v", describeResp.DBSubnetGroups)
	}

	subnetGroup := describeResp.DBSubnetGroups[0]

	d.Set("name", *subnetGroup.DBSubnetGroupName)
	d.Set("description", *subnetGroup.DBSubnetGroupDescription)

	subnets := make([]string, 0, len(subnetGroup.Subnets))
	for _, s := range subnetGroup.Subnets {
		subnets = append(subnets, *s.SubnetIdentifier)
	}
	d.Set("subnet_ids", subnets)

	return nil
}

func resourceAwsDbSubnetGroupDelete(d *schema.ResourceData, meta interface{}) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     "destroyed",
		Refresh:    resourceAwsDbSubnetGroupDeleteRefreshFunc(d, meta),
		Timeout:    3 * time.Minute,
		MinTimeout: 1 * time.Second,
	}
	_, err := stateConf.WaitForState()
	return err
}

func resourceAwsDbSubnetGroupDeleteRefreshFunc(
	d *schema.ResourceData,
	meta interface{}) resource.StateRefreshFunc {
	rdsconn := meta.(*AWSClient).rdsconn

	return func() (interface{}, string, error) {

		deleteOpts := rds.DeleteDBSubnetGroupMessage{
			DBSubnetGroupName: aws.String(d.Id()),
		}

		if err := rdsconn.DeleteDBSubnetGroup(&deleteOpts); err != nil {
			rdserr, ok := err.(aws.APIError)
			if !ok {
				return d, "error", err
			}

			if rdserr.Code != "DBSubnetGroupNotFoundFault" {
				return d, "error", err
			}
		}

		return d, "destroyed", nil
	}
}
