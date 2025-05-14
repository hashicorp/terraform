resource "test_instance" "foo" {
  ami = "bar"
}

check "test_instance_exists" {
  assert {
    condition = test_instance.foo.id != null
    error_message = "value should have been computed"
  }
}
