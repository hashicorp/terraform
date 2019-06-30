variable "ami" {
  default = [ "ami", "abc123" ]
}

resource "aws_instance" "quotes" {
  ami = "${join(\",\", var.ami)}"
}
