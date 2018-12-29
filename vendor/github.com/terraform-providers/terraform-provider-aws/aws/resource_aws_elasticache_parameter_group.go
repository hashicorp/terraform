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
	"github.com/aws/aws-sdk-go/service/elasticache"
)

func resourceAwsElasticacheParameterGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticacheParameterGroupCreate,
		Read:   resourceAwsElasticacheParameterGroupRead,
		Update: resourceAwsElasticacheParameterGroupUpdate,
		Delete: resourceAwsElasticacheParameterGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
				StateFunc: func(val interface{}) string {
					return strings.ToLower(val.(string))
				},
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
	resp, err := conn.CreateCacheParameterGroup(&createOpts)
	if err != nil {
		return fmt.Errorf("Error creating Cache Parameter Group: %s", err)
	}

	d.Partial(true)
	d.SetPartial("name")
	d.SetPartial("family")
	d.SetPartial("description")
	d.Partial(false)

	d.SetId(*resp.CacheParameterGroup.CacheParameterGroupName)
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

		toRemove, err := expandElastiCacheParameters(os.Difference(ns).List())
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] Parameters to remove: %#v", toRemove)

		toAdd, err := expandElastiCacheParameters(ns.Difference(os).List())
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] Parameters to add: %#v", toAdd)

		// We can only modify 20 parameters at a time, so walk them until
		// we've got them all.
		maxParams := 20

		for len(toRemove) > 0 {
			paramsToModify := make([]*elasticache.ParameterNameValue, 0)
			if len(toRemove) <= maxParams {
				paramsToModify, toRemove = toRemove[:], nil
			} else {
				paramsToModify, toRemove = toRemove[:maxParams], toRemove[maxParams:]
			}
			resetOpts := elasticache.ResetCacheParameterGroupInput{
				CacheParameterGroupName: aws.String(d.Get("name").(string)),
				ParameterNameValues:     paramsToModify,
			}

			log.Printf("[DEBUG] Reset Cache Parameter Group: %s", resetOpts)
			err := resource.Retry(30*time.Second, func() *resource.RetryError {
				_, err = conn.ResetCacheParameterGroup(&resetOpts)
				if err != nil {
					if isAWSErr(err, "InvalidCacheParameterGroupState", " has pending changes") {
						return resource.RetryableError(err)
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("Error resetting Cache Parameter Group: %s", err)
			}
		}

		for len(toAdd) > 0 {
			paramsToModify := make([]*elasticache.ParameterNameValue, 0)
			if len(toAdd) <= maxParams {
				paramsToModify, toAdd = toAdd[:], nil
			} else {
				paramsToModify, toAdd = toAdd[:maxParams], toAdd[maxParams:]
			}
			modifyOpts := elasticache.ModifyCacheParameterGroupInput{
				CacheParameterGroupName: aws.String(d.Get("name").(string)),
				ParameterNameValues:     paramsToModify,
			}

			log.Printf("[DEBUG] Modify Cache Parameter Group: %s", modifyOpts)
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
	conn := meta.(*AWSClient).elasticacheconn

	return resource.Retry(3*time.Minute, func() *resource.RetryError {
		deleteOpts := elasticache.DeleteCacheParameterGroupInput{
			CacheParameterGroupName: aws.String(d.Id()),
		}
		_, err := conn.DeleteCacheParameterGroup(&deleteOpts)
		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if ok && awsErr.Code() == "CacheParameterGroupNotFoundFault" {
				d.SetId("")
				return nil
			}
			if ok && awsErr.Code() == "InvalidCacheParameterGroupState" {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
}

func resourceAwsElasticacheParameterHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["value"].(string)))

	return hashcode.String(buf.String())
}
