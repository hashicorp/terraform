package rundeck

import (
	"fmt"

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
			},

			"max_thread_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},

			"continue_on_error": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"rank_order": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "ascending",
			},

			"rank_attribute": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"preserve_options_order": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
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
				Type:     schema.TypeBool,
				Optional: true,
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

						"exposed_to_scripts": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},

			"command": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"shell_command": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"inline_script": &schema.Schema{
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

						"job": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
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
								},
							},
						},

						"step_plugin": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     resourceRundeckJobPluginResource(),
						},

						"node_step_plugin": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     resourceRundeckJobPluginResource(),
						},
					},
				},
			},
		},
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

func jobFromResourceData(d *schema.ResourceData) (*rundeck.JobDetail, error) {
	job := &rundeck.JobDetail{
		ID:                        d.Id(),
		Name:                      d.Get("name").(string),
		GroupName:                 d.Get("group_name").(string),
		ProjectName:               d.Get("project_name").(string),
		Description:               d.Get("description").(string),
		LogLevel:                  d.Get("log_level").(string),
		AllowConcurrentExecutions: d.Get("allow_concurrent_executions").(bool),
		Dispatch: &rundeck.JobDispatch{
			MaxThreadCount:  d.Get("max_thread_count").(int),
			ContinueOnError: d.Get("continue_on_error").(bool),
			RankAttribute:   d.Get("rank_attribute").(string),
			RankOrder:       d.Get("rank_order").(string),
		},
	}

	sequence := &rundeck.JobCommandSequence{
		ContinueOnError:  d.Get("continue_on_error").(bool),
		OrderingStrategy: d.Get("command_ordering_strategy").(string),
		Commands:         []rundeck.JobCommand{},
	}

	commandConfigs := d.Get("command").([]interface{})
	for _, commandI := range commandConfigs {
		commandMap := commandI.(map[string]interface{})
		command := rundeck.JobCommand{
			ShellCommand:   commandMap["shell_command"].(string),
			Script:         commandMap["inline_script"].(string),
			ScriptFile:     commandMap["script_file"].(string),
			ScriptFileArgs: commandMap["script_file_args"].(string),
		}

		jobRefsI := commandMap["job"].([]interface{})
		if len(jobRefsI) > 1 {
			return nil, fmt.Errorf("rundeck command may have no more than one job")
		}
		if len(jobRefsI) > 0 {
			jobRefMap := jobRefsI[0].(map[string]interface{})
			command.Job = &rundeck.JobCommandJobRef{
				Name:           jobRefMap["name"].(string),
				GroupName:      jobRefMap["group_name"].(string),
				RunForEachNode: jobRefMap["run_for_each_node"].(bool),
				Arguments:      rundeck.JobCommandJobRefArguments(jobRefMap["args"].(string)),
			}
		}

		stepPluginsI := commandMap["step_plugin"].([]interface{})
		if len(stepPluginsI) > 1 {
			return nil, fmt.Errorf("rundeck command may have no more than one step plugin")
		}
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

		stepPluginsI = commandMap["node_step_plugin"].([]interface{})
		if len(stepPluginsI) > 1 {
			return nil, fmt.Errorf("rundeck command may have no more than one node step plugin")
		}
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

		sequence.Commands = append(sequence.Commands, command)
	}
	job.CommandSequence = sequence

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
			}

			for _, iv := range optionMap["value_choices"].([]interface{}) {
				option.ValueChoices = append(option.ValueChoices, iv.(string))
			}

			optionsConfig.Options = append(optionsConfig.Options, option)
		}
		job.OptionsConfig = optionsConfig
	}

	if d.Get("node_filter_query").(string) != "" {
		job.NodeFilter = &rundeck.JobNodeFilter{
			ExcludePrecedence: d.Get("node_filter_exclude_precedence").(bool),
			Query:             d.Get("node_filter_query").(string),
		}
	}

	return job, nil
}

func jobToResourceData(job *rundeck.JobDetail, d *schema.ResourceData) error {

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
	if job.Dispatch != nil {
		d.Set("max_thread_count", job.Dispatch.MaxThreadCount)
		d.Set("continue_on_error", job.Dispatch.ContinueOnError)
		d.Set("rank_attribute", job.Dispatch.RankAttribute)
		d.Set("rank_order", job.Dispatch.RankOrder)
	} else {
		d.Set("max_thread_count", nil)
		d.Set("continue_on_error", nil)
		d.Set("rank_attribute", nil)
		d.Set("rank_order", nil)
	}

	d.Set("node_filter_query", nil)
	d.Set("node_filter_exclude_precedence", nil)
	if job.NodeFilter != nil {
		d.Set("node_filter_query", job.NodeFilter.Query)
		d.Set("node_filter_exclude_precedence", job.NodeFilter.ExcludePrecedence)
	}

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
			}
			optionConfigsI = append(optionConfigsI, optionConfigI)
		}
	}
	d.Set("option", optionConfigsI)

	commandConfigsI := []interface{}{}
	if job.CommandSequence != nil {
		d.Set("command_ordering_strategy", job.CommandSequence.OrderingStrategy)
		for _, command := range job.CommandSequence.Commands {
			commandConfigI := map[string]interface{}{
				"shell_command":    command.ShellCommand,
				"inline_script":    command.Script,
				"script_file":      command.ScriptFile,
				"script_file_args": command.ScriptFileArgs,
			}

			if command.Job != nil {
				commandConfigI["job"] = []interface{}{
					map[string]interface{}{
						"name":              command.Job.Name,
						"group_name":        command.Job.GroupName,
						"run_for_each_node": command.Job.RunForEachNode,
						"args":              command.Job.Arguments,
					},
				}
			}

			if command.StepPlugin != nil {
				commandConfigI["step_plugin"] = []interface{}{
					map[string]interface{}{
						"type":   command.StepPlugin.Type,
						"config": map[string]string(command.StepPlugin.Config),
					},
				}
			}

			if command.NodeStepPlugin != nil {
				commandConfigI["node_step_plugin"] = []interface{}{
					map[string]interface{}{
						"type":   command.NodeStepPlugin.Type,
						"config": map[string]string(command.NodeStepPlugin.Config),
					},
				}
			}

			commandConfigsI = append(commandConfigsI, commandConfigI)
		}
	}
	d.Set("command", commandConfigsI)

	return nil
}
