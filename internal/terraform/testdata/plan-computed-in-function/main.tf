data "aws_data_source" "foo" {

}

resource "aws_instance" "bar" {
    attr = "${length(data.aws_data_source.foo.computed)}"
}
