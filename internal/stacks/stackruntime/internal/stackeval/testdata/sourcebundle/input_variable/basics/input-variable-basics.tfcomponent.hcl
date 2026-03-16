
variable "name" {
  type = string
}

stack "child" {
  source = "./child"

  inputs = {
    name = "child of ${var.name}"
  }
}
