// +build ignore

package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
)

func resourceAwsElasticBeanstalkConfigurationTemplate() *schema.Resource {
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
			"application": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"environment_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"option_settings": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"namespace": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"option_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: optionSettingHash,
			},
			"solution_stack_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func optionSettingHash(v interface{}) int {
	rd := v.(*schema.ResourceData)
	namespace := rd.Get("namespace").(string)
	optionName := rd.Get("option_name").(string)
	return hashcode.String(fmt.Sprintf("%s.%s", namespace, optionName))
}

func gatherOptionSettings(d *schema.ResourceData) []*elasticbeanstalk.ConfigurationOptionSetting {
	optionSettingsSet, ok := d.Get("option_settings").(*schema.Set)
	if !ok || optionSettingsSet == nil {
		optionSettingsSet = new(schema.Set)
	}

	return extractOptionSettings(optionSettingsSet)
}

func extractOptionSettings(s *schema.Set) []*elasticbeanstalk.ConfigurationOptionSetting {
	var options []*elasticbeanstalk.ConfigurationOptionSetting

	for _, elem := range s.List() {
		rd := elem.(*schema.ResourceData)
		opt := &elasticbeanstalk.ConfigurationOptionSetting{
			Namespace:  aws.String(rd.Get("namespace").(string)),
			OptionName: aws.String(rd.Get("option_name").(string)),
			Value:      aws.String(rd.Get("value").(string)),
		}
		options = append(options, opt)
	}

	return options
}

func resourceAwsElasticBeanstalkConfigurationTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	beanstalkConn := meta.(*AWSClient).elasticbeanstalkconn

	// Get the relevant properties
	name := d.Get("name").(string)
	appName := d.Get("application").(string)
	description := d.Get("description").(string)
	envId := d.Get("environment_id").(string)
	optionSettings := gatherOptionSettings(d)
	solutionStackName := d.Get("solution_stack_name").(string)

	_, err := beanstalkConn.ValidateConfigurationSettings(&elasticbeanstalk.ValidateConfigurationSettingsInput{
		ApplicationName: aws.String(appName),
		OptionSettings:  optionSettings,
	})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Elastic Beanstalk configuration template create: %s", name)

	req := &elasticbeanstalk.CreateConfigurationTemplateInput{
		ApplicationName:   aws.String(appName),
		Description:       aws.String(description),
		EnvironmentID:     aws.String(envId),
		OptionSettings:    optionSettings,
		SolutionStackName: aws.String(solutionStackName),
		TemplateName:      aws.String(name),
	}

	if _, err := beanstalkConn.CreateConfigurationTemplate(req); err != nil {
		return fmt.Errorf("Error creating Elastic Beanstalk application")
	}

	// Assign the bucket name as the resource ID
	d.SetId(appName + "-" + name)

	return resourceAwsS3BucketUpdate(d, meta)
}

func resourceAwsElasticBeanstalkConfigurationTemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	beanstalkConn := meta.(*AWSClient).elasticbeanstalkconn

	log.Printf("[DEBUG] Elastic Beanstalk configuration template update: %s", d.Get("name").(string))

	if d.HasChange("description") {
		if err := resourceAwsElasticBeanstalkConfigurationTemplateDescriptionUpdate(beanstalkConn, d); err != nil {
			return err
		}
	}

	if d.HasChange("option_settings") {
		if err := resourceAwsElasticBeanstalkConfigurationTemplateOptionSettingsUpdate(beanstalkConn, d); err != nil {
			return err
		}
	}

	return resourceAwsElasticBeanstalkConfigurationTemplateRead(d, meta)
}

func resourceAwsElasticBeanstalkConfigurationTemplateDescriptionUpdate(beanstalkConn *elasticbeanstalk.ElasticBeanstalk, d *schema.ResourceData) error {
	_, err := beanstalkConn.UpdateConfigurationTemplate(&elasticbeanstalk.UpdateConfigurationTemplateInput{
		ApplicationName: aws.String(d.Get("application").(string)),
		TemplateName:    aws.String(d.Get("name").(string)),
		Description:     aws.String(d.Get("description").(string)),
	})

	return err
}

func resourceAwsElasticBeanstalkConfigurationTemplateOptionSettingsUpdate(beanstalkConn *elasticbeanstalk.ElasticBeanstalk, d *schema.ResourceData) error {
	if d.HasChange("option_settings") {
		_, err := beanstalkConn.ValidateConfigurationSettings(&elasticbeanstalk.ValidateConfigurationSettingsInput{
			ApplicationName: aws.String(d.Get("application").(string)),
			TemplateName:    aws.String(d.Get("name").(string)),
			OptionSettings:  gatherOptionSettings(d),
		})
		if err != nil {
			return err
		}

		o, n := d.GetChange("option_settings")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := o.(*schema.Set)

		remove := extractOptionSettings(os.Difference(ns))
		add := extractOptionSettings(ns.Difference(os))

		req := &elasticbeanstalk.UpdateConfigurationTemplateInput{
			ApplicationName: aws.String(d.Get("application").(string)),
			TemplateName:    aws.String(d.Get("name").(string)),
			OptionSettings:  add,
		}

		for _, elem := range remove {
			req.OptionsToRemove = append(req.OptionsToRemove, &elasticbeanstalk.OptionSpecification{
				Namespace:  elem.Namespace,
				OptionName: elem.OptionName,
			})
		}

		if _, err := beanstalkConn.UpdateConfigurationTemplate(req); err != nil {
			return err
		}
	}

	return nil
}

func resourceAwsElasticBeanstalkConfigurationTemplateRead(d *schema.ResourceData, meta interface{}) error {
	beanstalkConn := meta.(*AWSClient).elasticbeanstalkconn

	log.Printf("[DEBUG] Elastic Beanstalk configuration template read: %s", d.Get("name").(string))

	resp, err := beanstalkConn.DescribeConfigurationSettings(&elasticbeanstalk.DescribeConfigurationSettingsInput{
		ApplicationName: aws.String(d.Id()),
		TemplateName:    aws.String(d.Get("name").(string)),
	})

	if err != nil {
		return err
	}

	// if len(resp.ConfigurationSettings) > 1 {

	// settings := make(map[string]map[string]string)
	// for _, setting := range resp.ConfigurationSettings {
	//   k := fmt.Sprintf("%s.%s", setting.)
	// }
	// }

	if len(resp.Applications) != 1 {
		log.Printf("[DEBUG] Elastic Beanstalk unexpected describe applications response: %+v", resp)
		return fmt.Errorf("Error reading application properties: found %d applications, expected 1", len(resp.Applications))
	}

	return d.Set("description", resp.Applications[0].Description)
}

func resourceAwsElasticBeanstalkApplicationDelete(d *schema.ResourceData, meta interface{}) error {
	beanstalkConn := meta.(*AWSClient).elasticbeanstalkconn

	_, err := beanstalkConn.DeleteApplication(&elasticbeanstalk.DeleteApplicationInput{
		ApplicationName: aws.String(d.Id()),
	})

	return err
}
