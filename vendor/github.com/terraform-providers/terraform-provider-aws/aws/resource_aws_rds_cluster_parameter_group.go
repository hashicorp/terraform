package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsRDSClusterParameterGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRDSClusterParameterGroupCreate,
		Read:   resourceAwsRDSClusterParameterGroupRead,
		Update: resourceAwsRDSClusterParameterGroupUpdate,
		Delete: resourceAwsRDSClusterParameterGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateDbParamGroupName,
			},
			"name_prefix": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateDbParamGroupNamePrefix,
			},
			"family": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "Managed by Terraform",
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
						"apply_method": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "immediate",
							// this parameter is not actually state, but a
							// meta-parameter describing how the RDS API call
							// to modify the parameter group should be made.
							// Future reads of the resource from AWS don't tell
							// us what we used for apply_method previously, so
							// by squashing state to an empty string we avoid
							// needing to do an update for every future run.
							StateFunc: func(interface{}) string { return "" },
						},
					},
				},
				Set: resourceAwsDbParameterHash,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsRDSClusterParameterGroupCreate(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn
	tags := tagsFromMapRDS(d.Get("tags").(map[string]interface{}))

	var groupName string
	if v, ok := d.GetOk("name"); ok {
		groupName = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		groupName = resource.PrefixedUniqueId(v.(string))
	} else {
		groupName = resource.UniqueId()
	}

	createOpts := rds.CreateDBClusterParameterGroupInput{
		DBClusterParameterGroupName: aws.String(groupName),
		DBParameterGroupFamily:      aws.String(d.Get("family").(string)),
		Description:                 aws.String(d.Get("description").(string)),
		Tags:                        tags,
	}

	log.Printf("[DEBUG] Create DB Cluster Parameter Group: %#v", createOpts)
	_, err := rdsconn.CreateDBClusterParameterGroup(&createOpts)
	if err != nil {
		return fmt.Errorf("Error creating DB Cluster Parameter Group: %s", err)
	}

	d.SetId(*createOpts.DBClusterParameterGroupName)
	log.Printf("[INFO] DB Cluster Parameter Group ID: %s", d.Id())

	return resourceAwsRDSClusterParameterGroupUpdate(d, meta)
}

