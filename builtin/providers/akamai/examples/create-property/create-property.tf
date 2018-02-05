provider "akamai" {
  edgerc = "~/.edgerc"
  papi_section = "global"
}

resource "akamai_property" "akamai_developer" {
  activate = false
  name = "akamaideveloper.com"

  contact = ["dshafik@akamai.com"]

  account_id = "act_B-F-1ACME"
  product_id = "prd_SPM"
  cp_code = "409449"
  contract_id = "ctr_C-1FRYVV3"
  group_id = "grp_68817"

  hostname = ["akamaideveloper.net"]

  network = "STAGING"

  rule_format = "v2016-11-15"

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
      name = "Performance"
      rule {
          name = "JPEG Images"
          behavior {
              name = "adaptiveImageCompression"
              option {
                   key = "tier1MobileCompressionMethod"
                   value = "COMPRESS"
              }
              option {
                 key = "tier1MobileCompressionValue"
                 value = "80"
              }
              option {
                   key = "tier2MobileCompressionMethod"
                   value = "COMPRESS"
              }
          }
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
