package rundeck

import (
	"fmt"

	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/apparentlymart/go-rundeck-api/rundeck"
)

func resourceRundeckJob() *schema.Resource {
	return &schema.Resource{
		Create: CreateJob,
		Update: UpdateJob,
		Delete: DeleteJob,
		Exists: JobExists,
		Read:   ReadJob,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"project_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"log_level": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "INFO",
			},

			"allow_concurrent_executions": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"max_thread_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},

			// false: Stop at the failed step: Fail immediately (default).
			// true: Run remaining steps before failing: Continue to next steps and fail the job at the end.
			"continue_on_error": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"rank_order": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "ascending",
				ValidateFunc: validateValueFunc([]string{"ascending", "descending"}),
			},

			"rank_attribute": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"preserve_options_order": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"command_ordering_strategy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "node-first",
			},

			"node_filter_query": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"node_filter_exclude_precedence": &schema.Schema{
				Type:       schema.TypeBool,
				Optional:   true,
				Deprecated: "Set in config rundeck-config.properties to enable: rundeck.nodefilters.showPrecedenceOption=true",
			},

			// true: Target nodes are selected by default
			// false: The user has to explicitly select target nodes
			"nodes_selected_by_default": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"schedule_cron": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"execution_timeout": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"execution_retry": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			// Element: Job>Dispatch
			"dispatch": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						// true: Fail the step without running on any remaining nodes.
						// false: Continue running on any remaining nodes before failing the step.
						"continue_on_error": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},

			"option": &schema.Schema{
				// This is a list because order is important when preserve_options_order is
				// set. When it's not set the order is unimportant but preserved by Rundeck/
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"default_value": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"value_choices": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},

						"value_choices_url": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"require_predefined_choice": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},

						"validation_regex": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"description": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"required": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},

						"allow_multiple_values": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},

						"multi_value_delimiter": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"obscure_input": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},

						"storage_path": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"exposed_to_scripts": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},

			"notification": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"onfailure": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem:     resourceJobNotification(),
						},

						"onstart": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem:     resourceJobNotification(),
						},

						"onsuccess": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem:     resourceJobNotification(),
						},
					},
				},
			},

			"command": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem:     resourceRundeckJobCommandResource(0),
			},
		},
	}
}

func resourceJobNotification() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"email": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subject": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"recipients": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},

						"attach_log": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},

			"webhook_urls": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceRundeckJobCommandResource(depth int) *schema.Resource {

	schemaMap := map[string]*schema.Schema{
		"description": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},

		"continue_on_error": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},

		"shell_command": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},

		"inline_script": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},

		"file_extension": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},

		"script_file": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},

		"script_file_args": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},

		"invocation_string": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},

		"arguments_quoted": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},

		"job": &schema.Schema{
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"name": &schema.Schema{
						Type:     schema.TypeString,
						Required: true,
					},
					"group_name": &schema.Schema{
						Type:     schema.TypeString,
						Optional: true,
					},
					"run_for_each_node": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
					},
					"args": &schema.Schema{
						Type:     schema.TypeString,
						Optional: true,
					},
					"node_filter_query": &schema.Schema{
						Type:     schema.TypeString,
						Optional: true,
					},
					"max_thread_count": &schema.Schema{
						Type:     schema.TypeInt,
						Optional: true,
					},
				},
			},
		},

		"step_plugin": &schema.Schema{
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 1,
			Elem:     resourceRundeckJobPluginResource(),
		},

		"node_step_plugin": &schema.Schema{
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 1,
			Elem:     resourceRundeckJobPluginResource(),
		},
	}

	if depth < 1 {
		schemaMap["errorhandler"] = &schema.Schema{
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 1,
			Elem:     resourceRundeckJobCommandResource(depth + 1),
		}
	}

	return &schema.Resource{
		Schema: schemaMap,
	}
}

func resourceRundeckJobPluginResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"config": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
		},
	}
}

