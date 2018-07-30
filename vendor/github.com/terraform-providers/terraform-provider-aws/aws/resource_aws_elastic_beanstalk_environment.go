package aws

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/hashicorp/terraform/helper/structure"
)

func resourceAwsElasticBeanstalkOptionSetting() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"namespace": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"value": {
				Type:     schema.TypeString,
				Required: true,
			},
			"resource": {
				Type:     schema.TypeString,
				Optional: true,
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
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		SchemaVersion: 1,
		MigrateState:  resourceAwsElasticBeanstalkEnvironmentMigrateState,

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
			"application": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"version_label": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"cname": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cname_prefix": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},
			"tier": {
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
			"setting": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     resourceAwsElasticBeanstalkOptionSetting(),
				Set:      optionSettingValueHash,
			},
			"all_settings": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     resourceAwsElasticBeanstalkOptionSetting(),
				Set:      optionSettingValueHash,
			},
			"solution_stack_name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"template_name"},
			},
			"template_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"wait_for_ready_timeout": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "20m",
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
			"poll_interval": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					duration, err := time.ParseDuration(value)
					if err != nil {
						errors = append(errors, fmt.Errorf(
							"%q cannot be parsed as a duration: %s", k, err))
					}
					if duration < 10*time.Second || duration > 60*time.Second {
						errors = append(errors, fmt.Errorf(
							"%q must be between 10s and 180s", k))
					}
					return
				},
			},
			"autoscaling_groups": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"instances": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"launch_configurations": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"load_balancers": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"queues": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"triggers": {
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
	version := d.Get("version_label").(string)
	settings := d.Get("setting").(*schema.Set)
	solutionStack := d.Get("solution_stack_name").(string)
	templateName := d.Get("template_name").(string)

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
			return fmt.Errorf("Cannot set cname_prefix for tier: %s.", tier)
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

	if version != "" {
		createOpts.VersionLabel = aws.String(version)
	}

	// Get the current time to filter getBeanstalkEnvironmentErrors messages
	t := time.Now()
	log.Printf("[DEBUG] Elastic Beanstalk Environment create opts: %s", createOpts)
	resp, err := conn.CreateEnvironment(&createOpts)
	if err != nil {
		return err
	}

	// Assign the application name as the resource ID
	d.SetId(*resp.EnvironmentId)

	waitForReadyTimeOut, err := time.ParseDuration(d.Get("wait_for_ready_timeout").(string))
	if err != nil {
		return err
	}

	pollInterval, err := time.ParseDuration(d.Get("poll_interval").(string))
	if err != nil {
		pollInterval = 0
		log.Printf("[WARN] Error parsing poll_interval, using default backoff")
	}

	stateConf := &resource.StateChangeConf{
		Pending:      []string{"Launching", "Updating"},
		Target:       []string{"Ready"},
		Refresh:      environmentStateRefreshFunc(conn, d.Id(), t),
		Timeout:      waitForReadyTimeOut,
		Delay:        10 * time.Second,
		PollInterval: pollInterval,
		MinTimeout:   3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Elastic Beanstalk Environment (%s) to become ready: %s",
			d.Id(), err)
	}

	envErrors, err := getBeanstalkEnvironmentErrors(conn, d.Id(), t)
	if err != nil {
		return err
	}
	if envErrors != nil {
		return envErrors
	}

	return resourceAwsElasticBeanstalkEnvironmentRead(d, meta)
}

