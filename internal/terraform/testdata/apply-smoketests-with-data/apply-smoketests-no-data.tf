terraform {
  experiments = [smoke_tests]

  required_providers {
    perchance = {
      source = "example.com/test/perchance"
    }
  }
}

variable "a" {
  type    = string
}

variable "b" {
  type    = string
}

variable "c" {
  type    = string
}

smoke_test "try" {
  precondition {
    condition     = var.a == "a"
    error_message = "A isn't."
  }

  data "perchance" "if_you_dont_mind" {
    b = var.b
    c = var.c
  }

  postcondition {
    condition     = data.perchance.if_you_dont_mind.splendid
    error_message = "Rather frightful, actually."
  }
}
