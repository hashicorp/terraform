provider "test" {
    foo = file("./data")
}

resource "test_instance" "foo" {
}
