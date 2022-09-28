variable "a" {
  type = string
}

variable "b" {
  type = list
}

variable "c" {
  type = map
}

variable "d" {
  type = set
}

variable "e" {
  type = "string"
}

variable "f" {
  type = "list"
}

variable "g" {
  type = "map"
}

variable "h" {
  type = object({})
}

variable "i" {
  type = object({
    foo = string
  })
}

variable "j" {
  type = tuple([])
}

variable "k" {
  type = tuple([number])
}

variable "l" {
  type = list(string)
}

variable "m" {
  type = list(
    object({
      foo = bool
    })
  )
}
