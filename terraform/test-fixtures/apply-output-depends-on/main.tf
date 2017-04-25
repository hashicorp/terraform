resource "aws_instance" "foo" {}

output "value" {
    value = "result"

    depends_on = ["aws_instance.foo"]
}
