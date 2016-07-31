package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
)

func resourceAwsElasticBeanstalkConfigurationTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticBeanstalkConfigurationTemplateCreate,
		Read:   resourceAwsElasticBeanstalkConfigurationTemplateRead,
		Update: resourceAwsElasticBeanstalkConfigurationTemplateUpdate,
		Delete: resourceAwsElasticBeanstalkConfigurationTemplateDelete,

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
			"setting": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     resourceAwsElasticBeanstalkOptionSetting(),
				Set:      optionSettingValueHash,
			},
			"solution_stack_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsElasticBeanstalkConfigurationTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	// Get the relevant properties
	name := d.Get("name").(string)
	appName := d.Get("application").(string)

	optionSettings := gatherOptionSettings(d)

	opts := elasticbeanstalk.CreateConfigurationTemplateInput{
		ApplicationName: aws.String(appName),
		TemplateName:    aws.String(name),
		OptionSettings:  optionSettings,
	}

	if attr, ok := d.GetOk("description"); ok {
		opts.Description = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("environment_id"); ok {
		opts.EnvironmentId = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("solution_stack_name"); ok {
		opts.SolutionStackName = aws.String(attr.(string))
	}

	log.Printf("[DEBUG] Elastic Beanstalk configuration template create opts: %s", opts)
	if _, err := conn.CreateConfigurationTemplate(&opts); err != nil {
		return fmt.Errorf("Error creating Elastic Beanstalk configuration template: %s", err)
	}

	d.SetId(name)

	return resourceAwsElasticBeanstalkConfigurationTemplateRead(d, meta)
}

func resourceAwsElasticBeanstalkConfigurationTemplateRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	log.Printf("[DEBUG] Elastic Beanstalk configuration template read: %s", d.Get("name").(string))

	resp, err := conn.DescribeConfigurationSettings(&elasticbeanstalk.DescribeConfigurationSettingsInput{
		TemplateName:    aws.String(d.Id()),
		ApplicationName: aws.String(d.Get("application").(string)),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "InvalidParameterValue" && strings.Contains(awsErr.Message(), "No Configuration Template named") {
				log.Printf("[WARN] No Configuration Template named (%s) found", d.Id())
				d.SetId("")
				return nil
			}
		}
		return err
	}

	if len(resp.ConfigurationSettings) != 1 {
		log.Printf("[DEBUG] Elastic Beanstalk unexpected describe configuration template response: %+v", resp)
		return fmt.Errorf("Error reading application properties: found %d applications, expected 1", len(resp.ConfigurationSettings))
	}

	d.Set("description", resp.ConfigurationSettings[0].Description)
	return nil
}

func resourceAwsElasticBeanstalkConfigurationTemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	log.Printf("[DEBUG] Elastic Beanstalk configuration template update: %s", d.Get("name").(string))

	if d.HasChange("description") {
		if err := resourceAwsElasticBeanstalkConfigurationTemplateDescriptionUpdate(conn, d); err != nil {
			return err
		}
	}

	if d.HasChange("setting") {
		if err := resourceAwsElasticBeanstalkConfigurationTemplateOptionSettingsUpdate(conn, d); err != nil {
			return err
		}
	}

	return resourceAwsElasticBeanstalkConfigurationTemplateRead(d, meta)
}

func resourceAwsElasticBeanstalkConfigurationTemplateDescriptionUpdate(conn *elasticbeanstalk.ElasticBeanstalk, d *schema.ResourceData) error {
	_, err := conn.UpdateConfigurationTemplate(&elasticbeanstalk.UpdateConfigurationTemplateInput{
		ApplicationName: aws.String(d.Get("application").(string)),
		TemplateName:    aws.String(d.Get("name").(string)),
		Description:     aws.String(d.Get("description").(string)),
	})

	return err
}

func resourceAwsElasticBeanstalkConfigurationTemplateOptionSettingsUpdate(conn *elasticbeanstalk.ElasticBeanstalk, d *schema.ResourceData) error {
	if d.HasChange("setting") {
		_, err := conn.ValidateConfigurationSettings(&elasticbeanstalk.ValidateConfigurationSettingsInput{
			ApplicationName: aws.String(d.Get("application").(string)),
			TemplateName:    aws.String(d.Get("name").(string)),
			OptionSettings:  gatherOptionSettings(d),
		})
		if err != nil {
			return err
		}

		o, n := d.GetChange("setting")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		rm := extractOptionSettings(os.Difference(ns))
		add := extractOptionSettings(ns.Difference(os))

		// Additions and removals of options are done in a single API call, so we
		// can't do our normal "remove these" and then later "add these", re-adding
		// any updated settings.
		// Because of this, we need to remove any settings in the "removable"
		// settings that are also found in the "add" settings, otherwise they
		// conflict. Here we loop through all the initial removables from the set
		// difference, and we build up a slice of settings not found in the "add"
		// set
		var remove []*elasticbeanstalk.ConfigurationOptionSetting
		for _, r := range rm {
			for _, a := range add {
				if *r.Namespace == *a.Namespace && *r.OptionName == *a.OptionName {
					continue
				}
				remove = append(remove, r)
			}
		}

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

		log.Printf("[DEBUG] Update Configuration Template request: %s", req)
		if _, err := conn.UpdateConfigurationTemplate(req); err != nil {
			return err
		}
	}

	return nil
}

func resourceAwsElasticBeanstalkConfigurationTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	application := d.Get("application").(string)

	_, err := conn.DeleteConfigurationTemplate(&elasticbeanstalk.DeleteConfigurationTemplateInput{
		ApplicationName: aws.String(application),
		TemplateName:    aws.String(d.Id()),
	})

	return err
}

func gatherOptionSettings(d *schema.ResourceData) []*elasticbeanstalk.ConfigurationOptionSetting {
	optionSettingsSet, ok := d.Get("setting").(*schema.Set)
	if !ok || optionSettingsSet == nil {
		optionSettingsSet = new(schema.Set)
	}

	return extractOptionSettings(optionSettingsSet)
}
