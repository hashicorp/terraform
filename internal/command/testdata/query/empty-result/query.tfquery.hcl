list "test_instance" "example" {
  provider = test

  config {
    ami = "ami-12345"
  }
}

list "test_instance" "example2" {
  provider = test

  config {
    ami = "ami-nonexistent"
  }
}
