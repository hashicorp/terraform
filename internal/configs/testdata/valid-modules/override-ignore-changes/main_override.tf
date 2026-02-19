resource "test_instance" "foo" {
  foo = "bar"
  lifecycle {
    ignore_changes = all
  }
}
