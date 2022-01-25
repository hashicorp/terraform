resource "aws_ami_list" "foo" {
  # assume this has a computed attr called "ids"
}

resource "aws_instance" "foo" {
  # this is erroneously referencing the list of all ids. The value of this
  # is unknown during plan, but we should still know that the unknown value
  # is a list of strings and so catch this during plan.
  ami = "${aws_ami_list.foo.ids}"
}
