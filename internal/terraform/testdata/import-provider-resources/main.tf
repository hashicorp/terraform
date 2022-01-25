provider "aws" {
    value = "${test_instance.bar.value}"
}

resource "aws_instance" "foo" {
    bar = "value"
}

resource "test_instance" "bar" {
    value = "yes"
}
