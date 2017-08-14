package aws

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
)

func resourceAwsDbParameterGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbParameterGroupCreate,
		Read:   resourceAwsDbParameterGroupRead,
		Update: resourceAwsDbParameterGroupUpdate,
		Delete: resourceAwsDbParameterGroupDelete,
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
						},
					},
				},
				Set: resourceAwsDbParameterHash,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsDbParameterGroupCreate(d *schema.ResourceData, meta interface{}) error {
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
	d.Set("name", groupName)

	createOpts := rds.CreateDBParameterGroupInput{
		DBParameterGroupName:   aws.String(groupName),
		DBParameterGroupFamily: aws.String(d.Get("family").(string)),
		Description:            aws.String(d.Get("description").(string)),
		Tags:                   tags,
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

	d.SetId(*createOpts.DBParameterGroupName)
	log.Printf("[INFO] DB Parameter Group ID: %s", d.Id())

	return resourceAwsDbParameterGroupUpdate(d, meta)
}

func resourceAwsDbParameterGroupRead(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn

	describeOpts := rds.DescribeDBParameterGroupsInput{
		DBParameterGroupName: aws.String(d.Id()),
	}

	describeResp, err := rdsconn.DescribeDBParameterGroups(&describeOpts)
	if err != nil {
		return err
	}

	if len(describeResp.DBParameterGroups) != 1 ||
		*describeResp.DBParameterGroups[0].DBParameterGroupName != d.Id() {
		return fmt.Errorf("Unable to find Parameter Group: %#v", describeResp.DBParameterGroups)
	}

	d.Set("name", describeResp.DBParameterGroups[0].DBParameterGroupName)
	d.Set("family", describeResp.DBParameterGroups[0].DBParameterGroupFamily)
	d.Set("description", describeResp.DBParameterGroups[0].Description)

	// Only include user customized parameters as there's hundreds of system/default ones
	describeParametersOpts := rds.DescribeDBParametersInput{
		DBParameterGroupName: aws.String(d.Id()),
		Source:               aws.String("user"),
	}

	describeParametersResp, err := rdsconn.DescribeDBParameters(&describeParametersOpts)
	if err != nil {
		return err
	}

	d.Set("parameter", flattenParameters(describeParametersResp.Parameters))

	paramGroup := describeResp.DBParameterGroups[0]
	arn, err := buildRDSPGARN(d.Id(), meta.(*AWSClient).partition, meta.(*AWSClient).accountid, meta.(*AWSClient).region)
	if err != nil {
		name := "<empty>"
		if paramGroup.DBParameterGroupName != nil && *paramGroup.DBParameterGroupName != "" {
			name = *paramGroup.DBParameterGroupName
		}
		log.Printf("[DEBUG] Error building ARN for DB Parameter Group, not setting Tags for Param Group %s", name)
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

		// Expand the "parameter" set to aws-sdk-go compat []rds.Parameter
		parameters, err := expandParameters(ns.Difference(os).List())
		if err != nil {
			return err
		}

		if len(parameters) > 0 {
			// We can only modify 20 parameters at a time, so walk them until
			// we've got them all.
			maxParams := 20
			for parameters != nil {
				paramsToModify := make([]*rds.Parameter, 0)
				if len(parameters) <= maxParams {
					paramsToModify, parameters = parameters[:], nil
				} else {
					paramsToModify, parameters = parameters[:maxParams], parameters[maxParams:]
				}
				modifyOpts := rds.ModifyDBParameterGroupInput{
					DBParameterGroupName: aws.String(d.Get("name").(string)),
					Parameters:           paramsToModify,
				}

				log.Printf("[DEBUG] Modify DB Parameter Group: %s", modifyOpts)
				_, err = rdsconn.ModifyDBParameterGroup(&modifyOpts)
				if err != nil {
					return fmt.Errorf("Error modifying DB Parameter Group: %s", err)
				}
			}
			d.SetPartial("parameter")
		}
	}

	if arn, err := buildRDSPGARN(d.Id(), meta.(*AWSClient).partition, meta.(*AWSClient).accountid, meta.(*AWSClient).region); err == nil {
		if err := setTagsRDS(rdsconn, d, arn); err != nil {
			return err
		} else {
			d.SetPartial("tags")
		}
	}

	d.Partial(false)

	return resourceAwsDbParameterGroupRead(d, meta)
}

func resourceAwsDbParameterGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	return resource.Retry(3*time.Minute, func() *resource.RetryError {
		deleteOpts := rds.DeleteDBParameterGroupInput{
			DBParameterGroupName: aws.String(d.Id()),
		}

		_, err := conn.DeleteDBParameterGroup(&deleteOpts)
		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if ok && awsErr.Code() == "DBParameterGroupNotFoundFault" {
				return resource.RetryableError(err)
			}
			if ok && awsErr.Code() == "InvalidDBParameterGroupState" {
				return resource.RetryableError(err)
			}
		}
		return resource.NonRetryableError(err)
	})
}

func resourceAwsDbParameterHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	// Store the value as a lower case string, to match how we store them in flattenParameters
	buf.WriteString(fmt.Sprintf("%s-", strings.ToLower(m["value"].(string))))

	return hashcode.String(buf.String())
}

func buildRDSPGARN(identifier, partition, accountid, region string) (string, error) {
	if partition == "" {
		return "", fmt.Errorf("Unable to construct RDS ARN because of missing AWS partition")
	}
	if accountid == "" {
		return "", fmt.Errorf("Unable to construct RDS ARN because of missing AWS Account ID")
	}
	arn := fmt.Sprintf("arn:%s:rds:%s:%s:pg:%s", partition, region, accountid, identifier)
	return arn, nil

}
