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
		return err
	}

	// if len(resp.ConfigurationSettings) > 1 {

	// settings := make(map[string]map[string]string)
	// for _, setting := range resp.ConfigurationSettings {
	//   k := fmt.Sprintf("%s.%s", setting.)
	// }
	// }

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

	if d.HasChange("option_settings") {
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
	if d.HasChange("option_settings") {
		_, err := conn.ValidateConfigurationSettings(&elasticbeanstalk.ValidateConfigurationSettingsInput{
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

		if _, err := conn.UpdateConfigurationTemplate(req); err != nil {
			return err
		}
	}

	return nil
}

func resourceAwsElasticBeanstalkConfigurationTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	_, err := conn.DeleteApplication(&elasticbeanstalk.DeleteApplicationInput{
		ApplicationName: aws.String(d.Id()),
	})

	return err
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
