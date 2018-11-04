# general options
complete -f -c terraform -l version -d 'Print version information'
complete -f -c terraform -l help -d 'Show help'

### apply
complete -f -c terraform -n '__fish_use_subcommand' -a apply -d 'Build or change infrastructure'
complete -f -c terraform -n '__fish_seen_subcommand_from apply' -o backup -d 'Path to backup the existing state file'
complete -f -c terraform -n '__fish_seen_subcommand_from apply' -o lock -d 'Lock the state file when locking is supported'
complete -f -c terraform -n '__fish_seen_subcommand_from apply' -o lock-timeout -d 'Duration to retry a state lock'
complete -f -c terraform -n '__fish_seen_subcommand_from apply' -o input -d 'Ask for input for variables if not directly set'
complete -f -c terraform -n '__fish_seen_subcommand_from apply' -o no-color -d 'If specified, output won\'t contain any color'
complete -f -c terraform -n '__fish_seen_subcommand_from apply' -o parallelism -d 'Limit the number of concurrent operations'
complete -f -c terraform -n '__fish_seen_subcommand_from apply' -o refresh -d 'Update state prior to checking for differences'
complete -f -c terraform -n '__fish_seen_subcommand_from apply' -o state -d 'Path to a Terraform state file'
complete -f -c terraform -n '__fish_seen_subcommand_from apply' -o state-out -d 'Path to write state'
complete -f -c terraform -n '__fish_seen_subcommand_from apply' -o target -d 'Resource to target'
complete -f -c terraform -n '__fish_seen_subcommand_from apply' -o var -d 'Set a variable in the Terraform configuration'
complete -f -c terraform -n '__fish_seen_subcommand_from apply' -o var-file -d 'Set variables from a file'

### console
complete -f -c terraform -n '__fish_use_subcommand' -a console -d 'Interactive console for Terraform interpolations'
complete -f -c terraform -n '__fish_seen_subcommand_from console' -o state -d 'Path to a Terraform state file'
complete -f -c terraform -n '__fish_seen_subcommand_from console' -o var -d 'Set a variable in the Terraform configuration'
complete -f -c terraform -n '__fish_seen_subcommand_from console' -o var-file -d 'Set variables from a file'

### destroy
complete -f -c terraform -n '__fish_use_subcommand' -a destroy -d 'Destroy Terraform-managed infrastructure'
complete -f -c terraform -n '__fish_seen_subcommand_from destroy' -o backup -d 'Path to backup the existing state file'
complete -f -c terraform -n '__fish_seen_subcommand_from destroy' -o force -d 'Don\'t ask for input for destroy confirmation'
complete -f -c terraform -n '__fish_seen_subcommand_from destroy' -o lock -d 'Lock the state file when locking is supported'
complete -f -c terraform -n '__fish_seen_subcommand_from destroy' -o lock-timeout -d 'Duration to retry a state lock'
complete -f -c terraform -n '__fish_seen_subcommand_from destroy' -o no-color -d 'If specified, output won\'t contain any color'
complete -f -c terraform -n '__fish_seen_subcommand_from destroy' -o parallelism -d 'Limit the number of concurrent operations'
complete -f -c terraform -n '__fish_seen_subcommand_from destroy' -o refresh -d 'Update state prior to checking for differences'
complete -f -c terraform -n '__fish_seen_subcommand_from destroy' -o state -d 'Path to a Terraform state file'
complete -f -c terraform -n '__fish_seen_subcommand_from destroy' -o state-out -d 'Path to write state'
complete -f -c terraform -n '__fish_seen_subcommand_from destroy' -o target -d 'Resource to target'
complete -f -c terraform -n '__fish_seen_subcommand_from destroy' -o var -d 'Set a variable in the Terraform configuration'
complete -f -c terraform -n '__fish_seen_subcommand_from destroy' -o var-file -d 'Set variables from a file'

