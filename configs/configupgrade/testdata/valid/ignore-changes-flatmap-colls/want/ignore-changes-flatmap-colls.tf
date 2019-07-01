resource "test_instance" "foo" {
  lifecycle {
    ignore_changes = [
      a,
      b,
    ]
  }
}
