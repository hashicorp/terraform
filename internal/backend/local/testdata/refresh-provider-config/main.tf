resource "test_instance" "foo" {
    ami = "bar"
}

provider "test" {
    value = "foo"
}
