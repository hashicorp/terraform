
locals {
  name = "jackson"
  childName = stack.child.outputted_name
  functional = format("Hello, %s!", "Ander")
  mappy = {
    name = "jackson",
    age = 30
  }

  listy = ["jackson", 30]
  booleany = true
  conditiony = local.booleany == true ? "true" : "false"
}

stack "child" {
  source = "./child"

  inputs = {
    name = "child of ${local.name}"
  }
}
