output "eph" {
  ephemeral = true
  value     = var.eph
}

output "not-eph" {
  value = "foo"
}
