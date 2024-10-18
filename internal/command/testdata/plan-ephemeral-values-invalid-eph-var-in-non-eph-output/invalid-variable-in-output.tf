variable "eph" {
  ephemeral = true
  default   = "foo"
}

output "not-eph" {
  value = var.eph
}
