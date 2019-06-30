resource "aws_instance" "web" {
    count = "${count.index}"
}
