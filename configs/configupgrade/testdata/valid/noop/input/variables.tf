
/* This multi-line comment
   should survive */

# This comment should survive
variable "foo" {
  default = 1 // This comment should also survive
}

// These adjacent comments should remain adjacent
// to one another.

variable "bar" {
  /* This comment should survive too */
  description = "bar the baz"
}

// This comment that isn't attached to anything should survive.