### env
complete -f -c terraform -n '__fish_use_subcommand' -a env -d 'Environment management'
complete -f -c terraform -n '__fish_seen_subcommand_from env' -a list -d 'List environments'
complete -f -c terraform -n '__fish_seen_subcommand_from env' -a select -d 'Select an environment'
complete -f -c terraform -n '__fish_seen_subcommand_from env' -a new -d 'Create a new environment'
complete -f -c terraform -n '__fish_seen_subcommand_from env' -a delete -d 'Delete an existing environment'

### fmt
complete -f -c terraform -n '__fish_use_subcommand' -a fmt -d 'Rewrite config files to canonical format'
complete -f -c terraform -n '__fish_seen_subcommand_from fmt' -o list -d 'List files whose formatting differs'
complete -f -c terraform -n '__fish_seen_subcommand_from fmt' -o write -d 'Write result to source file'
complete -f -c terraform -n '__fish_seen_subcommand_from fmt' -o diff -d 'Display diffs of formatting changes'

### get
complete -f -c terraform -n '__fish_use_subcommand' -a get -d 'Download and install modules for the configuration'
complete -f -c terraform -n '__fish_seen_subcommand_from get' -o update -d 'Check modules for updates'
complete -f -c terraform -n '__fish_seen_subcommand_from get' -o no-color -d 'If specified, output won\'t contain any color'

### graph
complete -f -c terraform -n '__fish_use_subcommand' -a graph -d 'Create a visual graph of Terraform resources'
complete -f -c terraform -n '__fish_seen_subcommand_from graph' -o draw-cycles -d 'Highlight any cycles in the graph'
complete -f -c terraform -n '__fish_seen_subcommand_from graph' -o module-depth -d 'Depth of modules to show in the output'
complete -f -c terraform -n '__fish_seen_subcommand_from graph' -o no-color -d 'If specified, output won\'t contain any color'
complete -f -c terraform -n '__fish_seen_subcommand_from graph' -o type -d 'Type of graph to output'

### import
complete -f -c terraform -n '__fish_use_subcommand' -a import -d 'Import existing infrastructure into Terraform'
complete -f -c terraform -n '__fish_seen_subcommand_from import' -o backup -d 'Path to backup the existing state file'
complete -f -c terraform -n '__fish_seen_subcommand_from import' -o config -d 'Path to a directory of configuration files'
complete -f -c terraform -n '__fish_seen_subcommand_from import' -o input -d 'Ask for input for variables if not directly set'
complete -f -c terraform -n '__fish_seen_subcommand_from import' -o lock -d 'Lock the state file when locking is supported'
complete -f -c terraform -n '__fish_seen_subcommand_from import' -o lock-timeout -d 'Duration to retry a state lock'
complete -f -c terraform -n '__fish_seen_subcommand_from import' -o no-color -d 'If specified, output won\'t contain any color'
complete -f -c terraform -n '__fish_seen_subcommand_from import' -o provider -d 'Specific provider to use for import'
complete -f -c terraform -n '__fish_seen_subcommand_from import' -o state -d 'Path to a Terraform state file'
complete -f -c terraform -n '__fish_seen_subcommand_from import' -o state-out -d 'Path to write state'
complete -f -c terraform -n '__fish_seen_subcommand_from import' -o var -d 'Set a variable in the Terraform configuration'
complete -f -c terraform -n '__fish_seen_subcommand_from import' -o var-file -d 'Set variables from a file'

### init
complete -f -c terraform -n '__fish_use_subcommand' -a init -d 'Initialize a new or existing Terraform configuration'
complete -f -c terraform -n '__fish_seen_subcommand_from init' -o backend -d 'Configure the backend for this environment'
complete -f -c terraform -n '__fish_seen_subcommand_from init' -o backend-config -d 'Backend configuration'
complete -f -c terraform -n '__fish_seen_subcommand_from init' -o get -d 'Download modules for this configuration'
complete -f -c terraform -n '__fish_seen_subcommand_from init' -o input -d 'Ask for input if necessary'
complete -f -c terraform -n '__fish_seen_subcommand_from init' -o lock -d 'Lock the state file when locking is supported'
complete -f -c terraform -n '__fish_seen_subcommand_from init' -o lock-timeout -d 'Duration to retry a state lock'
complete -f -c terraform -n '__fish_seen_subcommand_from init' -o no-color -d 'If specified, output won\'t contain any color'
complete -f -c terraform -n '__fish_seen_subcommand_from init' -o force-copy -d 'Suppress prompts about copying state data'

