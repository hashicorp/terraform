# Required
variable "foo" {
}

# Optional
variable "bar" {
    default = "baz"
}

# Mapping
variable "map" {
    default = {
        foo = "bar"
    }
}
