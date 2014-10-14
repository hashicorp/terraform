output "address" {
  value = "Instances: ${aws_instance.web.*.id}"
}
