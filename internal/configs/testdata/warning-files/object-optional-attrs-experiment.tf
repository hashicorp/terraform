terraform {
  experiments = [
    module_variable_optional_attrs, # WARNING: Experimental feature "module_variable_optional_attrs" is active
  ]
}

variable "a" {
  type = object({
    foo = optional(string)
    bar = optional(bool, true)
  })
}

variable "b" {
  type = list(
    object({
      foo = optional(string)
    })
  )
}

variable "c" {
  type = set(
    object({
      foo = optional(string)
    })
  )
}

variable "d" {
  type = map(
    object({
      foo = optional(string)
    })
  )
}
