resource "test_instance" "foo" {
    ami = "bar"
}

output "endpoint" {
  value = "foo.example.com"
}
