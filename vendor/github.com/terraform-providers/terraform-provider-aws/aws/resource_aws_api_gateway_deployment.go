package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsApiGatewayDeployment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayDeploymentCreate,
		Read:   resourceAwsApiGatewayDeploymentRead,
		Update: resourceAwsApiGatewayDeploymentUpdate,
		Delete: resourceAwsApiGatewayDeploymentDelete,

		Schema: map[string]*schema.Schema{
			"rest_api_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"stage_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"stage_description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"variables": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"created_date": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"invoke_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"execution_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsApiGatewayDeploymentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	// Create the gateway
	log.Printf("[DEBUG] Creating API Gateway Deployment")

	variables := make(map[string]string)
	for k, v := range d.Get("variables").(map[string]interface{}) {
		variables[k] = v.(string)
	}

	var err error
	deployment, err := conn.CreateDeployment(&apigateway.CreateDeploymentInput{
		RestApiId:        aws.String(d.Get("rest_api_id").(string)),
		StageName:        aws.String(d.Get("stage_name").(string)),
		Description:      aws.String(d.Get("description").(string)),
		StageDescription: aws.String(d.Get("stage_description").(string)),
		Variables:        aws.StringMap(variables),
	})
	if err != nil {
		return fmt.Errorf("Error creating API Gateway Deployment: %s", err)
	}

	d.SetId(*deployment.Id)
	log.Printf("[DEBUG] API Gateway Deployment ID: %s", d.Id())

	return resourceAwsApiGatewayDeploymentRead(d, meta)
}

func resourceAwsApiGatewayDeploymentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Reading API Gateway Deployment %s", d.Id())
	restApiId := d.Get("rest_api_id").(string)
	out, err := conn.GetDeployment(&apigateway.GetDeploymentInput{
		RestApiId:    aws.String(restApiId),
		DeploymentId: aws.String(d.Id()),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			log.Printf("[WARN] API Gateway Deployment (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	log.Printf("[DEBUG] Received API Gateway Deployment: %s", out)
	d.Set("description", out.Description)

	region := meta.(*AWSClient).region
	stageName := d.Get("stage_name").(string)

	d.Set("invoke_url", buildApiGatewayInvokeURL(restApiId, region, stageName))

	executionArn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "execute-api",
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("%s/%s", restApiId, stageName),
	}.String()
	d.Set("execution_arn", executionArn)

	if err := d.Set("created_date", out.CreatedDate.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Error setting created_date: %s", err)
	}

	return nil
}

func resourceAwsApiGatewayDeploymentUpdateOperations(d *schema.ResourceData) []*apigateway.PatchOperation {
	operations := make([]*apigateway.PatchOperation, 0)

	if d.HasChange("description") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/description"),
			Value: aws.String(d.Get("description").(string)),
		})
	}

	return operations
}

func resourceAwsApiGatewayDeploymentUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Updating API Gateway API Key: %s", d.Id())

	_, err := conn.UpdateDeployment(&apigateway.UpdateDeploymentInput{
		DeploymentId:    aws.String(d.Id()),
		RestApiId:       aws.String(d.Get("rest_api_id").(string)),
		PatchOperations: resourceAwsApiGatewayDeploymentUpdateOperations(d),
	})
	if err != nil {
		return err
	}

	return resourceAwsApiGatewayDeploymentRead(d, meta)
}

func resourceAwsApiGatewayDeploymentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Deleting API Gateway Deployment: %s", d.Id())

	// If the stage has been updated to point at a different deployment, then
	// the stage should not be removed when this deployment is deleted.
	shouldDeleteStage := false

	// API Gateway allows an empty state name (e.g. ""), but the AWS Go SDK
	// now has validation for the parameter, so we must check first.
	// InvalidParameter: 1 validation error(s) found.
	//  - minimum field size of 1, GetStageInput.StageName.
	stageName := d.Get("stage_name").(string)
	if stageName != "" {
		stage, err := conn.GetStage(&apigateway.GetStageInput{
			StageName: aws.String(stageName),
			RestApiId: aws.String(d.Get("rest_api_id").(string)),
		})

		if err != nil && !isAWSErr(err, apigateway.ErrCodeNotFoundException, "") {
			return fmt.Errorf("error getting referenced stage: %s", err)
		}

		if stage != nil && aws.StringValue(stage.DeploymentId) == d.Id() {
			shouldDeleteStage = true
		}
	}

	if shouldDeleteStage {
		if _, err := conn.DeleteStage(&apigateway.DeleteStageInput{
			StageName: aws.String(d.Get("stage_name").(string)),
			RestApiId: aws.String(d.Get("rest_api_id").(string)),
		}); err == nil {
			return nil
		}
	}

	_, err := conn.DeleteDeployment(&apigateway.DeleteDeploymentInput{
		DeploymentId: aws.String(d.Id()),
		RestApiId:    aws.String(d.Get("rest_api_id").(string)),
	})

	if isAWSErr(err, apigateway.ErrCodeNotFoundException, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting API Gateway Deployment (%s): %s", d.Id(), err)
	}

	return nil
}
