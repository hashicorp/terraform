terraform {
  experiments = [sensitive_variables]
}

variable "sensitive-value" {
  sensitive = "123" # must be boolean
}
