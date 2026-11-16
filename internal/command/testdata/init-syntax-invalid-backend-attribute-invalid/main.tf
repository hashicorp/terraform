
terraform {
  backend "local" {
    path = $invalid
  }
}

variable "input" {
  type = string
}
