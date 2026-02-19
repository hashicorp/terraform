output "out" {
  value     = sensitive("secret")
  sensitive = true
}
