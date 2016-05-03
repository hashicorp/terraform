package aws

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
)

func resourceAwsElasticBeanstalkOptionSetting() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"namespace": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"value": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsElasticBeanstalkEnvironment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticBeanstalkEnvironmentCreate,
		Read:   resourceAwsElasticBeanstalkEnvironmentRead,
		Update: resourceAwsElasticBeanstalkEnvironmentUpdate,
		Delete: resourceAwsElasticBeanstalkEnvironmentDelete,

		SchemaVersion: 1,
		MigrateState:  resourceAwsElasticBeanstalkEnvironmentMigrateState,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"application": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"cname": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"cname_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},
			"tier": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "WebServer",
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					switch value {
					case
						"Worker",
						"WebServer":
						return
					}
					errors = append(errors, fmt.Errorf("%s is not a valid tier. Valid options are WebServer or Worker", value))
					return
				},
				ForceNew: true,
			},
			"setting": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     resourceAwsElasticBeanstalkOptionSetting(),
				Set:      optionSettingValueHash,
			},
			"all_settings": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     resourceAwsElasticBeanstalkOptionSetting(),
				Set:      optionSettingValueHash,
			},
			"solution_stack_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"template_name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"solution_stack_name"},
			},
			"wait_for_ready_timeout": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "10m",
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					duration, err := time.ParseDuration(value)
					if err != nil {
						errors = append(errors, fmt.Errorf(
							"%q cannot be parsed as a duration: %s", k, err))
					}
					if duration < 0 {
						errors = append(errors, fmt.Errorf(
							"%q must be greater than zero", k))
					}
					return
				},
			},
			"autoscaling_groups": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"instances": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"launch_configurations": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"load_balancers": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"queues": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"triggers": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsElasticBeanstalkEnvironmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	// Get values from config
	name := d.Get("name").(string)
	cnamePrefix := d.Get("cname_prefix").(string)
	tier := d.Get("tier").(string)
	app := d.Get("application").(string)
	desc := d.Get("description").(string)
	settings := d.Get("setting").(*schema.Set)
	solutionStack := d.Get("solution_stack_name").(string)
	templateName := d.Get("template_name").(string)
	waitForReadyTimeOut, err := time.ParseDuration(d.Get("wait_for_ready_timeout").(string))
	if err != nil {
		return err
	}

	// TODO set tags
	// Note: at time of writing, you cannot view or edit Tags after creation
	// d.Set("tags", tagsToMap(instance.Tags))
	createOpts := elasticbeanstalk.CreateEnvironmentInput{
		EnvironmentName: aws.String(name),
		ApplicationName: aws.String(app),
		OptionSettings:  extractOptionSettings(settings),
		Tags:            tagsFromMapBeanstalk(d.Get("tags").(map[string]interface{})),
	}

	if desc != "" {
		createOpts.Description = aws.String(desc)
	}

	if cnamePrefix != "" {
		if tier != "WebServer" {
			return fmt.Errorf("Cannont set cname_prefix for tier: %s.", tier)
		}
		createOpts.CNAMEPrefix = aws.String(cnamePrefix)
	}

	if tier != "" {
		var tierType string

		switch tier {
		case "WebServer":
			tierType = "Standard"
		case "Worker":
			tierType = "SQS/HTTP"
		}
		environmentTier := elasticbeanstalk.EnvironmentTier{
			Name: aws.String(tier),
			Type: aws.String(tierType),
		}
		createOpts.Tier = &environmentTier
	}

	if solutionStack != "" {
		createOpts.SolutionStackName = aws.String(solutionStack)
	}

	if templateName != "" {
		createOpts.TemplateName = aws.String(templateName)
	}

	log.Printf("[DEBUG] Elastic Beanstalk Environment create opts: %s", createOpts)
	resp, err := conn.CreateEnvironment(&createOpts)
	if err != nil {
		return err
	}

	// Assign the application name as the resource ID
	d.SetId(*resp.EnvironmentId)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Launching", "Updating"},
		Target:     []string{"Ready"},
		Refresh:    environmentStateRefreshFunc(conn, d.Id()),
		Timeout:    waitForReadyTimeOut,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Elastic Beanstalk Environment (%s) to become ready: %s",
			d.Id(), err)
	}

	return resourceAwsElasticBeanstalkEnvironmentRead(d, meta)
}

