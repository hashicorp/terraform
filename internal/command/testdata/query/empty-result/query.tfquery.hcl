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
    // to force deterministic ordering in the result
    foo = list.test_instance.example.data[0].state.id
  }
}
