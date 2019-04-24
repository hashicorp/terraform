resource "test_instance" "one" {
}

resource "test_instance" "many" {
  count = 2
}

data "terraform_remote_state" "one" {
}

data "terraform_remote_state" "many" {
  count = 2
}

output "managed_one" {
  value = 1
}

output "managed_many" {
  value = length(test_instance.many)
}

output "data_one" {
  value = 1
}

output "data_many" {
  value = length(data.terraform_remote_state.many)
}
