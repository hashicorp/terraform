resource "null_resource" "foo" {}

output "simple" {
  value = ["some", "list"]
}

output "secret" {
  value = "my-secret"
  sensitive = true
}

output "complex" {
  value = {
    keyA = {
      someList = [1, 2, 3]
    }
    keyB = {
      someBool = true
      someStr = "hello"
    }
  }
}
