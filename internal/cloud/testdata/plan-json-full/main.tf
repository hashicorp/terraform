provider "tfcoremock" {}

# In order to generate the JSON logs contained in plan.log
# First ONLY apply tfcoremock_simple_resource.example (set the bool attribute
# to true). Make sure the complex_resource is commented out.
# Once applied, change the bool attribute to false and uncomment the complex
# resource.

resource "tfcoremock_simple_resource" "example" {
  id      = "my-simple-resource"
  bool    = false
  number  = 0
  string  = "Hello, world!"
  float   = 0
  integer = 0
}

resource "tfcoremock_complex_resource" "example" {
  id = "my-complex-resource"

  bool    = true
  number  = 0
  string  = "Hello, world!"
  float   = 0
  integer = 0

  list = [
    {
      string = "list.one"
    },
    {
      string = "list.two"
    }
  ]

  set = [
    {
      string = "set.one"
    },
    {
      string = "set.two"
    }
  ]

  map = {
    "one" : {
      string = "map.one"
    },
    "two" : {
      string = "map.two"
    }
  }

  object = {

    string = "nested object"

    object = {
      string = "nested nested object"
    }
  }

  list_block {
    string = "list_block.one"
  }

  list_block {
    string = "list_block.two"
  }

  list_block {
    string = "list_block.three"
  }

  set_block {
    string = "set_block.one"
  }

  set_block {
    string = "set_block.two"
  }
}
