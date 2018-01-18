provider "akamai" {
  edgerc = "/Users/Johanna/.edgerc"
  papi_section = "global"
}

resource "akamai_property" "akamaidevelopernet" {
  account_id = "act_B-C-1FRYVMN"
  contract_id = "ctr_C-1FRYVV3"
  group_id = "grp_68817"
  product_id = "prd_SPM"
  name = "test_property_terraform_jc_copy_from"
  cp_code = "409449"
  contact = ["dshafik@akamai.com"]
  hostname = ["akamaideveloper.net"]

  clone_from {
    property_id = "prp_410587"
    version = 1
    copy_hostnames = true
  }

  origin {
    is_secure = false
    hostname = "akamaideveloper.net"
    forward_hostname = "ORIGIN_HOSTNAME"
  }

  default {
    behavior {
      name = "downstreamCache"
      option {
        key = "behavior"
        value = "TUNNEL_ORIGIN"
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