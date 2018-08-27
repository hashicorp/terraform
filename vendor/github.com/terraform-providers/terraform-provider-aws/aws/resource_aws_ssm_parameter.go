package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsSsmParameter() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSsmParameterPut,
		Read:   resourceAwsSsmParameterRead,
		Update: resourceAwsSsmParameterPut,
		Delete: resourceAwsSsmParameterDelete,
		Exists: resourceAwsSmmParameterExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					ssm.ParameterTypeString,
					ssm.ParameterTypeStringList,
					ssm.ParameterTypeSecureString,
				}, false),
			},
			"value": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"key_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"overwrite": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"allowed_pattern": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsSmmParameterExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ssmconn := meta.(*AWSClient).ssmconn
	_, err := ssmconn.GetParameter(&ssm.GetParameterInput{
		Name:           aws.String(d.Id()),
		WithDecryption: aws.Bool(false),
	})
	if err != nil {
		if isAWSErr(err, ssm.ErrCodeParameterNotFound, "") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func resourceAwsSsmParameterRead(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[DEBUG] Reading SSM Parameter: %s", d.Id())

	resp, err := ssmconn.GetParameter(&ssm.GetParameterInput{
		Name:           aws.String(d.Id()),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("error getting SSM parameter: %s", err)
	}

	param := resp.Parameter
	d.Set("name", param.Name)
	d.Set("type", param.Type)
	d.Set("value", param.Value)

	describeParamsInput := &ssm.DescribeParametersInput{
		ParameterFilters: []*ssm.ParameterStringFilter{
			{
				Key:    aws.String("Name"),
				Option: aws.String("Equals"),
				Values: []*string{aws.String(d.Get("name").(string))},
			},
		},
	}
	describeResp, err := ssmconn.DescribeParameters(describeParamsInput)
	if err != nil {
		return fmt.Errorf("error describing SSM parameter: %s", err)
	}

	if describeResp == nil || len(describeResp.Parameters) == 0 || describeResp.Parameters[0] == nil {
		log.Printf("[WARN] SSM Parameter %q not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	detail := describeResp.Parameters[0]
	d.Set("key_id", detail.KeyId)
	d.Set("description", detail.Description)
	d.Set("allowed_pattern", detail.AllowedPattern)

	if tagList, err := ssmconn.ListTagsForResource(&ssm.ListTagsForResourceInput{
		ResourceId:   aws.String(d.Get("name").(string)),
		ResourceType: aws.String("Parameter"),
	}); err != nil {
		return fmt.Errorf("Failed to get SSM parameter tags for %s: %s", d.Get("name"), err)
	} else {
		d.Set("tags", tagsToMapSSM(tagList.TagList))
	}

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		Service:   "ssm",
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("parameter/%s", strings.TrimPrefix(d.Id(), "/")),
	}
	d.Set("arn", arn.String())

	return nil
}

func resourceAwsSsmParameterDelete(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[INFO] Deleting SSM Parameter: %s", d.Id())

	_, err := ssmconn.DeleteParameter(&ssm.DeleteParameterInput{
		Name: aws.String(d.Get("name").(string)),
	})
	if err != nil {
		return err
	}

	return nil
}

func resourceAwsSsmParameterPut(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[INFO] Creating SSM Parameter: %s", d.Get("name").(string))

	paramInput := &ssm.PutParameterInput{
		Name:           aws.String(d.Get("name").(string)),
		Type:           aws.String(d.Get("type").(string)),
		Value:          aws.String(d.Get("value").(string)),
		Overwrite:      aws.Bool(shouldUpdateSsmParameter(d)),
		AllowedPattern: aws.String(d.Get("allowed_pattern").(string)),
	}

	if d.HasChange("description") {
		_, n := d.GetChange("description")
		paramInput.Description = aws.String(n.(string))
	}

	if keyID, ok := d.GetOk("key_id"); ok {
		log.Printf("[DEBUG] Setting key_id for SSM Parameter %v: %s", d.Get("name"), keyID)
		paramInput.SetKeyId(keyID.(string))
	}

	log.Printf("[DEBUG] Waiting for SSM Parameter %v to be updated", d.Get("name"))
	if _, err := ssmconn.PutParameter(paramInput); err != nil {
		return fmt.Errorf("error creating SSM parameter: %s", err)
	}

	if err := setTagsSSM(ssmconn, d, d.Get("name").(string), "Parameter"); err != nil {
		return fmt.Errorf("error creating SSM parameter tags: %s", err)
	}

	d.SetId(d.Get("name").(string))

	return resourceAwsSsmParameterRead(d, meta)
}

func shouldUpdateSsmParameter(d *schema.ResourceData) bool {
	// If the user has specified a preference, return their preference
	if value, ok := d.GetOkExists("overwrite"); ok {
		return value.(bool)
	}

	// Since the user has not specified a preference, obey lifecycle rules
	// if it is not a new resource, otherwise overwrite should be set to false.
	return !d.IsNewResource()
}
