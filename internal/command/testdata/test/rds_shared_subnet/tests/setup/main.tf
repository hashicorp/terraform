resource "test_resource" "name" {
  value = "tftest-delete-me-normal-haddock"
}

output "name" {
  value = test_resource.name.value
}

resource "test_resource" "vpc" {
  value = "vpc-0ae7a165e6927405b"
}

output "vpc_id" {
  value = test_resource.vpc.value
}

resource "test_resource" "subnet_b" {
  value = "subnet-eu-west-1b-${test_resource.vpc.value}"
}

resource "test_resource" "subnet_c" {
  value = "subnet-eu-west-1c-${test_resource.vpc.value}"
}

resource "test_resource" "subnet_group" {
  value = "${test_resource.name.value}-${test_resource.subnet_b.value}-${test_resource.subnet_c.value}"
  # Add delay to simulate real AWS resource creation/deletion time
  destroy_wait_seconds = 2
}

output "subnet_group" {
  value = {
    name = test_resource.subnet_group.value
    id   = test_resource.subnet_group.value
  }
}

resource "test_resource" "password" {
  value = "supersecretpassword123"
}

output "password" {
  value     = test_resource.password.value
  sensitive = true
}
