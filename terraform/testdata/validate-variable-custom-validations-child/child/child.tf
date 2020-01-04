# This feature is currently experimental.
# (If you're currently cleaning up after concluding the experiment,
# remember to also clean up similar references in the configs package
# under "invalid-files" and "invalid-modules".)
terraform {
  experiments = [variable_validation]
}

variable "test" {
  type = string

  validation {
    condition     = var.test != "nope"
    error_message = "Value must not be \"nope\"."
  }
}
