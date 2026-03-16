variable "msg" {
  type    = string
  default = "default"
}

stack "child" {
  source = "../variable-output-roundtrip"

  inputs = {
    msg = var.msg
  }
}

output "msg" {
  type  = string
  value = stack.child.msg
}
