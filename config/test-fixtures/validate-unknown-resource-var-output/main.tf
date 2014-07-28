resource "aws_instance" "web" {
}

output "ip" {
  value = "${aws_instance.loadbalancer.foo}"
}
