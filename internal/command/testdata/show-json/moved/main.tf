resource "test_instance" "baz" {
  ami = "baz"
}

moved {
  from = test_instance.foo
  to   = test_instance.baz
}
