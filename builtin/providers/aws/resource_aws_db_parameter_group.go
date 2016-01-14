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
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/rds"
)

func resourceAwsDbParameterGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbParameterGroupCreate,
		Read:   resourceAwsDbParameterGroupRead,
		Update: resourceAwsDbParameterGroupUpdate,
		Delete: resourceAwsDbParameterGroupDelete,
		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validateDbParamGroupName,
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

func resourceAwsDbParameterGroupCreate(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn
	tags := tagsFromMapRDS(d.Get("tags").(map[string]interface{}))

	createOpts := rds.CreateDBParameterGroupInput{
		DBParameterGroupName:   aws.String(d.Get("name").(string)),
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
	arn, err := buildRDSPGARN(d, meta)
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
			modifyOpts := rds.ModifyDBParameterGroupInput{
				DBParameterGroupName: aws.String(d.Get("name").(string)),
				Parameters:           parameters,
			}

			log.Printf("[DEBUG] Modify DB Parameter Group: %s", modifyOpts)
			_, err = rdsconn.ModifyDBParameterGroup(&modifyOpts)
			if err != nil {
				return fmt.Errorf("Error modifying DB Parameter Group: %s", err)
			}
		}
		d.SetPartial("parameter")
	}

	if arn, err := buildRDSPGARN(d, meta); err == nil {
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

		deleteOpts := rds.DeleteDBParameterGroupInput{
			DBParameterGroupName: aws.String(d.Id()),
		}

		if _, err := rdsconn.DeleteDBParameterGroup(&deleteOpts); err != nil {
			rdserr, ok := err.(awserr.Error)
			if !ok {
				return d, "error", err
			}

			if rdserr.Code() != "DBParameterGroupNotFoundFault" {
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
	// Store the value as a lower case string, to match how we store them in flattenParameters
	buf.WriteString(fmt.Sprintf("%s-", strings.ToLower(m["value"].(string))))

	return hashcode.String(buf.String())
}

func buildRDSPGARN(d *schema.ResourceData, meta interface{}) (string, error) {
	iamconn := meta.(*AWSClient).iamconn
	region := meta.(*AWSClient).region
	// An zero value GetUserInput{} defers to the currently logged in user
	resp, err := iamconn.GetUser(&iam.GetUserInput{})
	if err != nil {
		return "", err
	}
	userARN := *resp.User.Arn
	accountID := strings.Split(userARN, ":")[4]
	arn := fmt.Sprintf("arn:aws:rds:%s:%s:pg:%s", region, accountID, d.Id())
	return arn, nil
}
