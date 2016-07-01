resource "aws_ami_list" "foo" {
  # assume this has a computed attr called "ids"
}

resource "aws_instance" "foo" {
  # this is erroneously referencing the list of all ids, but
  # since it is a computed attr, the Validate walk won't catch
  # it.
  ami = "${aws_ami_list.foo.ids}"
}
