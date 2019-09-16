resource "foo" "bar" {
    default = "bar"
    description = "bar"
    count = "${var.bar}"
}
