provider "akamai" {
  edgerc = "/Users/Johanna/.edgerc"
  papi_section = "global"
}

resource "akamai_property" "akamaidevelopernet" {
  account_id = "act_B-C-1FRYVMN"
  contract_id = "ctr_C-1FRYVV3"
  group_id = "grp_68817"
  product_id = "prd_SPM"
  name = "test_property_terraform_jc"
  cp_code = "409449"
  contact = ["dshafik@akamai.com"]
  hostname = ["akamaideveloper.net"]

  origin {
    is_secure = false
    hostname = "akamaideveloper.net"
    forward_hostname = "ORIGIN_HOSTNAME"
  }

  default {
    behavior {
      name = "downstreamCache"
      option {
        name = "behavior"
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
        name = "matchOperator"
        value = "IS_NOT"
      }
      option {
        name = "value"
        value = "CACHEABLE"
      }
    }
    behavior {
      name = "downstreamCache"
      option {
        name = "behavior"
        value = "TUNNEL_ORIGIN"
      }
    }
  }
}