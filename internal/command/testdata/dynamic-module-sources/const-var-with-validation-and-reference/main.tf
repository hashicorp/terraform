variable "const_var" {
  type  = string
  const = true

  # This validation block will be unknown during init because of the reference to var.non_const_var
  validation {
    condition     = var.non_const_var == var.const_var
    error_message = "The const_var variable must be equal the non_const_var variable"
  }
}

variable "non_const_var" {
  type = string
}

module "example" {
  source = "./modules/example"
  in     = var.const_var
}
