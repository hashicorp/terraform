resource "test_resource" "foo" {
  value = "bar"
}

output "foo" {
  value = {
    bar = "notbaz"
    qux = "quux"
    matches = "matches"
    xuq = "xuq"
  }
}

variable "sample" {
  type = list(object({
    bar = tuple([number])
    qux = string
  }))

  default = [ {
    bar = [1]
    qux = "quux"
  },
  {
    bar = [2]
    qux = "quux"
  }]  
}

variable "sample_sensitive" {
  sensitive = true
  type = list(object({
    bar = tuple([number])
    qux = string
  }))

  default = [ {
    bar = [1]
    qux = "quux"
  },
  {
    bar = [2]
    qux = "quux_sensitive"
  }]  
}

output "complex" {
  value = {
    root = var.sample
  }
}

output "complex_sensitive" {
  sensitive = true
  value = {
    root = var.sample_sensitive
  }
}