func resourceAwsElasticBeanstalkEnvironmentUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	envId := d.Id()
	waitForReadyTimeOut, err := time.ParseDuration(d.Get("wait_for_ready_timeout").(string))
	if err != nil {
		return err
	}

	updateOpts := elasticbeanstalk.UpdateEnvironmentInput{
		EnvironmentId: aws.String(envId),
	}

	if d.HasChange("description") {
		updateOpts.Description = aws.String(d.Get("description").(string))
	}

	if d.HasChange("solution_stack_name") {
		updateOpts.SolutionStackName = aws.String(d.Get("solution_stack_name").(string))
	}

	if d.HasChange("setting") {
		o, n := d.GetChange("setting")
		if o == nil {
			o = &schema.Set{F: optionSettingValueHash}
		}
		if n == nil {
			n = &schema.Set{F: optionSettingValueHash}
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		updateOpts.OptionSettings = extractOptionSettings(ns.Difference(os))
	}

	if d.HasChange("template_name") {
		updateOpts.TemplateName = aws.String(d.Get("template_name").(string))
	}

	log.Printf("[DEBUG] Elastic Beanstalk Environment update opts: %s", updateOpts)
	_, err = conn.UpdateEnvironment(&updateOpts)
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Launching", "Updating"},
		Target:     []string{"Ready"},
		Refresh:    environmentStateRefreshFunc(conn, d.Id()),
		Timeout:    waitForReadyTimeOut,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Elastic Beanstalk Environment (%s) to become ready: %s",
			d.Id(), err)
	}

	return resourceAwsElasticBeanstalkEnvironmentRead(d, meta)
}

func resourceAwsElasticBeanstalkEnvironmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	app := d.Get("application").(string)
	envId := d.Id()
	tier := d.Get("tier").(string)

	log.Printf("[DEBUG] Elastic Beanstalk environment read %s: id %s", d.Get("name").(string), d.Id())

	resp, err := conn.DescribeEnvironments(&elasticbeanstalk.DescribeEnvironmentsInput{
		ApplicationName: aws.String(app),
		EnvironmentIds:  []*string{aws.String(envId)},
	})

	if err != nil {
		return err
	}

	if len(resp.Environments) == 0 {
		log.Printf("[DEBUG] Elastic Beanstalk environment properties: could not find environment %s", d.Id())

		d.SetId("")
		return nil
	} else if len(resp.Environments) != 1 {
		return fmt.Errorf("Error reading application properties: found %d environments, expected 1", len(resp.Environments))
	}

	env := resp.Environments[0]

	if *env.Status == "Terminated" {
		log.Printf("[DEBUG] Elastic Beanstalk environment %s was terminated", d.Id())

		d.SetId("")
		return nil
	}

	resources, err := conn.DescribeEnvironmentResources(&elasticbeanstalk.DescribeEnvironmentResourcesInput{
		EnvironmentId: aws.String(envId),
	})

	if err != nil {
		return err
	}

	if err := d.Set("description", env.Description); err != nil {
		return err
	}

	if err := d.Set("cname", env.CNAME); err != nil {
		return err
	}

	if tier == "WebServer" {
		beanstalkCnamePrefixRegexp := regexp.MustCompile(`(^[^.]+).\w{2}-\w{4}-\d.elasticbeanstalk.com$`)
		var cnamePrefix string
		cnamePrefixMatch := beanstalkCnamePrefixRegexp.FindStringSubmatch(*env.CNAME)

		if cnamePrefixMatch == nil {
			cnamePrefix = ""
		} else {
			cnamePrefix = cnamePrefixMatch[1]
		}

		if err := d.Set("cname_prefix", cnamePrefix); err != nil {
			return err
		}
	}

	if err := d.Set("autoscaling_groups", flattenBeanstalkAsg(resources.EnvironmentResources.AutoScalingGroups)); err != nil {
		return err
	}

	if err := d.Set("instances", flattenBeanstalkInstances(resources.EnvironmentResources.Instances)); err != nil {
		return err
	}
	if err := d.Set("launch_configurations", flattenBeanstalkLc(resources.EnvironmentResources.LaunchConfigurations)); err != nil {
		return err
	}
	if err := d.Set("load_balancers", flattenBeanstalkElb(resources.EnvironmentResources.LoadBalancers)); err != nil {
		return err
	}
	if err := d.Set("queues", flattenBeanstalkSqs(resources.EnvironmentResources.Queues)); err != nil {
		return err
	}
	if err := d.Set("triggers", flattenBeanstalkTrigger(resources.EnvironmentResources.Triggers)); err != nil {
		return err
	}

	return resourceAwsElasticBeanstalkEnvironmentSettingsRead(d, meta)
}

