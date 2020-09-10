terraform {
  experiments = [sensitive_variables] # WARNING: Experimental feature "sensitive_variables" is active
}

variable "sensitive-value" {
  sensitive = true
}