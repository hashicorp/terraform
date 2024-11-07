variable "beep" {
  type    = string
  default = "BEEP"
}

output "beep" {
  type  = string
  value = var.beep
}

stack "specified" {
  source = "./child"
  inputs = {
    boop = var.beep
  }
}

stack "defaulted" {
  source = "./child"
  inputs = {}
}

output "specified" {
  type = string
  value = stack.specified.result
}

output "defaulted" {
  type = string
  value = stack.defaulted.result
}
