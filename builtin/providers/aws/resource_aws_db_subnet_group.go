package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/goamz/rds"
)

func resourceAwsDbSubnetGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbSubnetGroupCreate,
		Read:   resourceAwsDbSubnetGroupRead,
		Update: nil,
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
	p := meta.(*ResourceProvider)
	rdsconn := p.rdsconn

	subnetIdsSet := d.Get("subnet_ids").(*schema.Set)
	subnetIds := make([]string, subnetIdsSet.Len())
	for i, subnetId := range subnetIdsSet.List() {
		subnetIds[i] = subnetId.(string)
	}

	createOpts := rds.CreateDBSubnetGroup{
		DBSubnetGroupName:        d.Get("name").(string),
		DBSubnetGroupDescription: d.Get("description").(string),
		SubnetIds:                subnetIds,
	}

	log.Printf("[DEBUG] Create DB Subnet Group: %#v", createOpts)
	_, err := rdsconn.CreateDBSubnetGroup(&createOpts)
	if err != nil {
		return fmt.Errorf("Error creating DB Subnet Group: %s", err)
	}

	d.SetId(createOpts.DBSubnetGroupName)
	log.Printf("[INFO] DB Subnet Group ID: %s", d.Id())
	return resourceAwsDbSubnetGroupRead(d, meta)
}

func resourceAwsDbSubnetGroupDelete(d *schema.ResourceData, meta interface{}) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     "destroyed",
		Refresh:    resourceDbSubnetGroupDeleteRefreshFunc(d, meta),
		Timeout:    3 * time.Minute,
		MinTimeout: 1 * time.Second,
	}
	_, err := stateConf.WaitForState()
	return err
}

func resourceAwsDbSubnetGroupRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	rdsconn := p.rdsconn

	describeOpts := rds.DescribeDBSubnetGroups{
		DBSubnetGroupName: d.Id(),
	}

	describeResp, err := rdsconn.DescribeDBSubnetGroups(&describeOpts)
	if err != nil {
		return err
	}

	if len(describeResp.DBSubnetGroups) != 1 ||
		describeResp.DBSubnetGroups[0].Name != d.Id() {
	}

	d.Set("name", describeResp.DBSubnetGroups[0].Name)
	d.Set("description", describeResp.DBSubnetGroups[0].Description)
	d.Set("subnet_ids", describeResp.DBSubnetGroups[0].SubnetIds)

	return nil
}

func resourceDbSubnetGroupDeleteRefreshFunc(
	d *schema.ResourceData,
	meta interface{}) resource.StateRefreshFunc {
	p := meta.(*ResourceProvider)
	rdsconn := p.rdsconn

	return func() (interface{}, string, error) {

		deleteOpts := rds.DeleteDBSubnetGroup{
			DBSubnetGroupName: d.Id(),
		}

		if _, err := rdsconn.DeleteDBSubnetGroup(&deleteOpts); err != nil {
			rdserr, ok := err.(*rds.Error)
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
