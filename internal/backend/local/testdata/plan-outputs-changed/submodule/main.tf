output "foo" {
  value = "bar"
}

variable "foo" {
  ephemeral = true
  type      = string
  default   = "eph-val"
}

output "existing_eph" {
  ephemeral = true
  value     = var.foo
}

output "new_ephemeral" {
  ephemeral = true
  value     = var.foo
}
