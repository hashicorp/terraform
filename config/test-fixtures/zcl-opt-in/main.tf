#use-experimental-syntax

variable "foo" {
  # zcl should accept this expression and evaluate it, whereas hcl would
  # produce a syntax error.
  description = 1 + 2
}
