package aws

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsApiGatewayUsagePlan() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayUsagePlanCreate,
		Read:   resourceAwsApiGatewayUsagePlanRead,
		Update: resourceAwsApiGatewayUsagePlanUpdate,
		Delete: resourceAwsApiGatewayUsagePlanDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true, // Required since not addable nor removable afterwards
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"api_stages": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"api_id": {
							Type:     schema.TypeString,
							Required: true,
						},

						"stage": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"quota_settings": {
				Type:     schema.TypeSet,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"limit": {
							Type:     schema.TypeInt,
							Required: true, // Required as not removable singularly
						},

						"offset": {
							Type:     schema.TypeInt,
							Default:  0,
							Optional: true,
						},

						"period": {
							Type:         schema.TypeString,
							Required:     true, // Required as not removable
							ValidateFunc: validateApiGatewayUsagePlanQuotaSettingsPeriod,
						},
					},
				},
			},

			"throttle_settings": {
				Type:     schema.TypeSet,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"burst_limit": {
							Type:     schema.TypeInt,
							Default:  0,
							Optional: true,
						},

						"rate_limit": {
							Type:     schema.TypeFloat,
							Default:  0,
							Optional: true,
						},
					},
				},
			},

			"product_code": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsApiGatewayUsagePlanCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Print("[DEBUG] Creating API Gateway Usage Plan")

	params := &apigateway.CreateUsagePlanInput{
		Name: aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		params.Description = aws.String(v.(string))
	}

	if s, ok := d.GetOk("api_stages"); ok {
		stages := s.([]interface{})
		as := make([]*apigateway.ApiStage, 0)

		for _, v := range stages {
			sv := v.(map[string]interface{})
			stage := &apigateway.ApiStage{}

			if v, ok := sv["api_id"].(string); ok && v != "" {
				stage.ApiId = aws.String(v)
			}

			if v, ok := sv["stage"].(string); ok && v != "" {
				stage.Stage = aws.String(v)
			}

			as = append(as, stage)
		}

		if len(as) > 0 {
			params.ApiStages = as
		}
	}

	if v, ok := d.GetOk("quota_settings"); ok {
		settings := v.(*schema.Set).List()
		q, ok := settings[0].(map[string]interface{})

		if errors := validateApiGatewayUsagePlanQuotaSettings(q); len(errors) > 0 {
			return fmt.Errorf("Error validating the quota settings: %v", errors)
		}

		if !ok {
			return errors.New("At least one field is expected inside quota_settings")
		}

		qs := &apigateway.QuotaSettings{}

		if sv, ok := q["limit"].(int); ok {
			qs.Limit = aws.Int64(int64(sv))
		}

		if sv, ok := q["offset"].(int); ok {
			qs.Offset = aws.Int64(int64(sv))
		}

		if sv, ok := q["period"].(string); ok && sv != "" {
			qs.Period = aws.String(sv)
		}

		params.Quota = qs
	}

	if v, ok := d.GetOk("throttle_settings"); ok {
		settings := v.(*schema.Set).List()
		q, ok := settings[0].(map[string]interface{})

		if !ok {
			return errors.New("At least one field is expected inside throttle_settings")
		}

		ts := &apigateway.ThrottleSettings{}

		if sv, ok := q["burst_limit"].(int); ok {
			ts.BurstLimit = aws.Int64(int64(sv))
		}

		if sv, ok := q["rate_limit"].(float64); ok {
			ts.RateLimit = aws.Float64(sv)
		}

		params.Throttle = ts
	}

	up, err := conn.CreateUsagePlan(params)
	if err != nil {
		return fmt.Errorf("Error creating API Gateway Usage Plan: %s", err)
	}

	d.SetId(*up.Id)

	// Handle case of adding the product code since not addable when
	// creating the Usage Plan initially.
	if v, ok := d.GetOk("product_code"); ok {
		updateParameters := &apigateway.UpdateUsagePlanInput{
			UsagePlanId: aws.String(d.Id()),
			PatchOperations: []*apigateway.PatchOperation{
				{
					Op:    aws.String("add"),
					Path:  aws.String("/productCode"),
					Value: aws.String(v.(string)),
				},
			},
		}

		up, err = conn.UpdateUsagePlan(updateParameters)
		if err != nil {
			return fmt.Errorf("Error creating the API Gateway Usage Plan product code: %s", err)
		}
	}

	return resourceAwsApiGatewayUsagePlanRead(d, meta)
}

func resourceAwsApiGatewayUsagePlanRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Reading API Gateway Usage Plan: %s", d.Id())

	up, err := conn.GetUsagePlan(&apigateway.GetUsagePlanInput{
		UsagePlanId: aws.String(d.Id()),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			log.Printf("[WARN] API Gateway Usage Plan (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", up.Name)
	d.Set("description", up.Description)
	d.Set("product_code", up.ProductCode)

	if up.ApiStages != nil {
		if err := d.Set("api_stages", flattenApiGatewayUsageApiStages(up.ApiStages)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting api_stages error: %#v", err)
		}
	}

	if up.Throttle != nil {
		if err := d.Set("throttle_settings", flattenApiGatewayUsagePlanThrottling(up.Throttle)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting throttle_settings error: %#v", err)
		}
	}

	if up.Quota != nil {
		if err := d.Set("quota_settings", flattenApiGatewayUsagePlanQuota(up.Quota)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting quota_settings error: %#v", err)
		}
	}

	return nil
}

func resourceAwsApiGatewayUsagePlanUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Print("[DEBUG] Updating API Gateway Usage Plan")

	operations := make([]*apigateway.PatchOperation, 0)

	if d.HasChange("name") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/name"),
			Value: aws.String(d.Get("name").(string)),
		})
	}

	if d.HasChange("description") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/description"),
			Value: aws.String(d.Get("description").(string)),
		})
	}

	if d.HasChange("product_code") {
		v, ok := d.GetOk("product_code")

		if ok {
			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String("replace"),
				Path:  aws.String("/productCode"),
				Value: aws.String(v.(string)),
			})
		} else {
			operations = append(operations, &apigateway.PatchOperation{
				Op:   aws.String("remove"),
				Path: aws.String("/productCode"),
			})
		}
	}

	if d.HasChange("api_stages") {
		o, n := d.GetChange("api_stages")
		old := o.([]interface{})
		new := n.([]interface{})

		// Remove every stages associated. Simpler to remove and add new ones,
		// since there are no replacings.
		for _, v := range old {
			m := v.(map[string]interface{})
			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String("remove"),
				Path:  aws.String("/apiStages"),
				Value: aws.String(fmt.Sprintf("%s:%s", m["api_id"].(string), m["stage"].(string))),
			})
		}

		// Handle additions
		if len(new) > 0 {
			for _, v := range new {
				m := v.(map[string]interface{})
				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String("add"),
					Path:  aws.String("/apiStages"),
					Value: aws.String(fmt.Sprintf("%s:%s", m["api_id"].(string), m["stage"].(string))),
				})
			}
		}
	}

	if d.HasChange("throttle_settings") {
		o, n := d.GetChange("throttle_settings")

		os := o.(*schema.Set)
		ns := n.(*schema.Set)
		diff := ns.Difference(os).List()

		// Handle Removal
		if len(diff) == 0 {
			operations = append(operations, &apigateway.PatchOperation{
				Op:   aws.String("remove"),
				Path: aws.String("/throttle"),
			})
		}

		if len(diff) > 0 {
			d := diff[0].(map[string]interface{})

			// Handle Replaces
			if o != nil && n != nil {
				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String("replace"),
					Path:  aws.String("/throttle/rateLimit"),
					Value: aws.String(strconv.FormatFloat(d["rate_limit"].(float64), 'f', -1, 64)),
				})
				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String("replace"),
					Path:  aws.String("/throttle/burstLimit"),
					Value: aws.String(strconv.Itoa(d["burst_limit"].(int))),
				})
			}

			// Handle Additions
			if o == nil && n != nil {
				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String("add"),
					Path:  aws.String("/throttle/rateLimit"),
					Value: aws.String(strconv.FormatFloat(d["rate_limit"].(float64), 'f', -1, 64)),
				})
				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String("add"),
					Path:  aws.String("/throttle/burstLimit"),
					Value: aws.String(strconv.Itoa(d["burst_limit"].(int))),
				})
			}
		}
	}

	if d.HasChange("quota_settings") {
		o, n := d.GetChange("quota_settings")

		os := o.(*schema.Set)
		ns := n.(*schema.Set)
		diff := ns.Difference(os).List()

		// Handle Removal
		if len(diff) == 0 {
			operations = append(operations, &apigateway.PatchOperation{
				Op:   aws.String("remove"),
				Path: aws.String("/quota"),
			})
		}

		if len(diff) > 0 {
			d := diff[0].(map[string]interface{})

			if errors := validateApiGatewayUsagePlanQuotaSettings(d); len(errors) > 0 {
				return fmt.Errorf("Error validating the quota settings: %v", errors)
			}

			// Handle Replaces
			if o != nil && n != nil {
				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String("replace"),
					Path:  aws.String("/quota/limit"),
					Value: aws.String(strconv.Itoa(d["limit"].(int))),
				})
				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String("replace"),
					Path:  aws.String("/quota/offset"),
					Value: aws.String(strconv.Itoa(d["offset"].(int))),
				})
				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String("replace"),
					Path:  aws.String("/quota/period"),
					Value: aws.String(d["period"].(string)),
				})
			}

			// Handle Additions
			if o == nil && n != nil {
				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String("add"),
					Path:  aws.String("/quota/limit"),
					Value: aws.String(strconv.Itoa(d["limit"].(int))),
				})
				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String("add"),
					Path:  aws.String("/quota/offset"),
					Value: aws.String(strconv.Itoa(d["offset"].(int))),
				})
				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String("add"),
					Path:  aws.String("/quota/period"),
					Value: aws.String(d["period"].(string)),
				})
			}
		}
	}

	params := &apigateway.UpdateUsagePlanInput{
		UsagePlanId:     aws.String(d.Id()),
		PatchOperations: operations,
	}

	_, err := conn.UpdateUsagePlan(params)
	if err != nil {
		return fmt.Errorf("Error updating API Gateway Usage Plan: %s", err)
	}

	return resourceAwsApiGatewayUsagePlanRead(d, meta)
}

func resourceAwsApiGatewayUsagePlanDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	// Removing existing api stages associated
	if apistages, ok := d.GetOk("api_stages"); ok {
		log.Printf("[DEBUG] Deleting API Stages associated with Usage Plan: %s", d.Id())
		stages := apistages.([]interface{})
		operations := []*apigateway.PatchOperation{}

		for _, v := range stages {
			sv := v.(map[string]interface{})

			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String("remove"),
				Path:  aws.String("/apiStages"),
				Value: aws.String(fmt.Sprintf("%s:%s", sv["api_id"].(string), sv["stage"].(string))),
			})
		}

		_, err := conn.UpdateUsagePlan(&apigateway.UpdateUsagePlanInput{
			UsagePlanId:     aws.String(d.Id()),
			PatchOperations: operations,
		})
		if err != nil {
			return fmt.Errorf("Error removing API Stages associated with Usage Plan: %s", err)
		}
	}

	log.Printf("[DEBUG] Deleting API Gateway Usage Plan: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteUsagePlan(&apigateway.DeleteUsagePlanInput{
			UsagePlanId: aws.String(d.Id()),
		})

		if err == nil {
			return nil
		}

		return resource.NonRetryableError(err)
	})
}
