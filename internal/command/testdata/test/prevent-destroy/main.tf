
resource "test_resource" "resource" {
  lifecycle {
    // we should still be able to destroy this during tests.
    prevent_destroy = true
  }
}
