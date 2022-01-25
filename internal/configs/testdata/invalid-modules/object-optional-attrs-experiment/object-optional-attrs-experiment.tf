variable "a" {
  type = object({
    # The optional attributes experiment isn't enabled, so this isn't allowed.
    a = optional(string)
  })
}
