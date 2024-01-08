stack "child" {
  source = "../sensitive-output"

  inputs = {
  }
}

output "result" {
  type  = string
  value = stack.child.result
}
