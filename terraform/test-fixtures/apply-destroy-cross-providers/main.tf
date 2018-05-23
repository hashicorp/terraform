resource "aws_instance" "shared" {
}

module "child" {
    source = "./child"
    value = "${aws_instance.shared.id}"
}
