
variable "input" {
  type = number
}

variable "input2" {
  type = number
  ephemeral = true
  default = 0
}

output "output" {
  value = var.input > 5 ? var.input : null
}
