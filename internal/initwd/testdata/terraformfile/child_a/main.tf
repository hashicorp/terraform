variable "v" {
  description = "in child_a module"
  default     = ""
}

output "hello" {
  value = "Hello from child_a!"
}
