resource "aws_instance" "web" {
}

output "ip" {
  value = "foo"
  another = "nope"
}
