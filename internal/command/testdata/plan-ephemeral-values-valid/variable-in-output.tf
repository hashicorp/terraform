variable "valid-eph" {
  ephemeral = true
  default   = "foo"
}

output "valid-eph" {
  ephemeral = true
  value     = var.valid-eph
}
