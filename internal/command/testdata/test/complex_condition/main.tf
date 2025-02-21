resource "test_resource" "foo" {
  value = "bar"
}

output "foo" {
  value = {
    bar = "notbaz"
    qux = "quux"
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

output "complex" {
  value = {
    root = var.sample
  }
}