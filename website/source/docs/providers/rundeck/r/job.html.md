---
layout: "rundeck"
page_title: "Rundeck: rundeck_job"
sidebar_current: "docs-rundeck-resource-job"
description: |-
  The rundeck_job resource allows Rundeck jobs to be managed by Terraform.
---

# rundeck\_job

The job resource allows Rundeck jobs to be managed by Terraform. In Rundeck a job is a particular
named set of steps that can be executed against one or more of the nodes configured for its
associated project.

Each job belongs to a project. A project can be created with the `rundeck_project` resource.

## Example Usage

```
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

* `continue_on_error` - (Optional) Boolean defining whether Rundeck will continue to run
  subsequent steps if any intermediate step fails. Defaults to `false`, meaning that execution
  will stop and the execution will be considered to have failed.

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

* `option`: (Optional) Nested block defining an option a user may set when executing this job. A
  job may have any number of options. The structure of this nested block is described below.

* `command`: (Required) Nested block defining one step in the job workflow. A job must have one or
  more commands. The structure of this nested block is described below.

`option` blocks have the following structure:

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

* `exposed_to_scripts`: (Optional) Boolean controlling whether the value of this option is available
  to scripts executed by job commands. Defaults to `false`.

`command` blocks must have any one of the following combinations of arguments as contents:

* `shell_command` gives a single shell command to execute on the nodes.

* `inline_script` gives a whole shell script, inline in the configuration, to execute on the nodes.

* `script_file` and `script_file_args` together describe a script that is already pre-installed
  on the nodes which is to be executed.

* A `job` block, described below, causes another job within the same project to be executed as
  a command.

* A `step_plugin` block, described below, causes a step plugin to be executed as a command.

* A `node_step_plugin` block, described below, causes a node step plugin to be executed once for
  each node.

A command's `job` block has the following structure:

* `name`: (Required) The name of the job to execute. The target job must be in the same project
  as the current job.

* `group_name`: (Optional) The name of the group that the target job belongs to, if any.

* `run_for_each_node`: (Optional) Boolean controlling whether the job is run only once (`false`,
  the default) or whether it is run once for each node (`true`).

* `args`: (Optional) A string giving the arguments to pass to the target job, using
  [Rundeck's job arguments syntax](http://rundeck.org/docs/manual/jobs.html#job-reference-step).

A command's `step_plugin` or `node_step_plugin` block both have the following structure:

* `type`: (Required) The name of the plugin to execute.

* `config`: (Optional) Map of arbitrary configuration parameters for the selected plugin.

## Attributes Reference

The following attribute is exported:

* `id` - A unique identifier for the job.
