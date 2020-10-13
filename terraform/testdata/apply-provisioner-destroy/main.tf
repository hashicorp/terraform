resource "aws_instance" "foo" {
    for_each = var.input
    foo = "bar"

    provisioner "shell" {
        command = "create ${each.key} ${each.value}"
    }

    provisioner "shell" {
        when = "destroy"
        command  = "destroy ${each.key} ${self.foo}"
    }
}

variable "input" {
  type = map(string)
  default = {}
}
