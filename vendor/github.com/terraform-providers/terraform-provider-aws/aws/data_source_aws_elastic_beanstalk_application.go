package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsElasticBeanstalkApplication() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsElasticBeanstalkApplicationRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"appversion_lifecycle": {
				Type:     schema.TypeList,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"service_role": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"max_age_in_days": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"max_count": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"delete_source_from_s3": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAwsElasticBeanstalkApplicationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	// Get the name and description
	name := d.Get("name").(string)

	resp, err := conn.DescribeApplications(&elasticbeanstalk.DescribeApplicationsInput{
		ApplicationNames: []*string{aws.String(name)},
	})
	if err != nil {
		return fmt.Errorf("Error describing Applications (%s): %s", name, err)
	}

	if len(resp.Applications) > 1 || len(resp.Applications) < 1 {
		return fmt.Errorf("Error %d Applications matched, expected 1", len(resp.Applications))
	}

	app := resp.Applications[0]

	d.SetId(name)
	d.Set("arn", app.ApplicationArn)
	d.Set("name", app.ApplicationName)
	d.Set("description", app.Description)

	if app.ResourceLifecycleConfig != nil {
		d.Set("appversion_lifecycle", flattenResourceLifecycleConfig(app.ResourceLifecycleConfig))
	}

	return nil
}
