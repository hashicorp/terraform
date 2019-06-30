
resource "test_resource" "foo" {
  value = "${local.test}"
}

output "test" {
  value = "${local.test}"
}
