data "aws_data_source" "foo" {
    compute = "value"
}

resource "aws_instance" "bar" {
    count = "${data.aws_data_source.foo.value}"
}
