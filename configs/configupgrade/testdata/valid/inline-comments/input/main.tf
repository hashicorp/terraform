variable "list" {
  type = "list"

  default = [
    "foo", # I am a comment
    "bar", # I am also a comment
    "baz",
  ]
}

variable "list2" {
  type = "list"

  default = [
    "foo",
    "bar",
    "baz",
  ]
}

variable "list_the_third" {
  type = "list"

  default = ["foo", "bar", "baz"]
}
