variable "list_empty_default" {
  type = list(object({
    required_attribute              = string,
    optional_attribute              = optional(string),
    optional_attribute_with_default = optional(string, "Hello, world!"),
  }))
  default = []
}

variable "list_no_default" {
  type = list(object({
    required_attribute              = string,
    optional_attribute              = optional(string),
    optional_attribute_with_default = optional(string, "Hello, world!"),
  }))
}

variable "nested_optional_object" {
  type = object({
    nested_object = optional(object({
      flag = optional(bool, false)
    }))
  })
  default = {}
}

variable "nested_optional_object_with_default" {
  type = object({
    nested_object = optional(object({
      flag = optional(bool, false)
    }))
  })
  default = {
    nested_object = {}
  }
}

variable "nested_optional_object_with_embedded_default" {
  type = object({
    nested_object = optional(object({
      flag = optional(bool, false)
    }), {})
  })
  default = {}
}


output "list_empty_default" {
  value = var.list_empty_default
}

output "list_no_default" {
  value = var.list_no_default
}

output "nested_optional_object" {
  value = var.nested_optional_object
}

output "nested_optional_object_with_default" {
  value = var.nested_optional_object_with_default
}

output "nested_optional_object_with_embedded_default" {
  value = var.nested_optional_object_with_embedded_default
}
