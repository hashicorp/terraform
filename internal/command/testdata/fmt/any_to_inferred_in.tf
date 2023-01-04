variable "just_any" {
  type = any
}

variable "map_of_any" {
  type = map(any)
}

variable "list_of_any" {
  type = list(any)
}

variable "set_of_any" {
  type = list(any)
}

variable "object_attr_with_any" {
  type = object({
    any = any
    not_any = string
  })
}

variable "object_attr_with_optional_any" {
  type = object({
    any = optional(any)
    not_any = optional(string)
  })
}

variable "tuple_elem_with_any" {
  type = tuple([any, string])
}

variable "map_of_object_with_any" {
  type = map(object({
    any = any
  }))
}
