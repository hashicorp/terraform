provider "test" {
    foo = "bar"

    alias = "alias"
}

resource "test_instance" "foo" {
}
