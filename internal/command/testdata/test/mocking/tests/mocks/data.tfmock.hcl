mock_resource "test_resource" {
  override_during = plan
  defaults = {
    id = "aaaa"
  }
}
