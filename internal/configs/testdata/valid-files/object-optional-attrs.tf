variable "a" {
  type = object({
    foo = optional(string)
    bar = optional(bool, true)
  })
}

variable "b" {
  type = list(
    object({
      foo = optional(string)
    })
  )
}

variable "c" {
  type = set(
    object({
      foo = optional(string)
    })
  )
}

variable "d" {
  type = map(
    object({
      foo = optional(string)
    })
  )
}

variable "e" {
  type = object({
    foo = string
    bar = optional(bool, true)
  })
  default = null
}
