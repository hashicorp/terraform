
resource "test" "foo" {
  dynamic "foo" {
    for_each = []
    content {
      from = "override"
    }
  }
}
