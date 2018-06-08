resource "test_instance" "foo" {
}

output "foo_id" {
  value = "${test_instance.foo.id}"
}

resource "test_instance" "bar" {
  count = 3
}

output "bar_count" {
  value = "${test_instance.bar.count}"
}