func fetchAwsElasticBeanstalkEnvironmentSettings(d *schema.ResourceData, meta interface{}) (*schema.Set, error) {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	app := d.Get("application").(string)
	name := d.Get("name").(string)

	resp, err := conn.DescribeConfigurationSettings(&elasticbeanstalk.DescribeConfigurationSettingsInput{
		ApplicationName: aws.String(app),
		EnvironmentName: aws.String(name),
	})

	if err != nil {
		return nil, err
	}

	if len(resp.ConfigurationSettings) != 1 {
		return nil, fmt.Errorf("Error reading environment settings: received %d settings groups, expected 1", len(resp.ConfigurationSettings))
	}

	settings := &schema.Set{F: optionSettingValueHash}
	for _, optionSetting := range resp.ConfigurationSettings[0].OptionSettings {
		m := map[string]interface{}{}

		if optionSetting.Namespace != nil {
			m["namespace"] = *optionSetting.Namespace
		} else {
			return nil, fmt.Errorf("Error reading environment settings: option setting with no namespace: %v", optionSetting)
		}

		if optionSetting.OptionName != nil {
			m["name"] = *optionSetting.OptionName
		} else {
			return nil, fmt.Errorf("Error reading environment settings: option setting with no name: %v", optionSetting)
		}

		if optionSetting.Value != nil {
			switch *optionSetting.OptionName {
			case "SecurityGroups":
				m["value"] = dropGeneratedSecurityGroup(*optionSetting.Value, meta)
			case "Subnets", "ELBSubnets":
				m["value"] = sortValues(*optionSetting.Value)
			default:
				m["value"] = *optionSetting.Value
			}
		}

		settings.Add(m)
	}

	return settings, nil
}

func resourceAwsElasticBeanstalkEnvironmentSettingsRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Elastic Beanstalk environment settings read %s: id %s", d.Get("name").(string), d.Id())

	allSettings, err := fetchAwsElasticBeanstalkEnvironmentSettings(d, meta)
	if err != nil {
		return err
	}

	settings := d.Get("setting").(*schema.Set)

	log.Printf("[DEBUG] Elastic Beanstalk allSettings: %s", allSettings.GoString())
	log.Printf("[DEBUG] Elastic Beanstalk settings: %s", settings.GoString())

	// perform the set operation with only name/namespace as keys, excluding value
	// this is so we override things in the settings resource data key with updated values
	// from the api.  we skip values we didn't know about before because there are so many
	// defaults set by the eb api that we would delete many useful defaults.
	//
	// there is likely a better way to do this
	allSettingsKeySet := schema.NewSet(optionSettingKeyHash, allSettings.List())
	settingsKeySet := schema.NewSet(optionSettingKeyHash, settings.List())
	updatedSettingsKeySet := allSettingsKeySet.Intersection(settingsKeySet)

	log.Printf("[DEBUG] Elastic Beanstalk updatedSettingsKeySet: %s", updatedSettingsKeySet.GoString())

	updatedSettings := schema.NewSet(optionSettingValueHash, updatedSettingsKeySet.List())

	log.Printf("[DEBUG] Elastic Beanstalk updatedSettings: %s", updatedSettings.GoString())

	if err := d.Set("all_settings", allSettings.List()); err != nil {
		return err
	}

	if err := d.Set("setting", updatedSettings.List()); err != nil {
		return err
	}

	return nil
}

func resourceAwsElasticBeanstalkEnvironmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	waitForReadyTimeOut, err := time.ParseDuration(d.Get("wait_for_ready_timeout").(string))
	if err != nil {
		return err
	}

	opts := elasticbeanstalk.TerminateEnvironmentInput{
		EnvironmentId:      aws.String(d.Id()),
		TerminateResources: aws.Bool(true),
	}

	log.Printf("[DEBUG] Elastic Beanstalk Environment terminate opts: %s", opts)
	_, err = conn.TerminateEnvironment(&opts)

	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Terminating"},
		Target:     []string{"Terminated"},
		Refresh:    environmentStateRefreshFunc(conn, d.Id()),
		Timeout:    waitForReadyTimeOut,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Elastic Beanstalk Environment (%s) to become terminated: %s",
			d.Id(), err)
	}

	return nil
}

// environmentStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// the creation of the Beanstalk Environment
func environmentStateRefreshFunc(conn *elasticbeanstalk.ElasticBeanstalk, environmentId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeEnvironments(&elasticbeanstalk.DescribeEnvironmentsInput{
			EnvironmentIds: []*string{aws.String(environmentId)},
		})
		if err != nil {
			log.Printf("[Err] Error waiting for Elastic Beanstalk Environment state: %s", err)
			return -1, "failed", fmt.Errorf("[Err] Error waiting for Elastic Beanstalk Environment state: %s", err)
		}

		if resp == nil || len(resp.Environments) == 0 {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		var env *elasticbeanstalk.EnvironmentDescription
		for _, e := range resp.Environments {
			if environmentId == *e.EnvironmentId {
				env = e
			}
		}

		if env == nil {
			return -1, "failed", fmt.Errorf("[Err] Error finding Elastic Beanstalk Environment, environment not found")
		}

		return env, *env.Status, nil
	}
}

// we use the following two functions to allow us to split out defaults
// as they become overridden from within the template
func optionSettingValueHash(v interface{}) int {
	rd := v.(map[string]interface{})
	namespace := rd["namespace"].(string)
	optionName := rd["name"].(string)
	value, _ := rd["value"].(string)
	hk := fmt.Sprintf("%s:%s=%s", namespace, optionName, sortValues(value))
	log.Printf("[DEBUG] Elastic Beanstalk optionSettingValueHash(%#v): %s: hk=%s,hc=%d", v, optionName, hk, hashcode.String(hk))
	return hashcode.String(hk)
}

func optionSettingKeyHash(v interface{}) int {
	rd := v.(map[string]interface{})
	namespace := rd["namespace"].(string)
	optionName := rd["name"].(string)
	hk := fmt.Sprintf("%s:%s", namespace, optionName)
	log.Printf("[DEBUG] Elastic Beanstalk optionSettingKeyHash(%#v): %s: hk=%s,hc=%d", v, optionName, hk, hashcode.String(hk))
	return hashcode.String(hk)
}

func sortValues(v string) string {
	values := strings.Split(v, ",")
	sort.Strings(values)
	return strings.Join(values, ",")
}

func extractOptionSettings(s *schema.Set) []*elasticbeanstalk.ConfigurationOptionSetting {
	settings := []*elasticbeanstalk.ConfigurationOptionSetting{}

	if s != nil {
		for _, setting := range s.List() {
			settings = append(settings, &elasticbeanstalk.ConfigurationOptionSetting{
				Namespace:  aws.String(setting.(map[string]interface{})["namespace"].(string)),
				OptionName: aws.String(setting.(map[string]interface{})["name"].(string)),
				Value:      aws.String(setting.(map[string]interface{})["value"].(string)),
			})
		}
	}

	return settings
}

func dropGeneratedSecurityGroup(settingValue string, meta interface{}) string {
	conn := meta.(*AWSClient).ec2conn

	groups := strings.Split(settingValue, ",")

	resp, err := conn.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: aws.StringSlice(groups),
	})

	if err != nil {
		log.Printf("[DEBUG] Elastic Beanstalk error describing SecurityGroups: %v", err)
		return settingValue
	}

	var legitGroups []string
	for _, group := range resp.SecurityGroups {
		log.Printf("[DEBUG] Elastic Beanstalk SecurityGroup: %v", *group.GroupName)
		if !strings.HasPrefix(*group.GroupName, "awseb") {
			legitGroups = append(legitGroups, *group.GroupId)
		}
	}

	sort.Strings(legitGroups)

	return strings.Join(legitGroups, ",")
}
