variable "just_any" {
  type = inferred
}

variable "map_of_any" {
  type = map(inferred)
}

variable "list_of_any" {
  type = list(inferred)
}

variable "set_of_any" {
  type = list(inferred)
}

variable "object_attr_with_any" {
  type = object({
    any     = inferred
    not_any = string
  })
}

variable "object_attr_with_optional_any" {
  type = object({
    any     = optional(inferred)
    not_any = optional(string)
  })
}

variable "tuple_elem_with_any" {
  type = tuple([inferred, string])
}

variable "map_of_object_with_any" {
  type = map(object({
    any = inferred
  }))
}
