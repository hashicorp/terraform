stack "sensitive" {
  source = "../sensitive-output"

  inputs = {
  }
}

component "self" {
  source = "./"

  inputs = {
    secret = stack.sensitive.result
  }
}

output "result" {
  type  = string
  value = component.self.result
}
