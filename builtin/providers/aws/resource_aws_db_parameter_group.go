package aws

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/goamz/rds"
)

func resourceAwsDbParameterGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbParameterGroupCreate,
		Read:   resourceAwsDbParameterGroupRead,
		Update: resourceAwsDbParameterGroupUpdate,
		Delete: resourceAwsDbParameterGroupDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"family": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"parameter": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceAwsDbParameterHash,
			},
		},
	}
}

func resourceAwsDbParameterGroupCreate(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn

	createOpts := rds.CreateDBParameterGroup{
		DBParameterGroupName:   d.Get("name").(string),
		DBParameterGroupFamily: d.Get("family").(string),
		Description:            d.Get("description").(string),
	}

	log.Printf("[DEBUG] Create DB Parameter Group: %#v", createOpts)
	_, err := rdsconn.CreateDBParameterGroup(&createOpts)
	if err != nil {
		return fmt.Errorf("Error creating DB Parameter Group: %s", err)
	}

	d.Partial(true)
	d.SetPartial("name")
	d.SetPartial("family")
	d.SetPartial("description")
	d.Partial(false)

	d.SetId(createOpts.DBParameterGroupName)
	log.Printf("[INFO] DB Parameter Group ID: %s", d.Id())

	return resourceAwsDbParameterGroupUpdate(d, meta)
}

func resourceAwsDbParameterGroupRead(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn

	describeOpts := rds.DescribeDBParameterGroups{
		DBParameterGroupName: d.Id(),
	}

	describeResp, err := rdsconn.DescribeDBParameterGroups(&describeOpts)
	if err != nil {
		return err
	}

	if len(describeResp.DBParameterGroups) != 1 ||
		describeResp.DBParameterGroups[0].DBParameterGroupName != d.Id() {
		return fmt.Errorf("Unable to find Parameter Group: %#v", describeResp.DBParameterGroups)
	}

	d.Set("name", describeResp.DBParameterGroups[0].DBParameterGroupName)
	d.Set("family", describeResp.DBParameterGroups[0].DBParameterGroupFamily)
	d.Set("description", describeResp.DBParameterGroups[0].Description)

	// Only include user customized parameters as there's hundreds of system/default ones
	describeParametersOpts := rds.DescribeDBParameters{
		DBParameterGroupName: d.Id(),
		Source:               "user",
	}

	describeParametersResp, err := rdsconn.DescribeDBParameters(&describeParametersOpts)
	if err != nil {
		return err
	}

	d.Set("parameter", flattenParameters(describeParametersResp.Parameters))

	return nil
}

func resourceAwsDbParameterGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn

	d.Partial(true)

	if d.HasChange("parameter") {
		o, n := d.GetChange("parameter")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		// Expand the "parameter" set to goamz compat []rds.Parameter
		parameters, err := expandParameters(ns.Difference(os).List())
		if err != nil {
			return err
		}

		if len(parameters) > 0 {
			modifyOpts := rds.ModifyDBParameterGroup{
				DBParameterGroupName: d.Get("name").(string),
				Parameters:           parameters,
			}

			log.Printf("[DEBUG] Modify DB Parameter Group: %#v", modifyOpts)
			_, err = rdsconn.ModifyDBParameterGroup(&modifyOpts)
			if err != nil {
				return fmt.Errorf("Error modifying DB Parameter Group: %s", err)
			}
		}
		d.SetPartial("parameter")
	}

	d.Partial(false)

	return resourceAwsDbParameterGroupRead(d, meta)
}

func resourceAwsDbParameterGroupDelete(d *schema.ResourceData, meta interface{}) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     "destroyed",
		Refresh:    resourceAwsDbParameterGroupDeleteRefreshFunc(d, meta),
		Timeout:    3 * time.Minute,
		MinTimeout: 1 * time.Second,
	}
	_, err := stateConf.WaitForState()
	return err
}

func resourceAwsDbParameterGroupDeleteRefreshFunc(
	d *schema.ResourceData,
	meta interface{}) resource.StateRefreshFunc {
	rdsconn := meta.(*AWSClient).rdsconn

	return func() (interface{}, string, error) {

		deleteOpts := rds.DeleteDBParameterGroup{
			DBParameterGroupName: d.Id(),
		}

		if _, err := rdsconn.DeleteDBParameterGroup(&deleteOpts); err != nil {
			rdserr, ok := err.(*rds.Error)
			if !ok {
				return d, "error", err
			}

			if rdserr.Code != "DBParameterGroupNotFoundFault" {
				return d, "error", err
			}
		}

		return d, "destroyed", nil
	}
}

func resourceAwsDbParameterHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["value"].(string)))

	return hashcode.String(buf.String())
}
