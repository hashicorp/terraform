variable "input_object" {
  type = typedef.custom_module_object_type
}

output "output_obj" {
  value = var.input_object
}
