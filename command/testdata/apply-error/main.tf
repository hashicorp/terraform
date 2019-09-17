resource "test_instance" "foo" {
    ami = "bar"
}

resource "test_instance" "bar" {
    error = "true"
}
