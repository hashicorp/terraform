
component "single" {
  source = "./empty-module"
}

component "for_each" {
  source   = "./empty-module"
  for_each = {}
}

stack "single" {
  source = "./child"
}

stack "for_each" {
  source   = "./child"
  for_each = {}
}

variable "unused" {
  type = string
}

variable "unused_with_default" {
  type = string
  default = "default"
}

variable "ephemeral" {
  type      = string
  default   = null
  ephemeral = true
}

variable "sensitive" {
  type      = string
  default   = null
  sensitive = true
}

output "normal" {
  type  = string
  value = var.optional
}

output "ephemeral" {
  type      = string
  value     = var.ephemeral
  ephemeral = true
}

output "sensitive" {
  type      = string
  value     = var.sensitive
  sensitive = true
}