### output
complete -f -c terraform -n '__fish_use_subcommand' -a output -d 'Read an output from a state file'
complete -f -c terraform -n '__fish_seen_subcommand_from output' -o state -d 'Path to the state file to read'
complete -f -c terraform -n '__fish_seen_subcommand_from output' -o no-color -d 'If specified, output won\'t contain any color'
complete -f -c terraform -n '__fish_seen_subcommand_from output' -o module -d 'Return the outputs for a specific module'
complete -f -c terraform -n '__fish_seen_subcommand_from output' -o json -d 'Print output in JSON format'

### plan
complete -f -c terraform -n '__fish_use_subcommand' -a plan -d 'Generate and show an execution plan'
complete -f -c terraform -n '__fish_seen_subcommand_from plan' -o destroy -d 'Generate a plan to destroy all resources'
complete -f -c terraform -n '__fish_seen_subcommand_from plan' -o detailed-exitcode -d 'Return detailed exit codes'
complete -f -c terraform -n '__fish_seen_subcommand_from plan' -o input -d 'Ask for input for variables if not directly set'
complete -f -c terraform -n '__fish_seen_subcommand_from plan' -o lock -d 'Lock the state file when locking is supported'
complete -f -c terraform -n '__fish_seen_subcommand_from plan' -o lock-timeout -d 'Duration to retry a state lock'
complete -f -c terraform -n '__fish_seen_subcommand_from plan' -o no-color -d 'If specified, output won\'t contain any color'
complete -f -c terraform -n '__fish_seen_subcommand_from plan' -o out -d 'Write a plan file to the given path'
complete -f -c terraform -n '__fish_seen_subcommand_from plan' -o parallelism -d 'Limit the number of concurrent operations'
complete -f -c terraform -n '__fish_seen_subcommand_from plan' -o refresh -d 'Update state prior to checking for differences'
complete -f -c terraform -n '__fish_seen_subcommand_from plan' -o state -d 'Path to a Terraform state file'
complete -f -c terraform -n '__fish_seen_subcommand_from plan' -o target -d 'Resource to target'
complete -f -c terraform -n '__fish_seen_subcommand_from plan' -o var -d 'Set a variable in the Terraform configuration'
complete -f -c terraform -n '__fish_seen_subcommand_from plan' -o var-file -d 'Set variables from a file'

### push
complete -f -c terraform -n '__fish_use_subcommand' -a push -d 'Upload this Terraform module to Atlas to run'
complete -f -c terraform -n '__fish_seen_subcommand_from push' -o atlas-address -d 'An alternate address to an Atlas instance'
complete -f -c terraform -n '__fish_seen_subcommand_from push' -o upload-modules -d 'Lock modules and upload completely'
complete -f -c terraform -n '__fish_seen_subcommand_from push' -o name -d 'Name of the configuration in Atlas'
complete -f -c terraform -n '__fish_seen_subcommand_from push' -o token -d 'Access token to use to upload'
complete -f -c terraform -n '__fish_seen_subcommand_from push' -o overwrite -d 'Variable keys that should overwrite values in Atlas'
complete -f -c terraform -n '__fish_seen_subcommand_from push' -o var -d 'Set a variable in the Terraform configuration'
complete -f -c terraform -n '__fish_seen_subcommand_from push' -o var-file -d 'Set variables from a file'
complete -f -c terraform -n '__fish_seen_subcommand_from push' -o vcs -d 'Upload only files committed to your VCS'
complete -f -c terraform -n '__fish_seen_subcommand_from push' -o no-color -d 'If specified, output won\'t contain any color'

