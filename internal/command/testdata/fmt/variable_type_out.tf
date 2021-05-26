variable "a" {
  type = string
}

variable "b" {
  type = list(any)
}

variable "c" {
  type = map(any)
}

variable "d" {
  type = set(any)
}

variable "e" {
  type = string
}

variable "f" {
  type = list(string)
}

variable "g" {
  type = map(string)
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
