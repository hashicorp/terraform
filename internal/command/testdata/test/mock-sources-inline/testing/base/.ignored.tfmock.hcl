# If read, this file should cause issues. But, it should be ignored.

mock_resource "test_resource" {}

mock_data "test_resource" {}

override_resource {
  target = test_resource.foo
}
