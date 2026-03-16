
# A variable with a simple validation that does not reference the variable
# value inside the error_message.
variable "validated" {
  type = string

  validation {
    condition     = var.validated != "bad"
    error_message = "Value must not be 'bad'."
  }
}

# A variable whose error_message expression interpolates the variable value.
# When the input carries a sensitive or ephemeral mark, evaluating the
# interpolation causes the error_message result to inherit that mark — which
# is the behaviour we want to exercise.
variable "with_msg_ref" {
  type = string

  validation {
    condition     = var.with_msg_ref != "bad"
    error_message = "Got disallowed value '${var.with_msg_ref}'."
  }
}

# A variable with two validation blocks to verify that all rules are evaluated
# and all failures are reported independently.
#
# Inputs that trigger both failures simultaneously:
#   "bad" — length("bad") = 3 < 5 AND "bad" == "bad"
variable "multi_rule" {
  type = string

  validation {
    condition     = length(var.multi_rule) >= 5
    error_message = "Value must be at least 5 characters long."
  }

  validation {
    condition     = var.multi_rule != "bad"
    error_message = "Value must not be 'bad'."
  }
}
