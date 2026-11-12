
variable "name" {
  default = "world"
}

resource "local_file" "hello" {
  content  = "Hello, ${var.name}"
  filename = "${path.module}/hello.txt"
}


resource "null_resource" "test" {
  triggers = {
    greeting = "${local_file.hello.content}"
  }
}

output "greeting" {
  value = null_resource.test.triggers["greeting"]
}