func CreateJob(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.Client)

	job, err := jobFromResourceData(d)
	if err != nil {
		return err
	}

	jobSummary, err := client.CreateJob(job)
	if err != nil {
		return err
	}

	d.SetId(jobSummary.ID)
	d.Set("id", jobSummary.ID)

	return ReadJob(d, meta)
}

func UpdateJob(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.Client)

	job, err := jobFromResourceData(d)
	if err != nil {
		return err
	}

	jobSummary, err := client.CreateOrUpdateJob(job)
	if err != nil {
		return err
	}

	d.SetId(jobSummary.ID)
	d.Set("id", jobSummary.ID)

	return ReadJob(d, meta)
}

func DeleteJob(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.Client)

	err := client.DeleteJob(d.Id())
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func JobExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*rundeck.Client)

	_, err := client.GetJob(d.Id())
	if err != nil {
		if _, ok := err.(rundeck.NotFoundError); ok {
			err = nil
		}
		return false, err
	}

	return true, nil
}

func ReadJob(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.Client)

	job, err := client.GetJob(d.Id())
	if err != nil {
		return err
	}

	return jobToResourceData(job, d)
}

func commandFromMap(commandMap map[string]interface{}) (*rundeck.JobCommand, error) {

	// Element: Command
	command := rundeck.JobCommand{
		ContinueOnError: commandMap["continue_on_error"].(bool),
		ShellCommand:    commandMap["shell_command"].(string),
		Script:          commandMap["inline_script"].(string),
		FileExtension:   commandMap["file_extension"].(string),
		ScriptFile:      commandMap["script_file"].(string),
		ScriptFileArgs:  commandMap["script_file_args"].(string),
		Description:     commandMap["description"].(string),
	}

	// Element: Errorhandling
	if errorConfigI, present := commandMap["errorhandler"]; present {
		errorConfigI := errorConfigI.([]interface{})
		if len(errorConfigI) > 0 {
			errorConfigMap := errorConfigI[0].(map[string]interface{})
			errorHandler, error := commandFromMap(errorConfigMap)
			if error != nil {
				return nil, error
			}
			command.ErrorHandler = errorHandler
		}
	}

	// Element: Command>InvocationString
	if invocationString, ok := commandMap["invocation_string"]; ok {
		command.ScriptInterpreter = &rundeck.JobCommandScriptInterpreter{
			InvocationString: invocationString.(string),
			ArgsQuoted:       commandMap["arguments_quoted"].(bool),
		}
	}

	// Element: Command>Job(ref)
	jobRefsI := commandMap["job"].([]interface{})
	if len(jobRefsI) > 0 {
		jobRefMap := jobRefsI[0].(map[string]interface{})
		command.Job = &rundeck.JobCommandJobRef{
			Name:           jobRefMap["name"].(string),
			GroupName:      jobRefMap["group_name"].(string),
			RunForEachNode: jobRefMap["run_for_each_node"].(bool),
			NodeFilter: &rundeck.JobNodeFilter{
				Query: jobRefMap["node_filter_query"].(string),
			},
			Dispatch: &rundeck.JobDispatch{
				MaxThreadCount: jobRefMap["max_thread_count"].(int),
			},
			Arguments: rundeck.JobCommandJobRefArguments(jobRefMap["args"].(string)),
		}
	}

	// Element: Command>StepPlugin
	stepPluginsI := commandMap["step_plugin"].([]interface{})
	if len(stepPluginsI) > 0 {
		stepPluginMap := stepPluginsI[0].(map[string]interface{})
		configI := stepPluginMap["config"].(map[string]interface{})
		config := map[string]string{}
		for k, v := range configI {
			config[k] = v.(string)
		}
		command.StepPlugin = &rundeck.JobPlugin{
			Type:   stepPluginMap["type"].(string),
			Config: config,
		}
	}

	// Element: Command>NodeStepPlugin
	stepPluginsI = commandMap["node_step_plugin"].([]interface{})
	if len(stepPluginsI) > 0 {
		stepPluginMap := stepPluginsI[0].(map[string]interface{})
		configI := stepPluginMap["config"].(map[string]interface{})
		config := map[string]string{}
		for k, v := range configI {
			config[k] = v.(string)
		}
		command.NodeStepPlugin = &rundeck.JobPlugin{
			Type:   stepPluginMap["type"].(string),
			Config: config,
		}
	}
	return &command, nil
}

