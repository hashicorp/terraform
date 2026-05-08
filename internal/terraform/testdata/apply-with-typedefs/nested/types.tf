typedef "custom_module_object_type" {
  definition = object({
    a = number
    b = bool
    c = set(string)
    d = optional(string, "default from nested module!")
  })
}
