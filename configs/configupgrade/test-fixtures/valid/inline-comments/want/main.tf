variable "list" {
  type = list(string)

  default = [
    "foo", # I am a comment
    "bar", # I am also a comment
    "baz",
  ]
}