func notificationFromMap(notificationMap map[string]interface{}) (*rundeck.Notification, error) {
	notification := &rundeck.Notification{}

	// Element: Job>Notification>OnFailure>Email
	emailNotificationConfigI := notificationMap["email"].([]interface{})
	if len(emailNotificationConfigI) > 0 {
		emailNotificationMap := emailNotificationConfigI[0].(map[string]interface{})
		notification.Email = &rundeck.EmailNotification{
			AttachLog:  emailNotificationMap["attach_log"].(bool),
			Recipients: rundeck.NotificationEmails([]string{}),
			Subject:    emailNotificationMap["subject"].(string),
		}

		for _, iv := range emailNotificationMap["recipients"].([]interface{}) {
			notification.Email.Recipients = append(notification.Email.Recipients, iv.(string))
		}
	}

	// Element: Job>Notification>OnFailure>WebHook
	webHookUrls := notificationMap["webhook_urls"].([]interface{})
	if len(webHookUrls) > 0 {
		notification.WebHook = &rundeck.WebHookNotification{
			Urls: rundeck.NotificationUrls([]string{}),
		}

		for _, iv := range webHookUrls {
			notification.WebHook.Urls = append(notification.WebHook.Urls, iv.(string))
		}
	}

	return notification, nil
}

func notificationFromConfig(configI map[string]interface{}, handle string) (*rundeck.Notification, error) {
	notificationConfigI := configI[handle].([]interface{})
	if len(notificationConfigI) > 0 {
		notification, error := notificationFromMap(notificationConfigI[0].(map[string]interface{}))
		if error != nil {
			return nil, error
		}
		return notification, nil
	}
	return nil, nil
}

