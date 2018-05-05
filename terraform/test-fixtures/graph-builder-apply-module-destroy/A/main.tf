variable "input" {}

resource "null_resource" "foo" {
  triggers {
    input = "${var.input}"
  }
}

output "output" {
  value = "${null_resource.foo.id}"
}
