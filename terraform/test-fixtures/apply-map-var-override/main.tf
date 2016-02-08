variable "images" {
    default = {
        us-east-1 = "image-1234"
        us-west-2 = "image-4567"
    }
}

resource "aws_instance" "foo" {
    ami = "${lookup(var.images, "us-east-1")}"
}

resource "aws_instance" "bar" {
    ami = "${lookup(var.images, "us-west-2")}"
}