func jobFromResourceData(d *schema.ResourceData) (*rundeck.JobDetail, error) {

	// Element: Job
	job := &rundeck.JobDetail{
		ID:                     d.Id(),
		Name:                   d.Get("name").(string),
		GroupName:              d.Get("group_name").(string),
		ProjectName:            d.Get("project_name").(string),
		Description:            d.Get("description").(string),
		LogLevel:               d.Get("log_level").(string),
		NodesSelectedByDefault: d.Get("nodes_selected_by_default").(bool),
		Timeout:                d.Get("execution_timeout").(string),
		Retry:                  d.Get("execution_retry").(string),
	}

	// Issue: #8677: Rundeck Provider: rundeck_job doesn't have idempotency
	if d.Get("allow_concurrent_executions").(bool) {
		job.AllowConcurrentExecutions = true
	}

	// Element: Job>Notification
	notificationConfigI := d.Get("notification").([]interface{})
	if len(notificationConfigI) > 0 {
		notificationConfigI := notificationConfigI[0].(map[string]interface{})
		onFailure, error := notificationFromConfig(notificationConfigI, "onfailure")
		if error != nil {
			return nil, error
		}

		onStart, error := notificationFromConfig(notificationConfigI, "onstart")
		if error != nil {
			return nil, error
		}

		onSuccess, error := notificationFromConfig(notificationConfigI, "onsuccess")
		if error != nil {
			return nil, error
		}

		job.Notification = &rundeck.JobNotification{
			OnFailure: onFailure,
			OnStart:   onStart,
			OnSuccess: onSuccess,
		}
	}

	// Element: Job>Dispatch
	dispatchConfigI := d.Get("dispatch").([]interface{})
	if len(dispatchConfigI) > 0 {
		dispatchMap := dispatchConfigI[0].(map[string]interface{})
		job.Dispatch = &rundeck.JobDispatch{
			MaxThreadCount:  d.Get("max_thread_count").(int),
			ContinueOnError: dispatchMap["continue_on_error"].(bool),
			RankAttribute:   d.Get("rank_attribute").(string),
			RankOrder:       d.Get("rank_order").(string),
		}
	}

	// Element: Job>Schedule
	scheduleCron := d.Get("schedule_cron").(string)
	if len(scheduleCron) > 0 {
		scheduleCronArray := strings.Split(scheduleCron, " ")
		if len(scheduleCronArray) != 7 {
			return nil, fmt.Errorf("rundeck schedule_cron format is incorrect")
		}
		job.Schedule = &rundeck.JobSchedule{
			Month: rundeck.JobScheduleMonth{
				Month: scheduleCronArray[4],
			},
			Time: rundeck.JobScheduleTime{
				Hour:    scheduleCronArray[2],
				Minute:  scheduleCronArray[1],
				Seconds: scheduleCronArray[0],
			},
			Year: rundeck.JobScheduleYear{
				Year: scheduleCronArray[6],
			},
		}

		if scheduleCronArray[3] == "?" && scheduleCronArray[5] == "?" {
			return nil, fmt.Errorf("rundeck schedule_cron format is incorrect, to many ?")
		} else if scheduleCronArray[5] == "?" {
			job.Schedule.DayOfMonth = &rundeck.JobScheduleDayOfMonth{}
			job.Schedule.Month.Day = scheduleCronArray[3]
		} else if scheduleCronArray[3] == "?" {
			job.Schedule.WeekDay = &rundeck.JobScheduleWeekDay{
				Day: scheduleCronArray[5],
			}
		} else {
			return nil, fmt.Errorf("rundeck schedule_cron format is incorrect, missing ?")
		}
	}

	// Element: Job>Sequence
	sequence := &rundeck.JobCommandSequence{
		ContinueOnError:  d.Get("continue_on_error").(bool),
		OrderingStrategy: d.Get("command_ordering_strategy").(string),
		Commands:         []rundeck.JobCommand{},
	}

	// Element: Job>Command
	commandConfigs := d.Get("command").([]interface{})
	for _, commandI := range commandConfigs {
		command, error := commandFromMap(commandI.(map[string]interface{}))
		if error != nil {
			return nil, error
		}

		sequence.Commands = append(sequence.Commands, *command)
	}
	job.CommandSequence = sequence

	// Element: Job>Option
	optionConfigsI := d.Get("option").([]interface{})
	if len(optionConfigsI) > 0 {
		optionsConfig := &rundeck.JobOptions{
			PreserveOrder: d.Get("preserve_options_order").(bool),
			Options:       []rundeck.JobOption{},
		}
		for _, optionI := range optionConfigsI {
			optionMap := optionI.(map[string]interface{})
			option := rundeck.JobOption{
				Name:                    optionMap["name"].(string),
				DefaultValue:            optionMap["default_value"].(string),
				ValueChoices:            rundeck.JobValueChoices([]string{}),
				ValueChoicesURL:         optionMap["value_choices_url"].(string),
				RequirePredefinedChoice: optionMap["require_predefined_choice"].(bool),
				ValidationRegex:         optionMap["validation_regex"].(string),
				Description:             optionMap["description"].(string),
				IsRequired:              optionMap["required"].(bool),
				AllowsMultipleValues:    optionMap["allow_multiple_values"].(bool),
				MultiValueDelimiter:     optionMap["multi_value_delimiter"].(string),
				ObscureInput:            optionMap["obscure_input"].(bool),
				ValueIsExposedToScripts: optionMap["exposed_to_scripts"].(bool),
				StoragePath:             optionMap["storage_path"].(string),
			}

			for _, iv := range optionMap["value_choices"].([]interface{}) {
				option.ValueChoices = append(option.ValueChoices, iv.(string))
			}

			optionsConfig.Options = append(optionsConfig.Options, option)
		}
		job.OptionsConfig = optionsConfig
	}

	// Job>Filter
	if d.Get("node_filter_query").(string) != "" {
		job.NodeFilter = &rundeck.JobNodeFilter{
			ExcludePrecedence: d.Get("node_filter_exclude_precedence").(bool),
			Query:             d.Get("node_filter_query").(string),
		}
	}

	return job, nil
}

