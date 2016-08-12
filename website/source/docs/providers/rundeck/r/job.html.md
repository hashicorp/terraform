---
layout: "rundeck"
page_title: "Rundeck: rundeck_job"
sidebar_current: "docs-rundeck-resource-job"
description: |-
  The rundeck_job resource allows Rundeck jobs to be managed by Terraform.
---

<!-- TOC depthFrom:1 depthTo:6 withLinks:1 updateOnSave:1 orderedList:0 -->

- [rundeck\_job](#rundeckjob)
	- [Example Usage](#example-usage)
	- [Attributes Reference](#attributes-reference)
	- [Argument Reference](#argument-reference)
		- [Dispatch](#dispatch)
			- [Example Usage](#example-usage)
			- [Argument Reference](#argument-reference)
		- [Option](#option)
			- [Example Usage](#example-usage)
			- [Argument Reference](#argument-reference)
		- [Command](#command)
			- [Example Usage](#example-usage)
			- [Argument Reference](#argument-reference)
				- [Job Argument Reference](#job-argument-reference)
					- [Example Usage](#example-usage)
					- [Argument Reference](#argument-reference)
				- [(Node) Step Plugin Argument Reference](#node-step-plugin-argument-reference)
					- [Example Usage](#example-usage)
					- [Argument Reference](#argument-reference)
				- [Errorhandler Argument Reference](#errorhandler-argument-reference)
					- [Example Usage](#example-usage)
		- [Notification](#notification)
			- [Example Usage](#example-usage)
			- [Argument Reference](#argument-reference)
				- [Email Argument Reference](#email-argument-reference)
				- [WebHook Argument Reference](#webhook-argument-reference)

<!-- /TOC -->

# rundeck\_job

The job resource allows Rundeck jobs to be managed by Terraform. In Rundeck a job is a particular
named set of steps that can be executed against one or more of the nodes configured for its
associated project.

Each job belongs to a project. A project can be created with the `rundeck_project` resource.

## Example Usage

```terraform
resource "rundeck_job" "bounceweb" {
    name = "Bounce Web Servers"
    project_name = "anvils"
    node_filter_query = "tags: web"
    description = "Restart the service daemons on all the web servers"

    command {
        shell_command = "sudo service anvils restart"
    }
}
```

## Attributes Reference

The following attribute is exported:

* `id` - A unique identifier for the job.

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the job, used to describe the job in the Rundeck UI.

* `description` - (Required) A longer description of the job, describing the job in the Rundeck UI.

* `project_name` - (Required) The name of the project that this job should belong to.

* `group_name` - (Optional) The name of a group within the project in which to place the job.
  Setting this creates collapsable subcategories within the Rundeck UI's project job index.

* `log_level` - (Optional) The log level that Rundeck should use for this job. Defaults to "INFO".

* `allow_concurrent_executions` - (Optional) Boolean defining whether two or more executions of
  this job can run concurrently. The default is `false`, meaning that jobs will only run
  sequentially.

* `max_thread_count` - (Optional) The maximum number of threads to use to execute this job, which
  controls on how many nodes the commands can be run simulateneously. Defaults to 1, meaning that
  the nodes will be visited sequentially.

* `continue_on_error` - (Optional) This manages what to do if a step incurs and error:
  `false`: Stop at the failed step: Fail immediately (default).
  `true`: Run remaining steps before failing: Continue to next steps and fail the job at the end.

* `rank_attribute` - (Optional) The name of the attribute that will be used to decide in which
  order the nodes will be visited while executing the job across multiple nodes.

* `rank_order` - (Optional) Keyword deciding which direction the nodes are sorted in terms of
  the chosen `rank_attribute`. May be either "ascending" (the default) or "descending".

* `preserve_options_order`: (Optional) Boolean controlling whether the configured options will
  be presented in their configuration order when shown in the Rundeck UI. The default is `false`,
  which means that the options will be displayed in alphabetical order by name.

* `command_ordering_strategy`: (Optional) The name of the strategy used to describe how to
  traverse the matrix of nodes and commands. The default is "node-first", meaning that all commands
  will be executed on a single node before moving on to the next. May also be set to "step-first",
  meaning that a single step will be executed across all nodes before moving on to the next step.

* `node_filter_query` - (Optional) A query string using
  [Rundeck's node filter language](http://rundeck.org/docs/manual/node-filters.html#node-filter-syntax)
  that defines which subset of the project's nodes will be used to execute this job.

* `node_filter_exclude_precedence`: (Optional) Boolean controlling a deprecated Rundeck feature that controls
  whether node exclusions take priority over inclusions.

* `nodes_selected_by_default`: (Optional) Boolean controlling the user has to explicitly select target nodes.

* `schedule_cron`: (Optional) Schedule a cronjob.

* `execution_timeout`: (Optional) The maximum time for an execution to run. Time in seconds, or specify time units: "120m", "2h", "3d". Use blank or 0 to indicate no timeout. Can include option value references like "${option.timeout}". 

* `execution_retry`: (Optional) Maximum number of times to retry execution when this job is directly invoked. Retry will occur if the job fails or times out, but not if it is manually killed. Can use an option value reference like "${option.retry}". 

* `option`: (Optional) Nested block defining an option a user may set when executing this job. A
  job may have any number of options. The structure of this nested block is described below.

* `command`: (Required) Nested block defining one step in the job workflow. A job must have one or
  more commands. The structure of this nested block is described below.

* `dispatch`: (Optional) Nested block defining if the commands should be dispatch to the nodes. A job must have one dispatch item. The structure of this nested block is described below.

* `notification`: (Optional) Nested block defining notifications for a job. A job must have one notification item. The structure of this nested block is described below.

---

### Dispatch

Documentation reference: [node-dispatching-and-filtering](http://rundeck.org/docs/manual/jobs.html#node-dispatching-and-filtering)

#### Example Usage

```terraform
dispatch {
    continue_on_error = true
}
```

#### Argument Reference
`dispatch` blocks have the following arguments:

* `continue_on_error` - (Optional) This manages what to do if a step incurs and error on a dispatched node:  
  `false`: Fail the step without running on any remaining nodes.  
  `true`: Continue running on any remaining nodes before failing the step.

---

### Option

Documentation reference: [job-options](http://rundeck.org/docs/manual/jobs.html#job-options)

#### Example Usage

```terraform
option {
    name = "type"
    description = "Type of webservice"
    default_value = "SOAP"
    value_choices = ["SOAP","RESTFUL"]
    require_predefined_choice = true
    required = true
    allow_multiple_values = true
    multi_value_delimiter = "|"
}
```

#### Argument Reference

`option` blocks have the following arguments:

* `name`: (Required) Unique name that will be shown in the UI when entering values and used as
  a variable name for template substitutions.

* `default_value`: (Optional) A default value for the option.

* `value_choices`: (Optional) A list of strings giving a set of predefined values that the user
  may choose from when entering a value for the option.

* `value_choices_url`: (Optional) Can be used instead of `value_choices` to cause Rundeck to
  obtain a list of choices dynamically by fetching this URL.

* `require_predefined_choice`: (Optional) Boolean controlling whether the user is allowed to
  enter values not included in the predefined set of choices (`false`, the default) or whether
  a predefined choice is required (`true`).

* `validation_regex`: (Optional) A regular expression that a provided value must match in order
  to be accepted.

* `description`: (Optional) A longer description of the option to be shown in the UI.

* `required`: (Optional) Boolean defining whether the user must provide a value for the option.
  Defaults to `false`.

* `allow_multiple_values`: (Optional) Boolean defining whether the user may select multiple values
  from the set of predefined values. Defaults to `false`, meaning that the user may choose only
  one value.

* `multi_value_delimiter`: (Optional) Delimiter used to join together multiple values into a single
  string when `allow_multiple_values` is set and the user chooses multiple values.

* `obscure_input`: (Optional) Boolean controlling whether the value of this option should be obscured
  during entry and in execution logs. Defaults to `false`, but should be set to `true` when the
  requested value is a password, private key or any other secret value.

* `storage_path`: (Optional) Key storage path for a default password value.

* `exposed_to_scripts`: (Optional) Boolean controlling whether the value of this option is available
  to scripts executed by job commands. Defaults to `false`.


---

### Command

Documentation reference: [workflow-steps](http://rundeck.org/docs/manual/jobs.html#workflow-steps)

#### Example Usage

```terraform
command {
  description = "I am being executed on the remote server."
  inline_script = "echo Hello World!"
  invocation_string = "sudo"
  arguments_quoted = true
  file_extension = "sh"
}
```

#### Argument Reference

`command` blocks must have any one of the following combinations of arguments as contents:

* `description`: (Optional) A longer description of the option to be shown in the UI.

* `continue_on_error`: (Optional) When using a command as an errorhandler, this option is available for use. If the Workflow keepgoing is false, this allows the Workflow to continue when the Error Handler is successful.

* `shell_command` gives a single shell command to execute on the nodes.

* `inline_script` gives a whole shell script, inline in the configuration, to execute on the nodes.

* `file_extension` The file extension is used by the script file when it is copied to the node. The `.` is optional. E.g.: .ps1, or abc

* `invocation_string` Specify how to invoke the script file.

* `arguments_quoted ` If arguments are quoted, then the arguments passed to the `invocation_string` will be quoted as one string.

* `script_file` and `script_file_args` together describe a script that is already pre-installed
  on the nodes which is to be executed.

* A `job` block, described below, causes another job within the same project to be executed as
  a command.

* A `step_plugin` block, described below, causes a step plugin to be executed as a command.

* A `node_step_plugin` block, described below, causes a node step plugin to be executed once for
  each node.

* A `errorhandler` bloc, described below. The error handler will execute if the step fails.

##### Job Argument Reference

A command's `job` block is the same as a `rundeck_job`, the arguments that are not needed will be ignored.  
Due to technical depth, the following commands are supported in this job block:
* name
* group_name
* args
* run_for_each_node
* node_filter_query
* max_thread_count

###### Example Usage

```terraform
command {
  description = "This will reference a job."
  job {
    name = "Bounce Web Servers"
    group_name = "anvils"
    run_for_each_node = true
    args = "--type SOAP"
    node_filter_query = "tags: web"
  }
}
```

###### Argument Reference

A command's `job` block has the following structure:

* `name`: (Required) The name of the job to execute. The target job must be in the same project
  as the current job.

* `group_name`: (Optional) The name of the group that the target job belongs to, if any.

* `run_for_each_node`: (Optional) Boolean controlling whether the job is run only once (`false`,
  the default) or whether it is run once for each node (`true`).

* `args`: (Optional) A string giving the arguments to pass to the target job, using
  [Rundeck's job arguments syntax](http://rundeck.org/docs/manual/jobs.html#job-reference-step).

* `node_filter_query` - (Optional) A query string using
  [Rundeck's node filter language](http://rundeck.org/docs/manual/node-filters.html#node-filter-syntax)
  that defines which subset of the project's nodes will be used to execute this job.

* `max_thread_count` - (Optional) The maximum number of threads to use to execute this job, which
  controls on how many nodes the commands can be run simulateneously. Defaults to 1, meaning that
  the nodes will be visited sequentially.

##### (Node) Step Plugin Argument Reference

A command's `step_plugin` or `node_step_plugin` block both have the following structure:

###### Example Usage

```terraform
command {
  description = "This will call the anvils plugin."
  step_plugin {
    type = "anvilsPlugin"
    config = {
      enable = true
    }
  }
}
```

###### Argument Reference

A command's `step_plugin` or `node_step_plugin` block both have the following structure:

* `type`: (Required) The name of the plugin to execute.

* `config`: (Optional) Map of arbitrary configuration parameters for the selected plugin.

##### Errorhandler Argument Reference

Error handler has the same arguments as `command`. With an additional argument: `continue_on_error`.  
http://rundeck.org/docs/manual/jobs.html#error-handlers

###### Example Usage

```terraform
command {
  description = "This will call the error handler."
  inline_script = "exit 666"
  errorhandler {
    inline_script = "echo I have resolved it."
    continue_on_error = true
  }
}
```

---

### Notification

Job notifications are messages triggered by a job event. You can configure notifications to occur when a Job Execution starts or finishes, with either success or failure. The same notifications are supported on each event.  
The current supported events are:
* onfailure
* onstart
* onsuccess

Documentation reference: [job-notifications](http://rundeck.org/docs/manual/jobs.html#job-notifications)

#### Example Usage

```terraform
notification {
  onfailure {
    email {
      subject = "I failed my command :("
      recipients = ["admin@bounceweb.io"]
      attach_log = true
    }
  }
  onsuccess {
    webhook_urls = ["http://slack.bouceweb.io"]
  }
}
```

#### Argument Reference

Supported arguments are:
* email
* webhook_urls

##### Email Argument Reference
`email` blocks have the following arguments:

* `subject` (Required) Template of the email subject.

* `recipients`  (Required) Email addresses.

* `attach_log` (Optionial) Attach output log.

##### WebHook Argument Reference

* `webhook_urls` (Required) WebHook Url list.
