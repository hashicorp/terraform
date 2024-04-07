// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The Init view is used for the init command.
type Init interface {
	Diagnostics(diags tfdiags.Diagnostics)
	Output(messageCode string, params ...any)
	Log(messageCode string, params ...any)
	PrepareMessage(messageCode string, params ...any) string
}

// NewInit returns Init implementation for the given ViewType.
func NewInit(vt arguments.ViewType, view *View) Init {
	switch vt {
	case arguments.ViewJSON:
		return &InitJSON{
			view: NewJSONView(view),
		}
	case arguments.ViewHuman:
		return &InitHuman{
			view: view,
		}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// The InitHuman implementation renders human-readable text logs, suitable for
// a scrolling terminal.
type InitHuman struct {
	view *View
}

var _ Init = (*InitHuman)(nil)

func (v *InitHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *InitHuman) Output(messageCode string, params ...any) {
	v.view.streams.Println(v.PrepareMessage(messageCode, params...))
}

func (v *InitHuman) Log(messageCode string, params ...any) {
	v.view.streams.Println(v.PrepareMessage(messageCode, params...))
}

func (v *InitHuman) PrepareMessage(messageCode string, params ...any) string {
	message, ok := MessageRegistry[messageCode]
	if !ok {
		// display the message code as fallback if not found in the message registry
		return messageCode
	}

	if message.HumanValue == "" {
		// no need to apply colorization if the message is empty
		return message.HumanValue
	}

	return v.view.colorize.Color(strings.TrimSpace(fmt.Sprintf(message.HumanValue, params...)))
}

// The InitJSON implementation renders streaming JSON logs, suitable for
// integrating with other software.
type InitJSON struct {
	view *JSONView
}

var _ Init = (*InitJSON)(nil)

func (v *InitJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *InitJSON) Output(messageCode string, params ...any) {
	current_timestamp := time.Now().UTC().Format(time.RFC3339)

	json_data := map[string]string{
		"@level":     "info",
		"@message":   v.PrepareMessage(messageCode, params...),
		"@module":    "terraform.ui",
		"@timestamp": current_timestamp,
		"type":       "init_output"}

	init_output, _ := json.Marshal(json_data)
	v.view.view.streams.Println(string(init_output))
}

func (v *InitJSON) Log(messageCode string, params ...any) {
	v.view.Log(v.PrepareMessage(messageCode, params...))
}

func (v *InitJSON) PrepareMessage(messageCode string, params ...any) string {
	message, ok := MessageRegistry[messageCode]
	if !ok {
		// display the message code as fallback if not found in the message registry
		return messageCode
	}

	return strings.TrimSpace(fmt.Sprintf(message.JSONValue, params...))
}

// InitMessage represents a message string in both json and human decorated text format.
type InitMessage struct {
	HumanValue string
	JSONValue  string
}

var MessageRegistry map[string]InitMessage = map[string]InitMessage{
	"copying_configuration_message": {
		HumanValue: "[reset][bold]Copying configuration[reset] from %q...",
		JSONValue:  "Copying configuration from %q...",
	},
	"output_init_empty_message": {
		HumanValue: outputInitEmpty,
		JSONValue:  outputInitEmptyJSON,
	},
	"output_init_success_message": {
		HumanValue: outputInitSuccess,
		JSONValue:  outputInitSuccessJSON,
	},
	"output_init_success_cloud_message": {
		HumanValue: outputInitSuccessCloud,
		JSONValue:  outputInitSuccessCloudJSON,
	},
	"output_init_success_cli_message": {
		HumanValue: outputInitSuccessCLI,
		JSONValue:  outputInitSuccessCLI_JSON,
	},
	"output_init_success_cli_cloud_message": {
		HumanValue: outputInitSuccessCLICloud,
		JSONValue:  outputInitSuccessCLICloudJSON,
	},
	"upgrading_modules_message": {
		HumanValue: "[reset][bold]Upgrading modules...",
		JSONValue:  "Upgrading modules...",
	},
	"initializing_modules_message": {
		HumanValue: "[reset][bold]Initializing modules...",
		JSONValue:  "Initializing modules...",
	},
	"initializing_terraform_cloud_message": {
		HumanValue: "\n[reset][bold]Initializing Terraform Cloud...",
		JSONValue:  "Initializing Terraform Cloud...",
	},
	"initializing_backend_message": {
		HumanValue: "\n[reset][bold]Initializing the backend...",
		JSONValue:  "Initializing the backend...",
	},
	"initializing_provider_plugin_message": {
		HumanValue: "\n[reset][bold]Initializing provider plugins...",
		JSONValue:  "Initializing provider plugins...",
	},
	"dependencies_lock_changes_info": {
		HumanValue: dependenciesLockChangesInfo,
		JSONValue:  dependenciesLockChangesInfo,
	},
	"lock_info": {
		HumanValue: previousLockInfoHuman,
		JSONValue:  previousLockInfoJSON,
	},
	"provider_already_installed_message": {
		HumanValue: "- Using previously-installed %s v%s",
		JSONValue:  "- Using previously-installed %s v%s",
	},
	"built_in_provider_available_message": {
		HumanValue: "- %s is built in to Terraform",
		JSONValue:  "- %s is built in to Terraform",
	},
	"reusing_previous_version_info": {
		HumanValue: "- Reusing previous version of %s from the dependency lock file",
		JSONValue:  "- Reusing previous version of %s from the dependency lock file",
	},
	"finding_matching_version_message": {
		HumanValue: "- Finding %s versions matching %q...",
		JSONValue:  "- Finding %s versions matching %q...",
	},
	"finding_latest_version_message": {
		HumanValue: "- Finding latest version of %s...",
		JSONValue:  "- Finding latest version of %s...",
	},
	"using_provider_from_cache_dir_info": {
		HumanValue: "- Using %s v%s from the shared cache directory",
		JSONValue:  "- Using %s v%s from the shared cache directory",
	},
	"installing_provider_message": {
		HumanValue: "- Installing %s v%s...",
		JSONValue:  "- Installing %s v%s...",
	},
	"key_id": {
		HumanValue: ", key ID [reset][bold]%s[reset]",
		JSONValue:  ", key ID %s",
	},
	"installed_provider_version_info": {
		HumanValue: "- Installed %s v%s (%s%s)",
		JSONValue:  "- Installed %s v%s (%s%s)",
	},
	"partner_and_community_providers_message": {
		HumanValue: partnerAndCommunityProvidersInfo,
		JSONValue:  partnerAndCommunityProvidersInfo,
	},
	"init_config_error": {
		HumanValue: errInitConfigError,
		JSONValue:  errInitConfigErrorJSON,
	},
	"empty_message": {
		HumanValue: "",
		JSONValue:  "",
	},
}

const (
	CopyingConfigurationMessage         string = "copying_configuration_message"
	EmptyMessage                        string = "empty_message"
	OutputInitEmptyMessage              string = "output_init_empty_message"
	OutputInitSuccessMessage            string = "output_init_success_message"
	OutputInitSuccessCloudMessage       string = "output_init_success_cloud_message"
	OutputInitSuccessCLIMessage         string = "output_init_success_cli_message"
	OutputInitSuccessCLICloudMessage    string = "output_init_success_cli_cloud_message"
	UpgradingModulesMessage             string = "upgrading_modules_message"
	InitializingTerraformCloudMessage   string = "initializing_terraform_cloud_message"
	InitializingModulesMessage          string = "initializing_modules_message"
	InitializingBackendMessage          string = "initializing_backend_message"
	InitializingProviderPluginMessage   string = "initializing_provider_plugin_message"
	LockInfo                            string = "lock_info"
	DependenciesLockChangesInfo         string = "dependencies_lock_changes_info"
	ProviderAlreadyInstalledMessage     string = "provider_already_installed_message"
	BuiltInProviderAvailableMessage     string = "built_in_provider_available_message"
	ReusingPreviousVersionInfo          string = "reusing_previous_version_info"
	FindingMatchingVersionMessage       string = "finding_matching_version_message"
	FindingLatestVersionMessage         string = "finding_latest_version_message"
	UsingProviderFromCacheDirInfo       string = "using_provider_from_cache_dir_info"
	InstallingProviderMessage           string = "installing_provider_message"
	KeyID                               string = "key_id"
	InstalledProviderVersionInfo        string = "installed_provider_version_info"
	PartnerAndCommunityProvidersMessage string = "partner_and_community_providers_message"
	InitConfigError                     string = "init_config_error"
)

const outputInitEmpty = `
[reset][bold]Terraform initialized in an empty directory![reset]

The directory has no Terraform configuration files. You may begin working
with Terraform immediately by creating Terraform configuration files.
`

const outputInitEmptyJSON = `
Terraform initialized in an empty directory!

The directory has no Terraform configuration files. You may begin working
with Terraform immediately by creating Terraform configuration files.
`

const outputInitSuccess = `
[reset][bold][green]Terraform has been successfully initialized![reset][green]
`

const outputInitSuccessJSON = `
Terraform has been successfully initialized!
`

const outputInitSuccessCloud = `
[reset][bold][green]Terraform Cloud has been successfully initialized![reset][green]
`

const outputInitSuccessCloudJSON = `
Terraform Cloud has been successfully initialized!
`

const outputInitSuccessCLI = `[reset][green]
You may now begin working with Terraform. Try running "terraform plan" to see
any changes that are required for your infrastructure. All Terraform commands
should now work.

If you ever set or change modules or backend configuration for Terraform,
rerun this command to reinitialize your working directory. If you forget, other
commands will detect it and remind you to do so if necessary.
`

const outputInitSuccessCLI_JSON = `
You may now begin working with Terraform. Try running "terraform plan" to see
any changes that are required for your infrastructure. All Terraform commands
should now work.

If you ever set or change modules or backend configuration for Terraform,
rerun this command to reinitialize your working directory. If you forget, other
commands will detect it and remind you to do so if necessary.
`

const outputInitSuccessCLICloud = `[reset][green]
You may now begin working with Terraform Cloud. Try running "terraform plan" to
see any changes that are required for your infrastructure.

If you ever set or change modules or Terraform Settings, run "terraform init"
again to reinitialize your working directory.
`

const outputInitSuccessCLICloudJSON = `
You may now begin working with Terraform Cloud. Try running "terraform plan" to
see any changes that are required for your infrastructure.

If you ever set or change modules or Terraform Settings, run "terraform init"
again to reinitialize your working directory.
`

const previousLockInfoHuman = `
Terraform has created a lock file [bold].terraform.lock.hcl[reset] to record the provider
selections it made above. Include this file in your version control repository
so that Terraform can guarantee to make the same selections by default when
you run "terraform init" in the future.`

const previousLockInfoJSON = `
Terraform has created a lock file .terraform.lock.hcl to record the provider
selections it made above. Include this file in your version control repository
so that Terraform can guarantee to make the same selections by default when
you run "terraform init" in the future.`

const dependenciesLockChangesInfo = `
Terraform has made some changes to the provider dependency selections recorded
in the .terraform.lock.hcl file. Review those changes and commit them to your
version control system if they represent changes you intended to make.`

const partnerAndCommunityProvidersInfo = "\nPartner and community providers are signed by their developers.\n" +
	"If you'd like to know more about provider signing, you can read about it here:\n" +
	"https://www.terraform.io/docs/cli/plugins/signing.html"

const errInitConfigError = `
[reset]Terraform encountered problems during initialisation, including problems
with the configuration, described below.

The Terraform configuration must be valid before initialization so that
Terraform can determine which modules and providers need to be installed.
`

const errInitConfigErrorJSON = `
Terraform encountered problems during initialisation, including problems
with the configuration, described below.

The Terraform configuration must be valid before initialization so that
Terraform can determine which modules and providers need to be installed.
`
