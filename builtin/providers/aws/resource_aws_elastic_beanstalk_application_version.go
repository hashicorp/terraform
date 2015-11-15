package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"

)

func resourceAwsElasticBeanstalkApplicationVersion() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticBeanstalkApplicationVersionCreate,
		Read:   resourceAwsElasticBeanstalkApplicationVersionRead,
		Update: resourceAwsElasticBeanstalkApplicationVersionUpdate,
		Delete: resourceAwsElasticBeanstalkApplicationVersionDelete,

		Schema: map[string]*schema.Schema{
			"application": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"auto": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"bucket": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsElasticBeanstalkApplicationVersionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	application := d.Get("application").(string)
	auto := d.Get("auto").(bool)
	description := d.Get("description").(string)
	bucket := d.Get("bucket").(string)
	key := d.Get("key").(string)
	name := d.Get("name").(string)

	s3Location := elasticbeanstalk.S3Location{
		S3Bucket: aws.String(bucket),
		S3Key:    aws.String(key),
	}

	createOpts := elasticbeanstalk.CreateApplicationVersionInput{
		ApplicationName:       aws.String(application),
		AutoCreateApplication: aws.Bool(auto),
		Description:           aws.String(description),
		SourceBundle:          &s3Location,
		VersionLabel:          aws.String(name),
	}

	log.Printf("[DEBUG] Elastic Beanstalk Application Version create opts: %s", createOpts)
	_, err := conn.CreateApplicationVersion(&createOpts)
	if err != nil {
		return err
	}

	d.SetId(name)
	log.Printf("[INFO] Elastic Beanstalk Application Version Label: %s", name)

	return resourceAwsElasticBeanstalkApplicationVersionRead(d, meta)
}

func resourceAwsElasticBeanstalkApplicationVersionRead(d *schema.ResourceData, meta interface{}) error {
	beanstalkConn := meta.(*AWSClient).elasticbeanstalkconn

	name := d.Id()

	resp, err := beanstalkConn.DescribeApplicationVersions(&elasticbeanstalk.DescribeApplicationVersionsInput{
		VersionLabels: []*string{aws.String(name)},
	})

	if err != nil {
		return err
	}

	if len(resp.ApplicationVersions) == 0 {
		log.Printf("[DEBUG] Elastic Beanstalk application version read: application version not found")

		d.SetId("")

		return nil
	} else if len(resp.ApplicationVersions) != 1 {
		return fmt.Errorf("Error reading application version properties: found %d application versions, expected 1", len(resp.ApplicationVersions))
	}

	if err := d.Set("description", resp.ApplicationVersions[0].Description); err != nil {
		return err
	}

	return nil
}

func resourceAwsElasticBeanstalkApplicationVersionUpdate(d *schema.ResourceData, meta interface{}) error {
	beanstalkConn := meta.(*AWSClient).elasticbeanstalkconn

	if d.HasChange("description") {
		if err := resourceAwsElasticBeanstalkApplicationVersionDescriptionUpdate(beanstalkConn, d); err != nil {
			return err
		}
	}

	return resourceAwsElasticBeanstalkApplicationVersionRead(d, meta)

}

func resourceAwsElasticBeanstalkApplicationVersionDelete(d *schema.ResourceData, meta interface{}) error {
	beanstalkConn := meta.(*AWSClient).elasticbeanstalkconn

	application := d.Get("application").(string)
	name := d.Id()

	_, err := beanstalkConn.DeleteApplicationVersion(&elasticbeanstalk.DeleteApplicationVersionInput{
		ApplicationName: aws.String(application),
		VersionLabel:    aws.String(name),
	})

	d.SetId("")

	if awserr, ok := err.(awserr.Error); ok {
		// application version is pending delete, or no longer exists.
		if awserr.Code() == "InvalidParameterValue" {
			return nil
		}

	}
	return err
}

func resourceAwsElasticBeanstalkApplicationVersionDescriptionUpdate(beanstalkConn *elasticbeanstalk.ElasticBeanstalk, d *schema.ResourceData) error {
	application := d.Get("application").(string)
	description := d.Get("description").(string)
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Elastic Beanstalk application version: %s, update description: %s", name, description)

	_, err := beanstalkConn.UpdateApplicationVersion(&elasticbeanstalk.UpdateApplicationVersionInput{
		ApplicationName: aws.String(application),
		Description:     aws.String(description),
		VersionLabel:    aws.String(name),
	})

	return err
}
