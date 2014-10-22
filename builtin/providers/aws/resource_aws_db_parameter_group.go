package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/goamz/rds"
)

func resourceAwsDbParameterGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbParameterGroupCreate,
		Read:   resourceAwsDbParameterGroupRead,
		Update: nil,
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
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"apply_method": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
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
			},
		},
	}
}

func resourceAwsDbParameterGroupCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	rdsconn := p.rdsconn

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

	d.SetId(createOpts.DBParameterGroupName)
	log.Printf("[INFO] DB Parameter Group ID: %s", d.Id())

	if d.Get("parameter") != "" {
		// Expand the "parameter" set to goamz compat []rds.Parameter
		parameters, err := expandParameters(d.Get("parameter").([]interface{}))
		if err != nil {
			return err
		}

		modifyOpts := rds.ModifyDBParameterGroup{
			DBParameterGroupName:   d.Get("name").(string),
			Parameters:             parameters,
		}

		log.Printf("[DEBUG] Modify DB Parameter Group: %#v", modifyOpts)
		_, err = rdsconn.ModifyDBParameterGroup(&modifyOpts)
		if err != nil {
			return fmt.Errorf("Error modifying DB Parameter Group: %s", err)
		}
	}

	return resourceAwsDbParameterGroupRead(d, meta)
}

func resourceAwsDbParameterGroupDelete(d *schema.ResourceData, meta interface{}) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     "destroyed",
		Refresh:    resourceDbParameterGroupDeleteRefreshFunc(d, meta),
		Timeout:    3 * time.Minute,
		MinTimeout: 1 * time.Second,
	}
	_, err := stateConf.WaitForState()
	return err
}

func resourceAwsDbParameterGroupRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	rdsconn := p.rdsconn

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

func resourceDbParameterGroupDeleteRefreshFunc(
	d *schema.ResourceData,
	meta interface{}) resource.StateRefreshFunc {
	p := meta.(*ResourceProvider)
	rdsconn := p.rdsconn

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
