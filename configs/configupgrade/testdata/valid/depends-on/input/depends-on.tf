resource "test_instance" "foo" {
  depends_on = [
    "test_instance.bar",
    "test_instance.bar.0",
    "test_instance.bar.*",
    "test_instance.bar.invalid",
    "data.test_instance.baz",
    "data.test_instance.baz.invalid",
    "module.foo.bar",
    "module.foo",
  ]
}

output "foo" {
  value      = "a"
  depends_on = ["test_instance.foo"]
}
