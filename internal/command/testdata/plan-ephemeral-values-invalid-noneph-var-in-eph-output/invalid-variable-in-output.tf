variable "not-eph" {
  default = "foo"
}

output "eph" {
  ephemeral = true
  value     = var.not-eph
}
