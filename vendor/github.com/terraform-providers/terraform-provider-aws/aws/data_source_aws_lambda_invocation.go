package aws

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsLambdaInvocation() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsLambdaInvocationRead,

		Schema: map[string]*schema.Schema{
			"function_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"qualifier": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "$LATEST",
			},

			"input": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateJsonString,
			},

			"result": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"result_map": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceAwsLambdaInvocationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	functionName := d.Get("function_name").(string)
	qualifier := d.Get("qualifier").(string)
	input := []byte(d.Get("input").(string))

	res, err := conn.Invoke(&lambda.InvokeInput{
		FunctionName:   aws.String(functionName),
		InvocationType: aws.String(lambda.InvocationTypeRequestResponse),
		Payload:        input,
		Qualifier:      aws.String(qualifier),
	})

	if err != nil {
		return err
	}

	if res.FunctionError != nil {
		return fmt.Errorf("Lambda function (%s) returned error: (%s)", functionName, string(res.Payload))
	}

	if err = d.Set("result", string(res.Payload)); err != nil {
		return err
	}

	var result map[string]interface{}

	if err = json.Unmarshal(res.Payload, &result); err != nil {
		return err
	}

	if err = d.Set("result_map", result); err != nil {
		log.Printf("[WARN] Cannot use the result invocation as a string map: %s", err)
	}

	d.SetId(fmt.Sprintf("%s_%s_%x", functionName, qualifier, md5.Sum(input)))

	return nil
}
