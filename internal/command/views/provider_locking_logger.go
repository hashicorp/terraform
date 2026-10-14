// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

type ProviderLockingLogger interface {
	LogProviderLockfileCreated()
	LogProviderLockfileUpdated()
}

const createdLockInfoHuman = `
Terraform has created a lock file [bold].terraform.lock.hcl[reset] to record the provider
selections it made above. Include this file in your version control repository
so that Terraform can guarantee to make the same selections by default when
you run "terraform init" in the future.`

// Only used for JSON output in the init view, which has JSON output implemented using the
// full human-readable message text.
const createdLockInfoJSON = `
Terraform has created a lock file .terraform.lock.hcl to record the provider
selections it made above. Include this file in your version control repository
so that Terraform can guarantee to make the same selections by default when
you run "terraform init" in the future.`

const dependenciesLockChangesInfo = `
Terraform has made some changes to the provider dependency selections recorded
in the .terraform.lock.hcl file. Review those changes and commit them to your
version control system if they represent changes you intended to make.`
