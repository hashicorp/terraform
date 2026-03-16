variable "environment" {
  type = string
}

variable "password" {
  type = string
}

variable "vpc_id" {
  type = string
}

variable "db_subnet_group_name" {
  type = string
}

variable "destroy_wait_seconds" {
  type    = number
  default = 0
}

# Simulates the terraform-aws-modules/rds/aws module
# This represents the thin wrapper around the AWS RDS module
resource "test_resource" "db" {
  value = "${var.environment}-${var.db_subnet_group_name}"

  # Add some delay to simulate real RDS creation/deletion time
  destroy_wait_seconds = var.destroy_wait_seconds
}

output "db_instance_id" {
  value = test_resource.db.value
}

output "db_endpoint" {
  value = "${test_resource.db.value}.example.com"
}
