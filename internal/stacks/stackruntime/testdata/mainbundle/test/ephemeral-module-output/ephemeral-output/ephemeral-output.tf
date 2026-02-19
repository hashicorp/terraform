
ephemeral "testing_resource" "resource" {}

output "value" {
  value = ephemeral.testing_resource.resource.value
  ephemeral = true
}
