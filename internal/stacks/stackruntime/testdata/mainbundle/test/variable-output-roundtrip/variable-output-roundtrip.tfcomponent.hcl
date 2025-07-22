variable "msg" {
  type    = string
  default = "default"
}

output "msg" {
  type  = string
  value = var.msg
}
