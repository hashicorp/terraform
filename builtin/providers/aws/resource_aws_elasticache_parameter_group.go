package aws

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticache"
)

func resourceAwsElasticacheParameterGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticacheParameterGroupCreate,
		Read:   resourceAwsElasticacheParameterGroupRead,
		Update: resourceAwsElasticacheParameterGroupUpdate,
		Delete: resourceAwsElasticacheParameterGroupDelete,
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
				Set: resourceAwsElasticacheParameterHash,
			},
		},
	}
}

func resourceAwsElasticacheParameterGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	createOpts := elasticache.CreateCacheParameterGroupInput{
		CacheParameterGroupName:   aws.String(d.Get("name").(string)),
		CacheParameterGroupFamily: aws.String(d.Get("family").(string)),
		Description:               aws.String(d.Get("description").(string)),
	}

	log.Printf("[DEBUG] Create Cache Parameter Group: %#v", createOpts)
	_, err := conn.CreateCacheParameterGroup(&createOpts)
	if err != nil {
		return fmt.Errorf("Error creating Cache Parameter Group: %s", err)
	}

	d.Partial(true)
	d.SetPartial("name")
	d.SetPartial("family")
	d.SetPartial("description")
	d.Partial(false)

	d.SetId(*createOpts.CacheParameterGroupName)
	log.Printf("[INFO] Cache Parameter Group ID: %s", d.Id())

	return resourceAwsElasticacheParameterGroupUpdate(d, meta)
}

func resourceAwsElasticacheParameterGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	describeOpts := elasticache.DescribeCacheParameterGroupsInput{
		CacheParameterGroupName: aws.String(d.Id()),
	}

	describeResp, err := conn.DescribeCacheParameterGroups(&describeOpts)
	if err != nil {
		return err
	}

	if len(describeResp.CacheParameterGroups) != 1 ||
		*describeResp.CacheParameterGroups[0].CacheParameterGroupName != d.Id() {
		return fmt.Errorf("Unable to find Parameter Group: %#v", describeResp.CacheParameterGroups)
	}

	d.Set("name", describeResp.CacheParameterGroups[0].CacheParameterGroupName)
	d.Set("family", describeResp.CacheParameterGroups[0].CacheParameterGroupFamily)
	d.Set("description", describeResp.CacheParameterGroups[0].Description)

	// Only include user customized parameters as there's hundreds of system/default ones
	describeParametersOpts := elasticache.DescribeCacheParametersInput{
		CacheParameterGroupName: aws.String(d.Id()),
		Source:                  aws.String("user"),
	}

	describeParametersResp, err := conn.DescribeCacheParameters(&describeParametersOpts)
	if err != nil {
		return err
	}

	d.Set("parameter", flattenElastiCacheParameters(describeParametersResp.Parameters))

	return nil
}

func resourceAwsElasticacheParameterGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

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

		// Expand the "parameter" set to aws-sdk-go compat []elasticacheconn.Parameter
		parameters, err := expandElastiCacheParameters(ns.Difference(os).List())
		if err != nil {
			return err
		}

		if len(parameters) > 0 {
			modifyOpts := elasticache.ModifyCacheParameterGroupInput{
				CacheParameterGroupName: aws.String(d.Get("name").(string)),
				ParameterNameValues:     parameters,
			}

			log.Printf("[DEBUG] Modify Cache Parameter Group: %#v", modifyOpts)
			_, err = conn.ModifyCacheParameterGroup(&modifyOpts)
			if err != nil {
				return fmt.Errorf("Error modifying Cache Parameter Group: %s", err)
			}
		}
		d.SetPartial("parameter")
	}

	d.Partial(false)

	return resourceAwsElasticacheParameterGroupRead(d, meta)
}

func resourceAwsElasticacheParameterGroupDelete(d *schema.ResourceData, meta interface{}) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     []string{"destroyed"},
		Refresh:    resourceAwsElasticacheParameterGroupDeleteRefreshFunc(d, meta),
		Timeout:    3 * time.Minute,
		MinTimeout: 1 * time.Second,
	}
	_, err := stateConf.WaitForState()
	return err
}

func resourceAwsElasticacheParameterGroupDeleteRefreshFunc(
	d *schema.ResourceData,
	meta interface{}) resource.StateRefreshFunc {
	conn := meta.(*AWSClient).elasticacheconn

	return func() (interface{}, string, error) {

		deleteOpts := elasticache.DeleteCacheParameterGroupInput{
			CacheParameterGroupName: aws.String(d.Id()),
		}

		if _, err := conn.DeleteCacheParameterGroup(&deleteOpts); err != nil {
			elasticahceerr, ok := err.(awserr.Error)
			if ok && elasticahceerr.Code() == "CacheParameterGroupNotFoundFault" {
				d.SetId("")
				return d, "error", err
			}
			return d, "error", err
		}
		return d, "destroyed", nil
	}
}

func resourceAwsElasticacheParameterHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["value"].(string)))

	return hashcode.String(buf.String())
}
