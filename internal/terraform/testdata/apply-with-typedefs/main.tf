variable "root_string" {
  type    = typedef.custom_root_string_type
  default = true
}

variable "root_object" {
  type = module.nested_module["mod-1"].custom_module_object_type
  default = {
    a = 30
    b = false
    c = ["john", "smith"]
  }
}

module "nested_module" {
  for_each = toset(["mod-1", "mod-2"])
  source   = "./nested"
  input_object = {
    a = 10
    b = true
    c = ["smith", "john", each.key]
  }
}

output "root_string_out" {
  value = var.root_string
}

output "root_object_out" {
  value = var.root_object
}

output "nested_module_out" {
  value = module.nested_module["mod-1"].output_obj
}
