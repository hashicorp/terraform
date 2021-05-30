variable "input" {
  default = "Hello world"
}

output "notsensitive" {
  value = "${var.input}"
}

output "sensitive" {
  sensitive = true
  value = "${var.input}"
}
