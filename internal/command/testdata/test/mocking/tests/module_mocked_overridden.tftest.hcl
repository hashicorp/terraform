override_module {
  target = module.child
  outputs = {
    primary = [
      {
        id = "bbbb"
      }
    ]
    secondary = [
      {
        id = "cccc"
      }
    ]
  }
}

variables {
  instances = 3
  child_instances = 1
}

run "test" {

  override_module {
    target = module.child[1]
    outputs = {
      primary = [
        {
          id = "aaaa"
        }
      ]
      secondary = [
        {
          id = "dddd"
        }
      ]
    }
  }

  assert {
    condition = module.child[0].primary[0].id == "bbbb"
    error_message = "wrongly applied mocks"
  }

  assert {
    condition = module.child[0].secondary[0].id == "cccc"
    error_message = "did not apply mocks"
  }

  assert {
    condition = module.child[2].primary[0].id == "bbbb"
    error_message = "wrongly applied mocks"
  }

  assert {
    condition = module.child[2].secondary[0].id == "cccc"
    error_message = "did not apply mocks"
  }

  assert {
    condition = module.child[1].primary[0].id == "aaaa"
    error_message = "did not apply mocks"
  }

  assert {
    condition = module.child[1].secondary[0].id == "dddd"
    error_message = "did not apply mocks"
  }

}
