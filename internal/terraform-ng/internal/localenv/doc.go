// Package localenv deals with the local-mode representation of environments,
// which is as .tfenv.hcl files containing the per-environment settings.
//
// These "environment definition files" serve as the local-mode analog to what
// would be a named object accessed via the Terraform Cloud API when using
// remote operations. CLI commands that would send updates to the remote API
// in remote mode will instead make surgical modifications to the definition
// file in local mode. Local-only users can therefore choose either to
// manipulate the files directly or to use the same CLI commands that a
// Terraform Cloud user would, with equivalent effect.
package localenv