func resourceAwsRDSClusterParameterGroupRead(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn

	describeOpts := rds.DescribeDBClusterParameterGroupsInput{
		DBClusterParameterGroupName: aws.String(d.Id()),
	}

	describeResp, err := rdsconn.DescribeDBClusterParameterGroups(&describeOpts)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "DBParameterGroupNotFound" {
			log.Printf("[WARN] DB Cluster Parameter Group (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	if len(describeResp.DBClusterParameterGroups) != 1 ||
		*describeResp.DBClusterParameterGroups[0].DBClusterParameterGroupName != d.Id() {
		return fmt.Errorf("Unable to find Cluster Parameter Group: %#v", describeResp.DBClusterParameterGroups)
	}

	d.Set("name", describeResp.DBClusterParameterGroups[0].DBClusterParameterGroupName)
	d.Set("family", describeResp.DBClusterParameterGroups[0].DBParameterGroupFamily)
	d.Set("description", describeResp.DBClusterParameterGroups[0].Description)

	// Only include user customized parameters as there's hundreds of system/default ones
	describeParametersOpts := rds.DescribeDBClusterParametersInput{
		DBClusterParameterGroupName: aws.String(d.Id()),
		Source: aws.String("user"),
	}

	describeParametersResp, err := rdsconn.DescribeDBClusterParameters(&describeParametersOpts)
	if err != nil {
		return err
	}

	d.Set("parameter", flattenParameters(describeParametersResp.Parameters))

	paramGroup := describeResp.DBClusterParameterGroups[0]
	arn, err := buildRDSCPGARN(d.Id(), meta.(*AWSClient).partition, meta.(*AWSClient).accountid, meta.(*AWSClient).region)
	if err != nil {
		name := "<empty>"
		if paramGroup.DBClusterParameterGroupName != nil && *paramGroup.DBClusterParameterGroupName != "" {
			name = *paramGroup.DBClusterParameterGroupName
		}
		log.Printf("[DEBUG] Error building ARN for DB Cluster Parameter Group, not setting Tags for Cluster Param Group %s", name)
	} else {
		d.Set("arn", arn)
		resp, err := rdsconn.ListTagsForResource(&rds.ListTagsForResourceInput{
			ResourceName: aws.String(arn),
		})

		if err != nil {
			log.Printf("[DEBUG] Error retrieving tags for ARN: %s", arn)
		}

		var dt []*rds.Tag
		if len(resp.TagList) > 0 {
			dt = resp.TagList
		}
		d.Set("tags", tagsToMapRDS(dt))
	}

	return nil
}

func resourceAwsRDSClusterParameterGroupUpdate(d *schema.ResourceData, meta interface{}) error {
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

		// Expand the "parameter" set to aws-sdk-go compat []rds.Parameter
		parameters, err := expandParameters(ns.Difference(os).List())
		if err != nil {
			return err
		}

		if len(parameters) > 0 {
			modifyOpts := rds.ModifyDBClusterParameterGroupInput{
				DBClusterParameterGroupName: aws.String(d.Get("name").(string)),
				Parameters:                  parameters,
			}

			log.Printf("[DEBUG] Modify DB Cluster Parameter Group: %s", modifyOpts)
			_, err = rdsconn.ModifyDBClusterParameterGroup(&modifyOpts)
			if err != nil {
				return fmt.Errorf("Error modifying DB Cluster Parameter Group: %s", err)
			}
		}
		d.SetPartial("parameter")
	}

	if arn, err := buildRDSCPGARN(d.Id(), meta.(*AWSClient).partition, meta.(*AWSClient).accountid, meta.(*AWSClient).region); err == nil {
		if err := setTagsRDS(rdsconn, d, arn); err != nil {
			return err
		} else {
			d.SetPartial("tags")
		}
	}

	d.Partial(false)

	return resourceAwsRDSClusterParameterGroupRead(d, meta)
}

func resourceAwsRDSClusterParameterGroupDelete(d *schema.ResourceData, meta interface{}) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     []string{"destroyed"},
		Refresh:    resourceAwsRDSClusterParameterGroupDeleteRefreshFunc(d, meta),
		Timeout:    3 * time.Minute,
		MinTimeout: 1 * time.Second,
	}
	_, err := stateConf.WaitForState()
	return err
}

func resourceAwsRDSClusterParameterGroupDeleteRefreshFunc(
	d *schema.ResourceData,
	meta interface{}) resource.StateRefreshFunc {
	rdsconn := meta.(*AWSClient).rdsconn

	return func() (interface{}, string, error) {

		deleteOpts := rds.DeleteDBClusterParameterGroupInput{
			DBClusterParameterGroupName: aws.String(d.Id()),
		}

		if _, err := rdsconn.DeleteDBClusterParameterGroup(&deleteOpts); err != nil {
			rdserr, ok := err.(awserr.Error)
			if !ok {
				return d, "error", err
			}

			if rdserr.Code() != "DBParameterGroupNotFound" {
				return d, "error", err
			}
		}

		return d, "destroyed", nil
	}
}

func buildRDSCPGARN(identifier, partition, accountid, region string) (string, error) {
	if partition == "" {
		return "", fmt.Errorf("Unable to construct RDS Cluster ARN because of missing AWS partition")
	}
	if accountid == "" {
		return "", fmt.Errorf("Unable to construct RDS Cluster ARN because of missing AWS Account ID")
	}
	arn := fmt.Sprintf("arn:%s:rds:%s:%s:cluster-pg:%s", partition, region, accountid, identifier)
	return arn, nil

}
