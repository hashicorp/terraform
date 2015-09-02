package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
)

func resourceAwsElasticBeanstalkApplication() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticBeanstalkApplicationCreate,
		Read:   resourceAwsElasticBeanstalkApplicationRead,
		Update: resourceAwsElasticBeanstalkApplicationUpdate,
		Delete: resourceAwsElasticBeanstalkApplicationDelete,

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
	beanstalkConn := meta.(*AWSClient).elasticbeanstalkconn

	resp, err := beanstalkConn.DescribeApplications(&elasticbeanstalk.DescribeApplicationsInput{
		ApplicationNames: []*string{aws.String(d.Id())},
	})

	if err != nil {
		return err
	}

	if len(resp.Applications) == 0 {
		log.Printf("[DEBUG] Elastic Beanstalk application read: application not found")

		d.SetId("")

		return nil
	} else if len(resp.Applications) != 1 {
		return fmt.Errorf("Error reading application properties: found %d applications, expected 1", len(resp.Applications))
	}

	if err := d.Set("description", resp.Applications[0].Description); err != nil {
		return err
	}

	return nil
}

func resourceAwsElasticBeanstalkApplicationDelete(d *schema.ResourceData, meta interface{}) error {
	beanstalkConn := meta.(*AWSClient).elasticbeanstalkconn

	_, err := beanstalkConn.DeleteApplication(&elasticbeanstalk.DeleteApplicationInput{
		ApplicationName: aws.String(d.Id()),
	})

	d.SetId("")

	return err
}
