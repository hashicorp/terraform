variable "from_terraform_tfvars" {
  type = string
}

variable "from_auto_tfvars" {
  type = string
}

variable "from_auto_tfvars_overridden" {
  type = string
}

variable "from_auto_workspace_tfvars" {
  type = string
}

variable "from_auto_workspace_overridden" {
  type = string
}

resource "test_resource" "terraform_tfvars_resource" {
  value = var.from_terraform_tfvars
}

resource "test_resource" "auto_tfvars_resource" {
  value = var.from_auto_tfvars
}

resource "test_resource" "auto_tfvars_overridden_resource" {
  value = var.from_auto_tfvars_overridden
}

resource "test_resource" "auto_workspace_resource" {
  value = var.from_auto_workspace_tfvars
}

resource "test_resource" "auto_workspace_overridden_resource" {
  value = var.from_auto_workspace_overridden
}