locals {
  template_result = templatefile("${path.module}/template.tftpl", { name = "terraform" })
}

output "result" {
  value = local.template_result
}
