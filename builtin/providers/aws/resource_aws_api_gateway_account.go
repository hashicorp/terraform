package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsApiGatewayAccount() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayAccountUpdate,
		Read:   resourceAwsApiGatewayAccountRead,
		Update: resourceAwsApiGatewayAccountUpdate,
		Delete: resourceAwsApiGatewayAccountDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"cloudwatch_role_arn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"throttle_settings": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"burst_limit": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"rate_limit": &schema.Schema{
							Type:     schema.TypeFloat,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsApiGatewayAccountRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[INFO] Reading API Gateway Account %s", d.Id())
	account, err := conn.GetAccount(&apigateway.GetAccountInput{})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Received API Gateway Account: %s", account)

	if _, ok := d.GetOk("cloudwatch_role_arn"); ok {
		// CloudwatchRoleArn cannot be empty nor made empty via API
		// This resource can however be useful w/out defining cloudwatch_role_arn
		// (e.g. for referencing throttle_settings)
		d.Set("cloudwatch_role_arn", account.CloudwatchRoleArn)
	}
	d.Set("throttle_settings", flattenApiGatewayThrottleSettings(account.ThrottleSettings))

	return nil
}

func resourceAwsApiGatewayAccountUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	input := apigateway.UpdateAccountInput{}
	operations := make([]*apigateway.PatchOperation, 0)

	if d.HasChange("cloudwatch_role_arn") {
		arn := d.Get("cloudwatch_role_arn").(string)
		if len(arn) > 0 {
			// Unfortunately AWS API doesn't allow empty ARNs,
			// even though that's default settings for new AWS accounts
			// BadRequestException: The role ARN is not well formed
			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String("replace"),
				Path:  aws.String("/cloudwatchRoleArn"),
				Value: aws.String(arn),
			})
		}
	}
	input.PatchOperations = operations

	log.Printf("[INFO] Updating API Gateway Account: %s", input)

	// Retry due to eventual consistency of IAM
	expectedErrMsg := "The role ARN does not have required permissions set to API Gateway"
	otherErrMsg := "API Gateway could not successfully write to CloudWatch Logs using the ARN specified"
	var out *apigateway.Account
	var err error
	err = resource.Retry(2*time.Minute, func() *resource.RetryError {
		out, err = conn.UpdateAccount(&input)

		if err != nil {
			if isAWSErr(err, "BadRequestException", expectedErrMsg) ||
				isAWSErr(err, "BadRequestException", otherErrMsg) {
				log.Printf("[DEBUG] Retrying API Gateway Account update: %s", err)
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("Updating API Gateway Account failed: %s", err)
	}
	log.Printf("[DEBUG] API Gateway Account updated: %s", out)

	d.SetId("api-gateway-account")
	return resourceAwsApiGatewayAccountRead(d, meta)
}

func resourceAwsApiGatewayAccountDelete(d *schema.ResourceData, meta interface{}) error {
	// There is no API for "deleting" account or resetting it to "default" settings
	d.SetId("")
	return nil
}
