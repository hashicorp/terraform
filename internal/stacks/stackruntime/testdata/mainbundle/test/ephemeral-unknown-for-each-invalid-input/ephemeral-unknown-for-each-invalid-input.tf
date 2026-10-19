variable "number" {
  type = string
}

locals {
  parsed = tonumber(var.number)
}

output "value" {
  value     = local.parsed
  ephemeral = true
}
