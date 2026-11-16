resource "test_instance" "b" {
  num = 2
}

data "test_data_source" "b" {
  foo = "b"
}

output "out" {
  value = test_instance.b.id
}
