typedef "custom_string" {
  definition = string
}

typedef "custom_object" {
  definition = object({
    a = number
    b = bool
    c = list(string)
    d = optional(string, "default val")
  })
}
