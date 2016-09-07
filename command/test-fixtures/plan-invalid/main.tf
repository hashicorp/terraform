resource "test_instance" "foo" {
    count = 5
}

resource "test_instance" "bar" {
    count = "${length(test_instance.foo.*.id)}"
}
