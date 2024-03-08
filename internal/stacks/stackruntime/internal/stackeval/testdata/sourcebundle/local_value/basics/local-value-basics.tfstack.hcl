
locals {
  name = "jackson"
  childName = stack.child.outputted_name
}

stack "child" {
  source = "./child"

  inputs = {
    name = "child of ${local.name}"
  }
}
