
variable "v" {
  description = "in child_b module"
  default     = ""
}

output "hello" {
  value = "Hello from child_b!"
}
