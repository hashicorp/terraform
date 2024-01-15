
variable "input" {
  type = string
}

output "output" {
  value = var.input

  precondition {
    condition = var.input == "something incredibly specific"
    error_message = "this should fail"
  }
}
