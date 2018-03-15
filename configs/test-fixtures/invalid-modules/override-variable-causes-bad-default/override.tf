variable "foo" {
  type = string
  # Since we didn't also override the default, this is now invalid because
  # the existing default is not compatible with "string".
}
