resource "template_instance" "example" {
  compute_value = "template text"
  compute = "value"
}

output "base_config" {
  value = {
    base_template = template_instance.example.value
  }
}
