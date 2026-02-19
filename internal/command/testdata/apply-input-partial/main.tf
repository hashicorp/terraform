variable "foo" {}
variable "bar" {}

output "foo" {
  value = "${var.foo}"
}
output "bar" {
  value = "${var.bar}"
}
