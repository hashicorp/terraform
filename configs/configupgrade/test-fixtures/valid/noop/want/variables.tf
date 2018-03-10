
# This comment should survive
variable "foo" {
  default = 1 // This comment should also survive
}

variable "bar" {
  /* This comment should survive too */
  description = "bar the baz"
}

// This comment that isn't attached to anything should survive.
