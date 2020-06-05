resource "foo" "bar" {
    count = "${var.list}"
}
