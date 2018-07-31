package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsS3BucketMetric() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsS3BucketMetricPut,
		Read:   resourceAwsS3BucketMetricRead,
		Update: resourceAwsS3BucketMetricPut,
		Delete: resourceAwsS3BucketMetricDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"filter": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"prefix": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"tags": tagsSchema(),
					},
				},
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsS3BucketMetricPut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).s3conn
	bucket := d.Get("bucket").(string)
	name := d.Get("name").(string)

	metricsConfiguration := &s3.MetricsConfiguration{
		Id: aws.String(name),
	}

	if v, ok := d.GetOk("filter"); ok {
		filterList := v.([]interface{})
		filterMap := filterList[0].(map[string]interface{})
		metricsConfiguration.Filter = expandS3MetricsFilter(filterMap)
	}

	input := &s3.PutBucketMetricsConfigurationInput{
		Bucket:               aws.String(bucket),
		Id:                   aws.String(name),
		MetricsConfiguration: metricsConfiguration,
	}

	log.Printf("[DEBUG] Putting metric configuration: %s", input)
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := conn.PutBucketMetricsConfiguration(input)
		if err != nil {
			if isAWSErr(err, s3.ErrCodeNoSuchBucket, "") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Error putting S3 metric configuration: %s", err)
	}

	d.SetId(fmt.Sprintf("%s:%s", bucket, name))

	return resourceAwsS3BucketMetricRead(d, meta)
}

func resourceAwsS3BucketMetricDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).s3conn

	bucket, name, err := resourceAwsS3BucketMetricParseID(d.Id())
	if err != nil {
		return err
	}

	input := &s3.DeleteBucketMetricsConfigurationInput{
		Bucket: aws.String(bucket),
		Id:     aws.String(name),
	}

	log.Printf("[DEBUG] Deleting S3 bucket metric configuration: %s", input)
	_, err = conn.DeleteBucketMetricsConfiguration(input)
	if err != nil {
		if isAWSErr(err, s3.ErrCodeNoSuchBucket, "") || isAWSErr(err, "NoSuchConfiguration", "The specified configuration does not exist.") {
			return nil
		}
		return fmt.Errorf("Error deleting S3 metric configuration: %s", err)
	}

	return nil
}

func resourceAwsS3BucketMetricRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).s3conn

	bucket, name, err := resourceAwsS3BucketMetricParseID(d.Id())
	if err != nil {
		return err
	}

	d.Set("bucket", bucket)
	d.Set("name", name)

	input := &s3.GetBucketMetricsConfigurationInput{
		Bucket: aws.String(bucket),
		Id:     aws.String(name),
	}

	log.Printf("[DEBUG] Reading S3 bucket metrics configuration: %s", input)
	output, err := conn.GetBucketMetricsConfiguration(input)
	if err != nil {
		if isAWSErr(err, s3.ErrCodeNoSuchBucket, "") || isAWSErr(err, "NoSuchConfiguration", "The specified configuration does not exist.") {
			log.Printf("[WARN] %s S3 bucket metrics configuration not found, removing from state.", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if output.MetricsConfiguration.Filter != nil {
		if err := d.Set("filter", []interface{}{flattenS3MetricsFilter(output.MetricsConfiguration.Filter)}); err != nil {
			return err
		}
	}

	return nil
}

func expandS3MetricsFilter(m map[string]interface{}) *s3.MetricsFilter {
	var prefix string
	if v, ok := m["prefix"]; ok {
		prefix = v.(string)
	}

	var tags []*s3.Tag
	if v, ok := m["tags"]; ok {
		tags = tagsFromMapS3(v.(map[string]interface{}))
	}

	metricsFilter := &s3.MetricsFilter{}
	if prefix != "" && len(tags) > 0 {
		metricsFilter.And = &s3.MetricsAndOperator{
			Prefix: aws.String(prefix),
			Tags:   tags,
		}
	} else if len(tags) > 1 {
		metricsFilter.And = &s3.MetricsAndOperator{
			Tags: tags,
		}
	} else if len(tags) == 1 {
		metricsFilter.Tag = tags[0]
	} else {
		metricsFilter.Prefix = aws.String(prefix)
	}
	return metricsFilter
}

func flattenS3MetricsFilter(metricsFilter *s3.MetricsFilter) map[string]interface{} {
	m := make(map[string]interface{})

	if metricsFilter.And != nil {
		and := *metricsFilter.And
		if and.Prefix != nil {
			m["prefix"] = *and.Prefix
		}
		if and.Tags != nil {
			m["tags"] = tagsToMapS3(and.Tags)
		}
	} else if metricsFilter.Prefix != nil {
		m["prefix"] = *metricsFilter.Prefix
	} else if metricsFilter.Tag != nil {
		tags := []*s3.Tag{
			metricsFilter.Tag,
		}
		m["tags"] = tagsToMapS3(tags)
	}
	return m
}

func resourceAwsS3BucketMetricParseID(id string) (string, string, error) {
	idParts := strings.Split(id, ":")
	if len(idParts) != 2 {
		return "", "", fmt.Errorf("please make sure the ID is in the form BUCKET:NAME (i.e. my-bucket:EntireBucket")
	}
	bucket := idParts[0]
	name := idParts[1]
	return bucket, name, nil
}
