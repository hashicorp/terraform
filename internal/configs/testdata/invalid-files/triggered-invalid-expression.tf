resource "test_resource" "a" {
  count = 1
  lifecycle {
    replace_triggered_by = [ not_a_reference ]
  }
}
