

module "child" {
  source = "./child"
}

output "child_module" {
  value = module.child.module
}

output "child_root" {
  value = module.child.root
}

output "module" {
    value = path.module
}

output "root" {
    value = path.root
}

output "cwd" {
    value = path.cwd
}
