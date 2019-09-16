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
  value = "${test_instance.one.count}"
}

output "managed_many" {
  value = "${test_instance.many.count}"
}

output "data_one" {
  value = "${data.terraform_remote_state.one.count}"
}

output "data_many" {
  value = "${data.terraform_remote_state.many.count}"
}
