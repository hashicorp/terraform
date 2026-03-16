component "self" {
  source = "./"
  inputs = {
  }
}

output "result" {
  type = string
  value = component.self.out
}