### refresh
complete -f -c terraform -n '__fish_use_subcommand' -a refresh -d 'Update local state file against real resources'
complete -f -c terraform -n '__fish_seen_subcommand_from refresh' -o backup -d 'Path to backup the existing state file'
complete -f -c terraform -n '__fish_seen_subcommand_from refresh' -o input -d 'Ask for input for variables if not directly set'
complete -f -c terraform -n '__fish_seen_subcommand_from refresh' -o lock -d 'Lock the state file when locking is supported'
complete -f -c terraform -n '__fish_seen_subcommand_from refresh' -o lock-timeout -d 'Duration to retry a state lock'
complete -f -c terraform -n '__fish_seen_subcommand_from refresh' -o no-color -d 'If specified, output won\'t contain any color'
complete -f -c terraform -n '__fish_seen_subcommand_from refresh' -o state -d 'Path to a Terraform state file'
complete -f -c terraform -n '__fish_seen_subcommand_from refresh' -o state-out -d 'Path to write state'
complete -f -c terraform -n '__fish_seen_subcommand_from refresh' -o target -d 'Resource to target'
complete -f -c terraform -n '__fish_seen_subcommand_from refresh' -o var -d 'Set a variable in the Terraform configuration'
complete -f -c terraform -n '__fish_seen_subcommand_from refresh' -o var-file -d 'Set variables from a file'

### show
complete -f -c terraform -n '__fish_use_subcommand' -a show -d 'Inspect Terraform state or plan'
complete -f -c terraform -n '__fish_seen_subcommand_from show' -o no-color -d 'If specified, output won\'t contain any color'

### taint
complete -f -c terraform -n '__fish_use_subcommand' -a taint -d 'Manually mark a resource for recreation'
complete -f -c terraform -n '__fish_seen_subcommand_from taint' -o allow-missing -d 'Succeed even if resource is missing'
complete -f -c terraform -n '__fish_seen_subcommand_from taint' -o backup -d 'Path to backup the existing state file'
complete -f -c terraform -n '__fish_seen_subcommand_from taint' -o lock -d 'Lock the state file when locking is supported'
complete -f -c terraform -n '__fish_seen_subcommand_from taint' -o lock-timeout -d 'Duration to retry a state lock'
complete -f -c terraform -n '__fish_seen_subcommand_from taint' -o module -d 'The module path where the resource lives'
complete -f -c terraform -n '__fish_seen_subcommand_from taint' -o no-color -d 'If specified, output won\'t contain any color'
complete -f -c terraform -n '__fish_seen_subcommand_from taint' -o state -d 'Path to a Terraform state file'
complete -f -c terraform -n '__fish_seen_subcommand_from taint' -o state-out -d 'Path to write state'

### untaint
complete -f -c terraform -n '__fish_use_subcommand' -a untaint -d 'Manually unmark a resource as tainted'
complete -f -c terraform -n '__fish_seen_subcommand_from untaint' -o allow-missing -d 'Succeed even if resource is missing'
complete -f -c terraform -n '__fish_seen_subcommand_from untaint' -o backup -d 'Path to backup the existing state file'
complete -f -c terraform -n '__fish_seen_subcommand_from untaint' -o lock -d 'Lock the state file when locking is supported'
complete -f -c terraform -n '__fish_seen_subcommand_from untaint' -o lock-timeout -d 'Duration to retry a state lock'
complete -f -c terraform -n '__fish_seen_subcommand_from untaint' -o module -d 'The module path where the resource lives'
complete -f -c terraform -n '__fish_seen_subcommand_from untaint' -o no-color -d 'If specified, output won\'t contain any color'
complete -f -c terraform -n '__fish_seen_subcommand_from untaint' -o state -d 'Path to a Terraform state file'
complete -f -c terraform -n '__fish_seen_subcommand_from untaint' -o state-out -d 'Path to write state'

### validate
complete -f -c terraform -n '__fish_use_subcommand' -a validate -d 'Validate the Terraform files'
complete -f -c terraform -n '__fish_seen_subcommand_from validate' -o no-color -d 'If specified, output won\'t contain any color'

### version
complete -f -c terraform -n '__fish_use_subcommand' -a version -d 'Print the Terraform version'
