variable "list" {
  type = list(string)

  default = [
    "foo", # I am a comment
    "bar", # I am also a comment
    "baz",
  ]
}

variable "list2" {
  type = list(string)

  default = [
    "foo",
    "bar",
    "baz",
  ]
}

variable "list_the_third" {
  type = list(string)

  default = ["foo", "bar", "baz"]
}