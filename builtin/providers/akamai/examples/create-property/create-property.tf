provider "akamai" {
  edgerc = "/Users/dshafik/.edgerc"
  papi_section = "global"
}

resource "akamai_property" "daveyshafikcom" {
  group = "Davey Shafik"
  contract = "ctr_C-1FRYVV3"
  contact = ["dshafik@akamai.com"]

  hostname = ["test95.randomnoise.us"]

  cpcode = "409449"

  origin {
    is_secure = false
    hostname = "www.randomnoise.us"
  }

  compress {
    extensions    = ["css", "js"]
    content_types = ["text/html", "text/css"]
  }

  cache {
    match {
      extensions = ["css", "js"]
    }
    max_age = "30d"
    prefreshing = true
    prefetch = true
    query_params = true
    query_params_sort = true
  }

  rule {
    name = "Testing"

    criteria {
      name = "hostname"
      option {
        name = "matchOperator"
        value = "IS_ONE_OF"
      }
      option {
        name = "values"
        values = ["custom.example.com"]
      }
    }

    behavior {
      name = "foo"
      option {
        name = "bar"
        value = "test"
      }
    }

    comment = "A random comment"

    rule {
      name = "Testing"

      criteria {
        name = "hostname"
        option {
          name = "matchOperator"
          value = "IS_ONE_OF"
        }
        option {
          name = "values"
          values = ["custom.example.com"]
        }
      }

      behavior {
        name = "foo"
        option {
          name = "bar"
          value = "test"
        }
      }

      comment = "A random comment"

      rule {
        name = "Testing"

        criteria {
          name = "hostname"
          option {
            name = "matchOperator"
            value = "IS_ONE_OF"
          }
          option {
            name = "values"
            values = ["custom.example.com"]
          }
        }

        behavior {
          name = "foo"
          option {
            name = "bar"
            value = "test"
          }
        }

        comment = "A random comment"

        rule {
          name = "Testing"

          criteria {
            name = "hostname"
            option {
              name = "matchOperator"
              value = "IS_ONE_OF"
            }
            option {
              name = "values"
              values = ["custom.example.com"]
            }
          }

          behavior {
            name = "foo"
            option {
              name = "bar"
              value = "test"
            }
          }

          comment = "A random comment"

          rule {
            name = "Testing"

            criteria {
              name = "hostname"
              option {
                name = "matchOperator"
                value = "IS_ONE_OF"
              }
              option {
                name = "values"
                values = ["custom.example.com"]
              }
            }

            behavior {
              name = "foo"
              option {
                name = "bar"
                value = "test"
              }
            }

            comment = "A random comment"
          }
        }
      }
    }
  }
}