variable "sample" {
  type = bool
  default = true
}

output "name" {
  value = var.sample
}