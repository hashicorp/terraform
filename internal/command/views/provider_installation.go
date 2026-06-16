// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

type ProviderInstaller interface {
	LogProviderInstallationMessage(message ProviderInstallationMessageCode, params ...any)
}

type ProviderInstallerWithPrepareMessage interface {
	LogProviderInstallationMessage(message ProviderInstallationMessageCode, params ...any)
	MessagePreparer
}

type ProviderInstallationMessageCode string

var ProviderInstallationMessageRegistry map[ProviderInstallationMessageCode]Message = map[ProviderInstallationMessageCode]Message{
	"initializing_provider_plugin_message": {
		HumanValue: "\n[reset][bold]Initializing provider plugins...",
		JSONValue:  "Initializing provider plugins...",
	},
	"initializing_state_store_provider_plugin_message": {
		HumanValue: "\n[reset][bold]Initializing provider plugin for state store %q...",
		JSONValue:  "Initializing provider plugin for state store %q...",
	},
	"state_store_provider_interactive_approved_message": {
		HumanValue: "\n[reset][bold]The state store provider was approved by the user.",
		JSONValue:  "The state store provider was approved by the user.",
	},
	"state_store_provider_interactive_rejected_message": {
		HumanValue: "\n[reset][bold]The state store provider was rejected by the user.",
		JSONValue:  "The state store provider was rejected by the user.",
	},
	"state_store_provider_automation_approved_message": {
		HumanValue: "\n[reset][bold]The state store provider was approved automatically.",
		JSONValue:  "The state store provider was approved automatically.",
	},
	"lock_info": {
		HumanValue: previousLockInfoHuman,
		JSONValue:  previousLockInfoJSON,
	},
	"dependencies_lock_changes_info": {
		HumanValue: dependenciesLockChangesInfo,
		JSONValue:  dependenciesLockChangesInfo,
	},
	"provider_already_installed_message": {
		HumanValue: "- Using previously-installed %s v%s",
		JSONValue:  "%s v%s: Using previously-installed provider version",
	},
	"built_in_provider_available_message": {
		HumanValue: "- %s is built in to Terraform",
		JSONValue:  "%s is built in to Terraform",
	},
	"using_provider_from_cache_dir_info": {
		HumanValue: "- Using %s v%s from the shared cache directory",
		JSONValue:  "%s v%s: Using from the shared cache directory",
	},
	"installing_provider_message": {
		HumanValue: "- Installing %s v%s...",
		JSONValue:  "Installing provider version: %s v%s...",
	},
	"reusing_previous_version_info": {
		HumanValue: "- Reusing previous version of %s from the dependency lock file",
		JSONValue:  "%s: Reusing previous version from the dependency lock file",
	},
	"finding_matching_version_message": {
		HumanValue: "- Finding %s versions matching %q...",
		JSONValue:  "Finding matching versions for provider: %s, version_constraint: %q",
	},
	"finding_latest_version_message": {
		HumanValue: "- Finding latest version of %s...",
		JSONValue:  "%s: Finding latest version...",
	},
	"installed_provider_version_info": {
		HumanValue: "- Installed %s v%s (%s%s)",
		JSONValue:  "Installed provider version: %s v%s (%s%s)",
	},
	"partner_and_community_providers_message": {
		HumanValue: partnerAndCommunityProvidersInfo,
		JSONValue:  partnerAndCommunityProvidersInfo,
	},
}

const dependenciesLockChangesInfo = `
Terraform has made some changes to the provider dependency selections recorded
in the .terraform.lock.hcl file. Review those changes and commit them to your
version control system if they represent changes you intended to make.`

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

const partnerAndCommunityProvidersInfo = "\nPartner and community providers are signed by their developers.\n" +
	"If you'd like to know more about provider signing, you can read about it here:\n" +
	"https://developer.hashicorp.com/terraform/cli/plugins/signing"
