package aws

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	arn2 "github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

const awsMutexLambdaLayerKey = `aws_lambda_layer_version`

func resourceAwsLambdaLayerVersion() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLambdaLayerVersionPublish,
		Read:   resourceAwsLambdaLayerVersionRead,
		Delete: resourceAwsLambdaLayerVersionDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"layer_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"filename": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"s3_bucket", "s3_key", "s3_object_version"},
			},
			"s3_bucket": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"filename"},
			},
			"s3_key": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"filename"},
			},
			"s3_object_version": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"filename"},
			},
			"compatible_runtimes": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				MinItems: 0,
				MaxItems: 5,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(validLambdaRuntimes, false),
				},
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"license_info": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(0, 512),
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"layer_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"source_code_hash": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"source_code_size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"version": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsLambdaLayerVersionPublish(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	layerName := d.Get("layer_name").(string)
	filename, hasFilename := d.GetOk("filename")
	s3Bucket, bucketOk := d.GetOk("s3_bucket")
	s3Key, keyOk := d.GetOk("s3_key")
	s3ObjectVersion, versionOk := d.GetOk("s3_object_version")

	if !hasFilename && !bucketOk && !keyOk && !versionOk {
		return errors.New("filename or s3_* attributes must be set")
	}

	var layerContent *lambda.LayerVersionContentInput
	if hasFilename {
		awsMutexKV.Lock(awsMutexLambdaLayerKey)
		defer awsMutexKV.Unlock(awsMutexLambdaLayerKey)
		file, err := loadFileContent(filename.(string))
		if err != nil {
			return fmt.Errorf("Unable to load %q: %s", filename.(string), err)
		}
		layerContent = &lambda.LayerVersionContentInput{
			ZipFile: file,
		}
	} else {
		if !bucketOk || !keyOk {
			return errors.New("s3_bucket and s3_key must all be set while using s3 code source")
		}
		layerContent = &lambda.LayerVersionContentInput{
			S3Bucket: aws.String(s3Bucket.(string)),
			S3Key:    aws.String(s3Key.(string)),
		}
		if versionOk {
			layerContent.S3ObjectVersion = aws.String(s3ObjectVersion.(string))
		}
	}

	params := &lambda.PublishLayerVersionInput{
		Content:     layerContent,
		Description: aws.String(d.Get("description").(string)),
		LayerName:   aws.String(layerName),
		LicenseInfo: aws.String(d.Get("license_info").(string)),
	}

	if v, ok := d.GetOk("compatible_runtimes"); ok && v.(*schema.Set).Len() > 0 {
		params.CompatibleRuntimes = expandStringList(v.(*schema.Set).List())
	}

	log.Printf("[DEBUG] Publishing Lambda layer: %s", params)
	result, err := conn.PublishLayerVersion(params)
	if err != nil {
		return fmt.Errorf("Error creating lambda layer: %s", err)
	}

	d.SetId(aws.StringValue(result.LayerVersionArn))
	return resourceAwsLambdaLayerVersionRead(d, meta)
}

func resourceAwsLambdaLayerVersionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	layerName, version, err := resourceAwsLambdaLayerVersionParseId(d.Id())
	if err != nil {
		return fmt.Errorf("Error parsing lambda layer ID: %s", err)
	}

	layerVersion, err := conn.GetLayerVersion(&lambda.GetLayerVersionInput{
		LayerName:     aws.String(layerName),
		VersionNumber: aws.Int64(version),
	})

	if isAWSErr(err, lambda.ErrCodeResourceNotFoundException, "") {
		log.Printf("[WARN] Lambda Layer Version (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading Lambda Layer version (%s): %s", d.Id(), err)
	}

	if err := d.Set("layer_name", layerName); err != nil {
		return fmt.Errorf("Error setting lambda layer name: %s", err)
	}
	if err := d.Set("version", strconv.FormatInt(version, 10)); err != nil {
		return fmt.Errorf("Error setting lambda layer version: %s", err)
	}
	if err := d.Set("arn", layerVersion.LayerArn); err != nil {
		return fmt.Errorf("Error setting lambda layer arn: %s", err)
	}
	if err := d.Set("layer_arn", layerVersion.LayerVersionArn); err != nil {
		return fmt.Errorf("Error setting lambda layer qualified arn: %s", err)
	}
	if err := d.Set("description", layerVersion.Description); err != nil {
		return fmt.Errorf("Error setting lambda layer description: %s", err)
	}
	if err := d.Set("license_info", layerVersion.LicenseInfo); err != nil {
		return fmt.Errorf("Error setting lambda layer license info: %s", err)
	}
	if err := d.Set("created_date", layerVersion.CreatedDate); err != nil {
		return fmt.Errorf("Error setting lambda layer created date: %s", err)
	}
	if err := d.Set("source_code_hash", layerVersion.Content.CodeSha256); err != nil {
		return fmt.Errorf("Error setting lambda layer source code hash: %s", err)
	}
	if err := d.Set("source_code_size", layerVersion.Content.CodeSize); err != nil {
		return fmt.Errorf("Error setting lambda layer source code size: %s", err)
	}
	if err := d.Set("compatible_runtimes", flattenStringList(layerVersion.CompatibleRuntimes)); err != nil {
		return fmt.Errorf("Error setting lambda layer compatible runtimes: %s", err)
	}

	return nil
}

func resourceAwsLambdaLayerVersionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	version, err := strconv.ParseInt(d.Get("version").(string), 10, 64)
	if err != nil {
		return fmt.Errorf("Error parsing lambda layer version: %s", err)
	}

	_, err = conn.DeleteLayerVersion(&lambda.DeleteLayerVersionInput{
		LayerName:     aws.String(d.Get("layer_name").(string)),
		VersionNumber: aws.Int64(version),
	})
	if err != nil {
		return fmt.Errorf("error deleting Lambda Layer Version (%s): %s", d.Id(), err)
	}

	log.Printf("[DEBUG] Lambda layer %q deleted", d.Get("arn").(string))
	return nil
}

func resourceAwsLambdaLayerVersionParseId(id string) (layerName string, version int64, err error) {
	arn, err := arn2.Parse(id)
	if err != nil {
		return
	}
	parts := strings.Split(arn.Resource, ":")
	if len(parts) != 3 || parts[0] != "layer" {
		err = fmt.Errorf("lambda_layer ID must be a valid Layer ARN")
		return
	}

	layerName = parts[1]
	version, err = strconv.ParseInt(parts[2], 10, 64)
	return
}