func commandToMap(command rundeck.JobCommand) (*map[string]interface{}, error) {

	// Elment: Command
	commandConfigI := map[string]interface{}{
		"continue_on_error": command.ContinueOnError,
		"shell_command":     command.ShellCommand,
		"inline_script":     command.Script,
		"file_extension":    command.FileExtension,
		"script_file":       command.ScriptFile,
		"script_file_args":  command.ScriptFileArgs,
		"description":       command.Description,
	}

	// Element: Errorhandling
	if command.ErrorHandler != nil {
		errorHandler, error := commandToMap(*command.ErrorHandler)
		if error != nil {
			return nil, error
		}
		commandConfigI["errorhandling"] = errorHandler
	}

	// Element: ScriptInterpreter
	if command.ScriptInterpreter != nil {
		commandConfigI["invocation_string"] = command.ScriptInterpreter.InvocationString
		commandConfigI["arguments_quoted"] = command.ScriptInterpreter.ArgsQuoted
	}

	// Element: JobRef
	if command.Job != nil {
		commandConfigI["job"] = []interface{}{
			map[string]interface{}{
				"name":              command.Job.Name,
				"group_name":        command.Job.GroupName,
				"run_for_each_node": command.Job.RunForEachNode,
				"args":              command.Job.Arguments,
			},
		}

		if command.Job.NodeFilter != nil {
			commandConfigI["node_filter_query"] = command.Job.NodeFilter.Query
		}

		if command.Job.Dispatch != nil {
			commandConfigI["max_thread_count"] = command.Job.Dispatch.MaxThreadCount
		}
	}

	// Element: StepPlugin
	if command.StepPlugin != nil {
		commandConfigI["step_plugin"] = []interface{}{
			map[string]interface{}{
				"type":   command.StepPlugin.Type,
				"config": map[string]string(command.StepPlugin.Config),
			},
		}
	}

	// Element: NodeStepPlugin
	if command.NodeStepPlugin != nil {
		commandConfigI["node_step_plugin"] = []interface{}{
			map[string]interface{}{
				"type":   command.NodeStepPlugin.Type,
				"config": map[string]string(command.NodeStepPlugin.Config),
			},
		}
	}

	return &commandConfigI, nil
}

func notificationToMap(notification *rundeck.Notification) (*map[string]interface{}, error) {
	notificationMap := map[string]interface{}{}
	if notification.Email != nil {
		notificationMap["email"] = map[string]interface{}{
			"subject":    notification.Email.Subject,
			"recipients": notification.Email.Recipients,
			"attach_log": notification.Email.AttachLog,
		}
	}

	if notification.WebHook != nil {
		notificationMap["webhook_urls"] = notification.WebHook.Urls
	}

	return &notificationMap, nil
}