func resourceAwsElasticBeanstalkEnvironmentUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	envId := d.Id()

	var hasChange bool

	updateOpts := elasticbeanstalk.UpdateEnvironmentInput{
		EnvironmentId: aws.String(envId),
	}

	if d.HasChange("description") {
		hasChange = true
		updateOpts.Description = aws.String(d.Get("description").(string))
	}

	if d.HasChange("solution_stack_name") {
		hasChange = true
		if v, ok := d.GetOk("solution_stack_name"); ok {
			updateOpts.SolutionStackName = aws.String(v.(string))
		}
	}

	if d.HasChange("setting") {
		hasChange = true
		o, n := d.GetChange("setting")
		if o == nil {
			o = &schema.Set{F: optionSettingValueHash}
		}
		if n == nil {
			n = &schema.Set{F: optionSettingValueHash}
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		rm := extractOptionSettings(os.Difference(ns))
		add := extractOptionSettings(ns.Difference(os))

		// Additions and removals of options are done in a single API call, so we
		// can't do our normal "remove these" and then later "add these", re-adding
		// any updated settings.
		// Because of this, we need to exclude any settings in the "removable"
		// settings that are also found in the "add" settings, otherwise they
		// conflict. Here we loop through all the initial removables from the set
		// difference, and create a new slice `remove` that contains those settings
		// found in `rm` but not in `add`
		var remove []*elasticbeanstalk.ConfigurationOptionSetting
		if len(add) > 0 {
			for _, r := range rm {
				var update = false
				for _, a := range add {
					// ResourceNames are optional. Some defaults come with it, some do
					// not. We need to guard against nil/empty in state as well as
					// nil/empty from the API
					if a.ResourceName != nil {
						if r.ResourceName == nil {
							continue
						}
						if *r.ResourceName != *a.ResourceName {
							continue
						}
					}
					if *r.Namespace == *a.Namespace && *r.OptionName == *a.OptionName {
						log.Printf("[DEBUG] Updating Beanstalk setting (%s::%s) \"%s\" => \"%s\"", *a.Namespace, *a.OptionName, *r.Value, *a.Value)
						update = true
						break
					}
				}
				// Only remove options that are not updates
				if !update {
					remove = append(remove, r)
				}
			}
		} else {
			remove = rm
		}

		for _, elem := range remove {
			updateOpts.OptionsToRemove = append(updateOpts.OptionsToRemove, &elasticbeanstalk.OptionSpecification{
				Namespace:  elem.Namespace,
				OptionName: elem.OptionName,
			})
		}

		updateOpts.OptionSettings = add
	}

	if d.HasChange("template_name") {
		hasChange = true
		if v, ok := d.GetOk("template_name"); ok {
			updateOpts.TemplateName = aws.String(v.(string))
		}
	}

	if d.HasChange("version_label") {
		hasChange = true
		updateOpts.VersionLabel = aws.String(d.Get("version_label").(string))
	}

	if hasChange {
		// Get the current time to filter getBeanstalkEnvironmentErrors messages
		t := time.Now()
		log.Printf("[DEBUG] Elastic Beanstalk Environment update opts: %s", updateOpts)
		_, err := conn.UpdateEnvironment(&updateOpts)
		if err != nil {
			return err
		}

		waitForReadyTimeOut, err := time.ParseDuration(d.Get("wait_for_ready_timeout").(string))
		if err != nil {
			return err
		}
		pollInterval, err := time.ParseDuration(d.Get("poll_interval").(string))
		if err != nil {
			pollInterval = 0
			log.Printf("[WARN] Error parsing poll_interval, using default backoff")
		}

		stateConf := &resource.StateChangeConf{
			Pending:      []string{"Launching", "Updating"},
			Target:       []string{"Ready"},
			Refresh:      environmentStateRefreshFunc(conn, d.Id(), t),
			Timeout:      waitForReadyTimeOut,
			Delay:        10 * time.Second,
			PollInterval: pollInterval,
			MinTimeout:   3 * time.Second,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf(
				"Error waiting for Elastic Beanstalk Environment (%s) to become ready: %s",
				d.Id(), err)
		}

		envErrors, err := getBeanstalkEnvironmentErrors(conn, d.Id(), t)
		if err != nil {
			return err
		}
		if envErrors != nil {
			return envErrors
		}
	}

	if d.HasChange("tags") {
		o, n := d.GetChange("tags")
		oldTags := tagsFromMapBeanstalk(o.(map[string]interface{}))
		newTags := tagsFromMapBeanstalk(n.(map[string]interface{}))

		tagsToAdd, tagNamesToRemove := diffTagsBeanstalk(oldTags, newTags)

		updateTags := elasticbeanstalk.UpdateTagsForResourceInput{
			ResourceArn:  aws.String(d.Get("arn").(string)),
			TagsToAdd:    tagsToAdd,
			TagsToRemove: tagNamesToRemove,
		}

		// Get the current time to filter getBeanstalkEnvironmentErrors messages
		t := time.Now()
		log.Printf("[DEBUG] Elastic Beanstalk Environment update tags: %s", updateTags)
		_, err := conn.UpdateTagsForResource(&updateTags)
		if err != nil {
			return err
		}

		waitForReadyTimeOut, err := time.ParseDuration(d.Get("wait_for_ready_timeout").(string))
		if err != nil {
			return err
		}
		pollInterval, err := time.ParseDuration(d.Get("poll_interval").(string))
		if err != nil {
			pollInterval = 0
			log.Printf("[WARN] Error parsing poll_interval, using default backoff")
		}

		stateConf := &resource.StateChangeConf{
			Pending:      []string{"Launching", "Updating"},
			Target:       []string{"Ready"},
			Refresh:      environmentStateRefreshFunc(conn, d.Id(), t),
			Timeout:      waitForReadyTimeOut,
			Delay:        10 * time.Second,
			PollInterval: pollInterval,
			MinTimeout:   3 * time.Second,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf(
				"Error waiting for Elastic Beanstalk Environment (%s) to become ready: %s",
				d.Id(), err)
		}

		envErrors, err := getBeanstalkEnvironmentErrors(conn, d.Id(), t)
		if err != nil {
			return err
		}
		if envErrors != nil {
			return envErrors
		}
	}

	return resourceAwsElasticBeanstalkEnvironmentRead(d, meta)
}

func resourceAwsElasticBeanstalkEnvironmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticbeanstalkconn

	envId := d.Id()

	log.Printf("[DEBUG] Elastic Beanstalk environment read %s: id %s", d.Get("name").(string), d.Id())

	resp, err := conn.DescribeEnvironments(&elasticbeanstalk.DescribeEnvironmentsInput{
		EnvironmentIds: []*string{aws.String(envId)},
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

	d.Set("arn", env.EnvironmentArn)

	if err := d.Set("name", env.EnvironmentName); err != nil {
		return err
	}

	if err := d.Set("application", env.ApplicationName); err != nil {
		return err
	}

	if err := d.Set("description", env.Description); err != nil {
		return err
	}

	if err := d.Set("cname", env.CNAME); err != nil {
		return err
	}

	if err := d.Set("version_label", env.VersionLabel); err != nil {
		return err
	}

	if err := d.Set("tier", *env.Tier.Name); err != nil {
		return err
	}

	if env.CNAME != nil {
		beanstalkCnamePrefixRegexp := regexp.MustCompile(`(^[^.]+)(.\w{2}-\w{4,9}-\d)?.elasticbeanstalk.com$`)
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
	} else {
		if err := d.Set("cname_prefix", ""); err != nil {
			return err
		}
	}

	if err := d.Set("solution_stack_name", env.SolutionStackName); err != nil {
		return err
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

	tags, err := conn.ListTagsForResource(&elasticbeanstalk.ListTagsForResourceInput{
		ResourceArn: aws.String(d.Get("arn").(string)),
	})

	if err != nil {
		return err
	}

	if err := d.Set("tags", tagsToMapBeanstalk(tags.ResourceTags)); err != nil {
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

		if *optionSetting.Namespace == "aws:autoscaling:scheduledaction" && optionSetting.ResourceName != nil {
			m["resource"] = *optionSetting.ResourceName
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

	opts := elasticbeanstalk.TerminateEnvironmentInput{
		EnvironmentId:      aws.String(d.Id()),
		TerminateResources: aws.Bool(true),
	}

	// Get the current time to filter getBeanstalkEnvironmentErrors messages
	t := time.Now()
	log.Printf("[DEBUG] Elastic Beanstalk Environment terminate opts: %s", opts)
	_, err := conn.TerminateEnvironment(&opts)

	if err != nil {
		return err
	}

	waitForReadyTimeOut, err := time.ParseDuration(d.Get("wait_for_ready_timeout").(string))
	if err != nil {
		return err
	}
	pollInterval, err := time.ParseDuration(d.Get("poll_interval").(string))
	if err != nil {
		pollInterval = 0
		log.Printf("[WARN] Error parsing poll_interval, using default backoff")
	}

	stateConf := &resource.StateChangeConf{
		Pending:      []string{"Terminating"},
		Target:       []string{"Terminated"},
		Refresh:      environmentStateRefreshFunc(conn, d.Id(), t),
		Timeout:      waitForReadyTimeOut,
		Delay:        10 * time.Second,
		PollInterval: pollInterval,
		MinTimeout:   3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Elastic Beanstalk Environment (%s) to become terminated: %s",
			d.Id(), err)
	}

	envErrors, err := getBeanstalkEnvironmentErrors(conn, d.Id(), t)
	if err != nil {
		return err
	}
	if envErrors != nil {
		return envErrors
	}

	return nil
}

// environmentStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// the creation of the Beanstalk Environment
func environmentStateRefreshFunc(conn *elasticbeanstalk.ElasticBeanstalk, environmentId string, t time.Time) resource.StateRefreshFunc {
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

		envErrors, err := getBeanstalkEnvironmentErrors(conn, environmentId, t)
		if err != nil {
			return -1, "failed", err
		}
		if envErrors != nil {
			return -1, "failed", envErrors
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
	var resourceName string
	if v, ok := rd["resource"].(string); ok {
		resourceName = v
	}
	value, _ := rd["value"].(string)
	value, _ = structure.NormalizeJsonString(value)
	hk := fmt.Sprintf("%s:%s%s=%s", namespace, optionName, resourceName, sortValues(value))
	log.Printf("[DEBUG] Elastic Beanstalk optionSettingValueHash(%#v): %s: hk=%s,hc=%d", v, optionName, hk, hashcode.String(hk))
	return hashcode.String(hk)
}

func optionSettingKeyHash(v interface{}) int {
	rd := v.(map[string]interface{})
	namespace := rd["namespace"].(string)
	optionName := rd["name"].(string)
	var resourceName string
	if v, ok := rd["resource"].(string); ok {
		resourceName = v
	}
	hk := fmt.Sprintf("%s:%s%s", namespace, optionName, resourceName)
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
			optionSetting := elasticbeanstalk.ConfigurationOptionSetting{
				Namespace:  aws.String(setting.(map[string]interface{})["namespace"].(string)),
				OptionName: aws.String(setting.(map[string]interface{})["name"].(string)),
				Value:      aws.String(setting.(map[string]interface{})["value"].(string)),
			}
			if *optionSetting.Namespace == "aws:autoscaling:scheduledaction" {
				if v, ok := setting.(map[string]interface{})["resource"].(string); ok && v != "" {
					optionSetting.ResourceName = aws.String(v)
				}
			}
			settings = append(settings, &optionSetting)
		}
	}

	return settings
}

func dropGeneratedSecurityGroup(settingValue string, meta interface{}) string {
	conn := meta.(*AWSClient).ec2conn

	groups := strings.Split(settingValue, ",")

	// Check to see if groups are ec2-classic or vpc security groups
	ec2Classic := true
	beanstalkSGRegexp := "sg-[0-9a-fA-F]{8}"
	for _, g := range groups {
		if ok, _ := regexp.MatchString(beanstalkSGRegexp, g); ok {
			ec2Classic = false
			break
		}
	}

	var resp *ec2.DescribeSecurityGroupsOutput
	var err error

	if ec2Classic {
		resp, err = conn.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
			GroupNames: aws.StringSlice(groups),
		})
	} else {
		resp, err = conn.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
			GroupIds: aws.StringSlice(groups),
		})
	}

	if err != nil {
		log.Printf("[DEBUG] Elastic Beanstalk error describing SecurityGroups: %v", err)
		return settingValue
	}

	log.Printf("[DEBUG] Elastic Beanstalk using ec2-classic security-groups: %t", ec2Classic)
	var legitGroups []string
	for _, group := range resp.SecurityGroups {
		log.Printf("[DEBUG] Elastic Beanstalk SecurityGroup: %v", *group.GroupName)
		if !strings.HasPrefix(*group.GroupName, "awseb") {
			if ec2Classic {
				legitGroups = append(legitGroups, *group.GroupName)
			} else {
				legitGroups = append(legitGroups, *group.GroupId)
			}
		}
	}

	sort.Strings(legitGroups)

	return strings.Join(legitGroups, ",")
}

