# Required
variable "foo" {
}

# Optional
variable "bar" {
  default = "baz"
}

# Mapping
variable "map" {
  default = {
    foo = "bar"
  }
}

# Complex Object Types
variable "object_map" {
  type = map(object({
    foo = string,
    bar = any
  }))
}

variable "object_list" {
  type = list(object({
    foo = string,
    bar = any
  }))
}
