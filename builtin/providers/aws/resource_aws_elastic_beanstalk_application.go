package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/hashicorp/terraform/helper/resource"
)

func resourceAwsElasticBeanstalkApplication() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticBeanstalkApplicationCreate,
		Read:   resourceAwsElasticBeanstalkApplicationRead,
		Update: resourceAwsElasticBeanstalkApplicationUpdate,
		Delete: resourceAwsElasticBeanstalkApplicationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
		},
	}
}

func resourceAwsElasticBeanstalkApplicationCreate(d *schema.ResourceData, meta interface{}) error {
	beanstalkConn := meta.(*AWSClient).elasticbeanstalkconn

	// Get the name and description
	name := d.Get("name").(string)
	description := d.Get("description").(string)

	log.Printf("[DEBUG] Elastic Beanstalk application create: %s, description: %s", name, description)

	req := &elasticbeanstalk.CreateApplicationInput{
		ApplicationName: aws.String(name),
		Description:     aws.String(description),
	}

	_, err := beanstalkConn.CreateApplication(req)
	if err != nil {
		return err
	}

	d.SetId(name)

	return resourceAwsElasticBeanstalkApplicationRead(d, meta)
}

func resourceAwsElasticBeanstalkApplicationUpdate(d *schema.ResourceData, meta interface{}) error {
	beanstalkConn := meta.(*AWSClient).elasticbeanstalkconn

	if d.HasChange("description") {
		if err := resourceAwsElasticBeanstalkApplicationDescriptionUpdate(beanstalkConn, d); err != nil {
			return err
		}
	}

	return resourceAwsElasticBeanstalkApplicationRead(d, meta)
}

func resourceAwsElasticBeanstalkApplicationDescriptionUpdate(beanstalkConn *elasticbeanstalk.ElasticBeanstalk, d *schema.ResourceData) error {
	name := d.Get("name").(string)
	description := d.Get("description").(string)

	log.Printf("[DEBUG] Elastic Beanstalk application: %s, update description: %s", name, description)

	_, err := beanstalkConn.UpdateApplication(&elasticbeanstalk.UpdateApplicationInput{
		ApplicationName: aws.String(name),
		Description:     aws.String(description),
	})

	return err
}

func resourceAwsElasticBeanstalkApplicationRead(d *schema.ResourceData, meta interface{}) error {
	a, err := getBeanstalkApplication(d, meta)
	if err != nil {
		return err
	}
	if a == nil {
		return err
	}

	d.Set("name", a.ApplicationName)
	d.Set("description", a.Description)
	return nil
}

func resourceAwsElasticBeanstalkApplicationDelete(d *schema.ResourceData, meta interface{}) error {
	beanstalkConn := meta.(*AWSClient).elasticbeanstalkconn

	a, err := getBeanstalkApplication(d, meta)
	if err != nil {
		return err
	}
	_, err = beanstalkConn.DeleteApplication(&elasticbeanstalk.DeleteApplicationInput{
		ApplicationName: aws.String(d.Id()),
	})

	return resource.Retry(10*time.Second, func() *resource.RetryError {
		if a, _ = getBeanstalkApplication(d, meta); a != nil {
			return resource.RetryableError(
				fmt.Errorf("Beanstalk Application still exists"))
		}
		return nil
	})
}

func getBeanstalkApplication(
	d *schema.ResourceData,
	meta interface{}) (*elasticbeanstalk.ApplicationDescription, error) {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	resp, err := conn.DescribeApplications(&elasticbeanstalk.DescribeApplicationsInput{
		ApplicationNames: []*string{aws.String(d.Id())},
	})

	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() != "InvalidBeanstalkAppID.NotFound" {
			log.Printf("[Err] Error reading Elastic Beanstalk Application (%s): Application not found", d.Id())
			d.SetId("")
			return nil, nil
		}
		return nil, err
	}

	switch {
	case len(resp.Applications) > 1:
		return nil, fmt.Errorf("Error %d Applications matched, expected 1", len(resp.Applications))
	case len(resp.Applications) == 0:
		d.SetId("")
		return nil, nil
	default:
		return resp.Applications[0], nil
	}
}
