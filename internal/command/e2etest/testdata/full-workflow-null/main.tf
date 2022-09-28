
variable "name" {
  default = "world"
}

data "template_file" "test" {
  template = "Hello, $${name}"

  vars = {
    name = "${var.name}"
  }
}

resource "null_resource" "test" {
  triggers = {
    greeting = "${data.template_file.test.rendered}"
  }
}

output "greeting" {
  value = "${null_resource.test.triggers["greeting"]}"
}
