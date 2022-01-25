variable "list" {
}


output "result" {
  value = length(var.list)
}
