run "check_variables_set" {
  assert {
    condition = test_resource.terraform_tfvars_resource.value == "foo"
    error_message = "Not able to read from terraform.tfvars"
  }

  assert {
    condition = test_resource.auto_tfvars_resource.value == "bar"
    error_message = "Not able to read from *.auto.tfvars"
  }

  assert {
    condition = test_resource.auto_tfvars_overridden_resource.value == "bbar"
    error_message = "Value not overridden loading from *.auto.tfvars"
  }

  assert {
    condition = test_resource.auto_workspace_resource.value == "foobar"
    error_message = "Not able to read from *.auto.default.tfvars"
  }

  assert {
    condition = test_resource.auto_workspace_overridden_resource.value == "cban"
    error_message = "Value not overridden loading from *.auto.default.tfvars"
  }
}