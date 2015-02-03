output "address" {
  value = "Instances: ${element(aws_instance.web.*.id, 0)}"
}
