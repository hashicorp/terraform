resource "test_object" "b" {
  test_string = "foo"
}

output "output" {
  value = "${test_object.b.test_string}"
}
