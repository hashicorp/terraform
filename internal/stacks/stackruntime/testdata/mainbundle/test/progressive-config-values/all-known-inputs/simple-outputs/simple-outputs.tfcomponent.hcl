# Component with outputs that depend on managed resources
# These should trigger config value emission during apply

variable "static_input" {
  type = string
}

variable "prefix_input" {
  type = string
}

# Outputs will come from the .tf file