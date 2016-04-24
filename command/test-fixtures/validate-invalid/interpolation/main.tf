variable "otherresourcename" {
  default = "aws_instance.web1"
}

variable "vairable_with_interpolation" {
  default = "${var.otherresourcename}"
}

resource "aws_instance" "web" {
  depends_on = ["${var.otherresourcename}}"]
}
