
variable "contents" {
  type = string
}

resource "random_integer" "random" {
  min = 1000000
  max = 9999999
  seed = "F78CB410-BA01-44E1-82E1-37D61F7CB158"
}

locals {
  contents = jsonencode({
    values = {
      id = {
        string = random_integer.random.id
      }
      string = {
        string = var.contents
      }
    }
  })
}

resource "local_file" "data_file" {
  filename = "terraform.data/${random_integer.random.id}.json"
  content = local.contents
}

output "id" {
  value = random_integer.random.id
}
