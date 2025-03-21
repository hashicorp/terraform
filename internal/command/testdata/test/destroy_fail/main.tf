
resource "test_resource" "resource" {
  value        = "Hello, world!"
  destroy_fail = true
}

resource "test_resource" "another" {
  value        = "Hello, world!"
  destroy_fail = true
}