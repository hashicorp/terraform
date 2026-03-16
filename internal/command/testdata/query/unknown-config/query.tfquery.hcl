list "test_instance" "example" {
  provider = test

  config {
    ami = uuid() // forces the config to be unknown at plan time
  }
}
