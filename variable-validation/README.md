# Terraform v0.13 Beta: Custom validation rules for module variables

Terraform v0.13 introduces a new feature to allow module developers to declare
custom validation rules for any input variable, using a new `validation` block
inside a `variable` block:

```hcl
variable "ami_id" {
  type = string

  validation {
    condition     = can(regex("^ami-", var.example))
    error_message = "Must be an AMI id, starting with \"ami-\"."
  }
}
```

This feature was actually originally introduced as experimental in a v0.12
minor release, but as of v0.13 it is now considered a stable feature and no
longer requires explicitly opting in to the associated experiment.

This directory contains a configuration showing a simple example of variable
validation that is intentionally written to fail `terraform apply` and
illustrate the effect of violating a validation constraint.

Try adapting this example to be a hypothetical solution to some real-world
validation scenarios you've seen in your own modules. We'd love to hear about
your experiences, including any bugs or rough edges you encounter.

## Usage Notes

### Feedback from the earlier experiment

Because this feature was already available behind an experimental feature
gate in v0.12 releases, there has already been
[feedback on its capabilities](https://github.com/hashicorp/terraform/labels/experiment%2Fvariable-validation).

We've elected to initially release the feature exactly as it was implemented
under the experiment because all of the feedback we received describes
backward-compatible enhancements and so we intend to respond to those in
later releases. If you have suggestions for possible enhancements to the
capabilities of this feature, please check the existing experiment feedback
first to see if there's already an issue covering a similar suggestion.

### For input variable validation only

There have been prior discussions about a related concern in Terraform module
development of writing out explicit _internal assertions_ that Terraform
would check inside a module _during its processing_. While that has some similar
building blocks to custom variable validation -- condition expressions and
error messages -- we are not considering custom variable validation to be a
solution to both problems.

Internal assertions remain an open design area and are likely to be addressed
in some way in later Terraform releases. Therefore we'd ask that feedback about
custom variable validation be focused on the needs of providing helpful
feedback to callers of a module about the given arguments. Other error
conditions, such as a data resource returning data in an unexpected format
inside a module, are not in scope for this feature and may be addressed by
other features later.

