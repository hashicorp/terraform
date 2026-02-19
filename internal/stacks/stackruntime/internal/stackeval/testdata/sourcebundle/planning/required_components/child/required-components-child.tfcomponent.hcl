variable "in" {
  type    = string
  default = ""
}

output "out" {
  type  = string
  value = var.in
}
