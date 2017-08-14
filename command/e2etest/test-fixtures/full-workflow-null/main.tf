
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

resource "null_resource" "no_store" {
  triggers = {
    secret_key = "SECRET"
  }

  lifecycle {
    no_store = ["triggers"]
  }
}

output "greeting" {
  value = "${null_resource.test.triggers["greeting"]}"
}
