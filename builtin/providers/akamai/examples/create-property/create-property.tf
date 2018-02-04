provider "akamai" {
  edgerc = "/Users/Johanna/.edgerc"
  papi_section = "global"
}

resource "akamai_property" "akamai_developer" {
  name = "akamaideveloper.com"

  contact = ["dshafik@akamai.com"]

  account_id = "act_B-F-1ACME"
  product_id = "prd_SPM"
  cp_code = "123456"

  hostname = ["akamaideveloper.net"]

  origin {
    is_secure = false
    hostname = "akamaideveloper.net"
    forward_hostname = "ORIGIN_HOSTNAME"
  }

  rules {
    behavior {
      name = "downstreamCache"
      option {
        key = "behavior"
        value = "TUNNEL_ORIGIN"
      }
    }

    rule {
      name = "Uncacheable Responses"
      comment = "Cache me outside"
      criteria {
        name = "cacheability"
        option {
          key = "matchOperator"
          value = "IS_NOT"
        }
        option {
          key = "value"
          value = "CACHEABLE"
        }
      }
      behavior {
        name = "downstreamCache"
        option {
          key = "behavior"
          value = "TUNNEL_ORIGIN"
        }
      }
      rule {
        name = "Uncacheable Responses"
        comment = "Child rule"
        criteria {
          name = "cacheability"
          option {
            key = "matchOperator"
            value = "IS_NOT"
          }
          option {
            key = "value"
            value = "CACHEABLE"
          }
        }
        behavior {
          name = "downstreamCache"
          option {
            key = "behavior"
            value = "TUNNEL_ORIGIN"
          }
        }
      }
    }
  }
}
