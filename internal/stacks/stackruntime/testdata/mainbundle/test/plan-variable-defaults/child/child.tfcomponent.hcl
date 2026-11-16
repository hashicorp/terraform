variable "boop" {
  type    = string
  default = "BOOP"
}

output "result" {
  type  = string
  value = var.boop
}