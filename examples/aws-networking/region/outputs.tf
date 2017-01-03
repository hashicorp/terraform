output "vpc_id" {
  value = "${aws_vpc.main.id}"
}

output "primary_subnet_id" {
  value = "${module.primary_subnet.subnet_id}"
}

output "secondary_subnet_id" {
  value = "${module.secondary_subnet.subnet_id}"
}