func jobToResourceData(job *rundeck.JobDetail, d *schema.ResourceData) error {

	// Element: Job
	d.SetId(job.ID)
	d.Set("id", job.ID)
	d.Set("name", job.Name)
	d.Set("group_name", job.GroupName)

	// The project name is not consistently returned in all rundeck versions,
	// so we'll only update it if it's set. Jobs can't move between projects
	// anyway, so this is harmless.
	if job.ProjectName != "" {
		d.Set("project_name", job.ProjectName)
	}

	d.Set("description", job.Description)
	d.Set("log_level", job.LogLevel)

	d.Set("allow_concurrent_executions", job.AllowConcurrentExecutions)
	d.Set("nodes_selected_by_default", job.NodesSelectedByDefault)
	d.Set("execution_timeout", job.Timeout)
	d.Set("execution_retry", job.Retry)

	// Element: Job>Filter
	d.Set("node_filter_query", nil)
	d.Set("node_filter_exclude_precedence", nil)
	if job.NodeFilter != nil {
		d.Set("node_filter_query", job.NodeFilter.Query)
		d.Set("node_filter_exclude_precedence", job.NodeFilter.ExcludePrecedence)
	}

	// Element: Job>Schedule
	if job.Schedule != nil {
		dayOfMonth := "?"
		weekDay := "?"
		if job.Schedule.DayOfMonth != nil {
			dayOfMonth = job.Schedule.Month.Day
		} else {
			weekDay = job.Schedule.WeekDay.Day
		}

		d.Set("schedule_cron", fmt.Sprintf("%s %s %s %s %s %s %s",
			job.Schedule.Time.Seconds,
			job.Schedule.Time.Minute,
			job.Schedule.Time.Hour,
			dayOfMonth,
			job.Schedule.Month.Month,
			weekDay,
			job.Schedule.Year.Year))
	}

	// Element: Job>Option
	optionConfigsI := []interface{}{}
	if job.OptionsConfig != nil {
		d.Set("preserve_options_order", job.OptionsConfig.PreserveOrder)
		for _, option := range job.OptionsConfig.Options {
			optionConfigI := map[string]interface{}{
				"name":                      option.Name,
				"default_value":             option.DefaultValue,
				"value_choices":             option.ValueChoices,
				"value_choices_url":         option.ValueChoicesURL,
				"require_predefined_choice": option.RequirePredefinedChoice,
				"validation_regex":          option.ValidationRegex,
				"decription":                option.Description,
				"required":                  option.IsRequired,
				"allow_multiple_values":     option.AllowsMultipleValues,
				"multi_value_delimiter":     option.MultiValueDelimiter,
				"obscure_input":             option.ObscureInput,
				"exposed_to_scripts":        option.ValueIsExposedToScripts,
				"storage_path":              option.StoragePath,
			}
			optionConfigsI = append(optionConfigsI, optionConfigI)
		}
	}
	d.Set("option", optionConfigsI)

	// Element: Job>Notification
	if job.Notification != nil {
		jobNotificationMap := map[string]interface{}{}
		if job.Notification.OnFailure != nil {
			notificationMap, error := notificationToMap(job.Notification.OnFailure)
			if error != nil {
				return error
			}

			jobNotificationMap["onfailure"] = notificationMap
		}

		if job.Notification.OnStart != nil {
			notificationMap, error := notificationToMap(job.Notification.OnStart)
			if error != nil {
				return error
			}

			jobNotificationMap["onstart"] = notificationMap
		}

		if job.Notification.OnSuccess != nil {
			notificationMap, error := notificationToMap(job.Notification.OnSuccess)
			if error != nil {
				return error
			}

			jobNotificationMap["onsuccess"] = notificationMap
		}

		d.Set("notification", jobNotificationMap)
	}

	// Element: Job>Command
	commandConfigsI := []interface{}{}
	if job.CommandSequence != nil {
		d.Set("continue_on_error", job.CommandSequence.ContinueOnError)
		d.Set("command_ordering_strategy", job.CommandSequence.OrderingStrategy)
		for _, command := range job.CommandSequence.Commands {
			commandConfigI, error := commandToMap(command)
			if error != nil {
				return error
			}

			commandConfigsI = append(commandConfigsI, commandConfigI)
		}
	}
	d.Set("command", commandConfigsI)

	// Element: Job>Dispatch
	if job.Dispatch != nil {
		d.Set("max_thread_count", job.Dispatch.MaxThreadCount)
		d.Set("rank_attribute", job.Dispatch.RankAttribute)
		d.Set("rank_order", job.Dispatch.RankOrder)
		dispatchConfigI := map[string]interface{}{
			"continue_on_error": job.Dispatch.ContinueOnError,
		}
		d.Set("dispatch", dispatchConfigI)
	}

	return nil
}
