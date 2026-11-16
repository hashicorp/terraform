resource "terraform_data" "a" {
}

resource "terraform_data" "b" {
  input = terraform_data.a.id
}

resource "terraform_data" "c" {
  triggers_replace = terraform_data.b
}

resource "terraform_data" "d" {
  input = [ terraform_data.b, terraform_data.c ]
}

output "d" {
  value = terraform_data.d
}
