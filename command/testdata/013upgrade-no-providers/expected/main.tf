variable "x" {
  default = 3
}

variable "y" {
  default = 5
}

output "product" {
  value = var.x * var.y
}
