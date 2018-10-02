data "aws_data_source" "foo" {
    optional_attr = "value"
}

resource "aws_instance" "bar" {
    attr = "${length(data.aws_data_source.foo.computed)}"
}