type beanstalkEnvironmentError struct {
	eventDate     *time.Time
	environmentID string
	message       *string
}

func (e beanstalkEnvironmentError) Error() string {
	return e.eventDate.String() + " (" + e.environmentID + ") : " + *e.message
}

type beanstalkEnvironmentErrors []*beanstalkEnvironmentError

func (e beanstalkEnvironmentErrors) Len() int           { return len(e) }
func (e beanstalkEnvironmentErrors) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e beanstalkEnvironmentErrors) Less(i, j int) bool { return e[i].eventDate.Before(*e[j].eventDate) }

func getBeanstalkEnvironmentErrors(conn *elasticbeanstalk.ElasticBeanstalk, environmentId string, t time.Time) (*multierror.Error, error) {
	environmentErrors, err := conn.DescribeEvents(&elasticbeanstalk.DescribeEventsInput{
		EnvironmentId: aws.String(environmentId),
		Severity:      aws.String("ERROR"),
		StartTime:     aws.Time(t),
	})

	if err != nil {
		return nil, fmt.Errorf("[Err] Unable to get Elastic Beanstalk Evironment events: %s", err)
	}

	var events beanstalkEnvironmentErrors
	for _, event := range environmentErrors.Events {
		e := &beanstalkEnvironmentError{
			eventDate:     event.EventDate,
			environmentID: environmentId,
			message:       event.Message,
		}
		events = append(events, e)
	}
	sort.Sort(beanstalkEnvironmentErrors(events))

	var result *multierror.Error
	for _, event := range events {
		result = multierror.Append(result, event)
	}

	return result, nil
}
