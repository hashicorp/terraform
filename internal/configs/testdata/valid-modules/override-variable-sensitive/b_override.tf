variable "false_true" {
}

variable "true_false" {
}

variable "false_false_true" {
  sensitive = true
}

variable "true_true_false" {
  sensitive = false
}

variable "false_true_false" {
  sensitive = false
}

variable "true_false_true" {
  sensitive = true
}
