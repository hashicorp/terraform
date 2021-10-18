resource "test_instance" "baz" {
  ami = "baz"
}

terraform {
  experiments = [ config_driven_move ]
}

moved {
  from = test_instance.foo
  to   = test_instance.baz
}
