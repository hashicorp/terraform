terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.1"
    }
  }
}

provider "tfcoremock" {}

resource "tfcoremock_complex_resource" "complex" {
  id = "64564E36-BFCB-458B-9405-EBBF6A3CAC7A"

  number  = 987654321.0
  integer = 123456789
  float   = 123456789.0

  string = "a not very long or complex string"

  bool = true

  list = [
    {
      string = "this is my first entry in the list, and doesn't contain anything interesting"
    },
    {
      string = "this is my second entry in the list\nI am a bit more interesting\nand contain multiple lines\nbut I've been edited"
    },
    {
      string = "this is my third entry, and I actually have a nested list"

      list = [
        {
          number = 0
        },
        {
          number = 1
        },
        {
          number = 3
        },
        {
          number = 4
        }
      ]
    },
    {
      string = "this is my fourth entry, and I actually have a nested set and I edited my test"

      set = [
        {
          number = 0
        },
        {
          number = 2
        },
      ]
    }
  ]

  object = {
    string = "i am a nested object"

    number = 0

    object = {
      string = "i am a nested nested object"
      bool   = true
    }
  }

  map = {
    "key_one" = {
      string = "this is my first entry in the map, and doesn't contain anything interesting"
    },
    "key_two" = {
      string = "this is my second entry in the map\nI am a bit more interesting\nand contain multiple lines"
    },
    "key_three" = {
      string = "this is my third entry, and I actually have a nested list"

      list = [
        {
          number = 0
        },
        {
          number = 3
        },
        {
          number = 1
        },
        {
          number = 2
        }
      ]
    },
    "key_four" = {
      string = "this is my fourth entry, and I actually have a nested set"

      set = [
        {
          number = 0
        },
        {
          number = 1
        },
        {
          number = 3
        },
        {
          number = 4
        },
      ]
    }
  }

  set = [
    {
      string = "this is my first entry in the set, and doesn't contain anything interesting"
    },
    {
      string = "this is my second entry in the set\nI am a bit more interesting\nand contain multiple lines"
    },
    {
      string = "this is my third entry, and I actually have a nested list"

      list = [
        {
          number = 0
        },
        {
          number = 1
        },
        {
          number = 2
        }
      ]
    },
    {
      string = "this is my fourth entry, and I actually have a nested set"

      set = [
        {
          number = 0
        },
        {
          number = 1
        },
      ]
    }
  ]

  list_block {
    string = "{\"index\":0}"
  }

  list_block {
    string = "{\"index\":1}"

    list = [
      {
        number = 0
      },
      {
        number = 1
      },
      {
        number = 2
      }
    ]
  }

  set_block {
    string = "{\"index\":1}"

    list = [
      {
        number = 0
      },
      {
        number = 1
      },
      {
        number = 2
      }
    ]
  }

  set_block {
    string = "{\"index\":2}"

    set = [
      {
        number = 0
      },
      {
        number = 1
      },
    ]
  }

  set_block {
    string = "{\"index\":3}"
  }
}