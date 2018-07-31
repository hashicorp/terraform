package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/neptune"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

const neptuneClusterParameterGroupMaxParamsBulkEdit = 20

func resourceAwsNeptuneClusterParameterGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsNeptuneClusterParameterGroupCreate,
		Read:   resourceAwsNeptuneClusterParameterGroupRead,
		Update: resourceAwsNeptuneClusterParameterGroupUpdate,
		Delete: resourceAwsNeptuneClusterParameterGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateNeptuneParamGroupName,
			},
			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc:  validateNeptuneParamGroupNamePrefix,
			},
			"family": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "Managed by Terraform",
			},
			"parameter": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
						"apply_method": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  neptune.ApplyMethodPendingReboot,
							ValidateFunc: validation.StringInSlice([]string{
								neptune.ApplyMethodImmediate,
								neptune.ApplyMethodPendingReboot,
							}, false),
						},
					},
				},
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsNeptuneClusterParameterGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn
	tags := tagsFromMapNeptune(d.Get("tags").(map[string]interface{}))

	var groupName string
	if v, ok := d.GetOk("name"); ok {
		groupName = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		groupName = resource.PrefixedUniqueId(v.(string))
	} else {
		groupName = resource.UniqueId()
	}

	createOpts := neptune.CreateDBClusterParameterGroupInput{
		DBClusterParameterGroupName: aws.String(groupName),
		DBParameterGroupFamily:      aws.String(d.Get("family").(string)),
		Description:                 aws.String(d.Get("description").(string)),
		Tags:                        tags,
	}

	log.Printf("[DEBUG] Create Neptune Cluster Parameter Group: %#v", createOpts)
	resp, err := conn.CreateDBClusterParameterGroup(&createOpts)
	if err != nil {
		return fmt.Errorf("Error creating Neptune Cluster Parameter Group: %s", err)
	}

	d.SetId(aws.StringValue(createOpts.DBClusterParameterGroupName))
	log.Printf("[INFO] Neptune Cluster Parameter Group ID: %s", d.Id())

	d.Set("arn", resp.DBClusterParameterGroup.DBClusterParameterGroupArn)

	return resourceAwsNeptuneClusterParameterGroupUpdate(d, meta)
}

func resourceAwsNeptuneClusterParameterGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn

	describeOpts := neptune.DescribeDBClusterParameterGroupsInput{
		DBClusterParameterGroupName: aws.String(d.Id()),
	}

	describeResp, err := conn.DescribeDBClusterParameterGroups(&describeOpts)
	if err != nil {
		if isAWSErr(err, neptune.ErrCodeDBParameterGroupNotFoundFault, "") {
			log.Printf("[WARN] Neptune Cluster Parameter Group (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	if len(describeResp.DBClusterParameterGroups) == 0 {
		log.Printf("[WARN] Neptune Cluster Parameter Group (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("name", describeResp.DBClusterParameterGroups[0].DBClusterParameterGroupName)
	d.Set("family", describeResp.DBClusterParameterGroups[0].DBParameterGroupFamily)
	d.Set("description", describeResp.DBClusterParameterGroups[0].Description)
	arn := aws.StringValue(describeResp.DBClusterParameterGroups[0].DBClusterParameterGroupArn)
	d.Set("arn", arn)

	// Only include user customized parameters as there's hundreds of system/default ones
	describeParametersOpts := neptune.DescribeDBClusterParametersInput{
		DBClusterParameterGroupName: aws.String(d.Id()),
		Source: aws.String("user"),
	}

	describeParametersResp, err := conn.DescribeDBClusterParameters(&describeParametersOpts)
	if err != nil {
		return err
	}

	if err := d.Set("parameter", flattenNeptuneParameters(describeParametersResp.Parameters)); err != nil {
		return fmt.Errorf("error setting neptune parameter: %s", err)
	}

	resp, err := conn.ListTagsForResource(&neptune.ListTagsForResourceInput{
		ResourceName: aws.String(arn),
	})
	if err != nil {
		log.Printf("[DEBUG] Error retrieving tags for ARN: %s", arn)
	}

	if err := d.Set("tags", tagsToMapNeptune(resp.TagList)); err != nil {
		return fmt.Errorf("error setting neptune tags: %s", err)
	}

	return nil
}

func resourceAwsNeptuneClusterParameterGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn

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

		parameters, err := expandNeptuneParameters(ns.Difference(os).List())
		if err != nil {
			return err
		}

		if len(parameters) > 0 {
			// We can only modify 20 parameters at a time, so walk them until
			// we've got them all.
			for parameters != nil {
				paramsToModify := make([]*neptune.Parameter, 0)
				if len(parameters) <= neptuneClusterParameterGroupMaxParamsBulkEdit {
					paramsToModify, parameters = parameters[:], nil
				} else {
					paramsToModify, parameters = parameters[:neptuneClusterParameterGroupMaxParamsBulkEdit], parameters[neptuneClusterParameterGroupMaxParamsBulkEdit:]
				}
				parameterGroupName := d.Get("name").(string)
				modifyOpts := neptune.ModifyDBClusterParameterGroupInput{
					DBClusterParameterGroupName: aws.String(parameterGroupName),
					Parameters:                  paramsToModify,
				}

				log.Printf("[DEBUG] Modify Neptune Cluster Parameter Group: %s", modifyOpts)
				_, err = conn.ModifyDBClusterParameterGroup(&modifyOpts)
				if err != nil {
					return fmt.Errorf("Error modifying Neptune Cluster Parameter Group: %s", err)
				}
			}
			d.SetPartial("parameter")
		}
	}

	arn := d.Get("arn").(string)
	if err := setTagsNeptune(conn, d, arn); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	d.Partial(false)

	return resourceAwsNeptuneClusterParameterGroupRead(d, meta)
}

func resourceAwsNeptuneClusterParameterGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn

	input := neptune.DeleteDBClusterParameterGroupInput{
		DBClusterParameterGroupName: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting Neptune Cluster Parameter Group: %s", d.Id())
	_, err := conn.DeleteDBClusterParameterGroup(&input)
	if err != nil {
		if isAWSErr(err, neptune.ErrCodeDBParameterGroupNotFoundFault, "") {
			return nil
		}
		return fmt.Errorf("error deleting Neptune Cluster Parameter Group (%s): %s", d.Id(), err)
	}

	return nil
}
