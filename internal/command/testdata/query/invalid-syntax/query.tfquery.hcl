list "test_instance" "example" {
  provider = test

  config {
    ami = "ami-12345"
  }
}


// resource type not supported in query files
resource "test_instance" "example" {
  provider = test
}
