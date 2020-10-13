variable "foo" {
    default = "bar"
}

variable "snack" {
    default = "popcorn"
}

variable "secret_snack" {
    default   = "seaweed snacks"
    sensitive = true
}

locals {
    snack_bar = [var.snack, var.secret_snack]
}
