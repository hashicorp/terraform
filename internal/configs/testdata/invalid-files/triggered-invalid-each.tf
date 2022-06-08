resource "test_resource" "a" {
  for_each = var.input
  lifecycle {
    // cannot use each.val
    replace_triggered_by = [ test_resource.b[each.val] ]
  }
}
