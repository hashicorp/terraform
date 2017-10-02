provider "akamai" {
  edgerc = "/Users/Johanna/.edgerc"
  papi_section = "global"
}

resource "akamai_property" "akamaidevelopernet" {
  account_id = "act_B-C-1FRYVMN"
  contract_id = "ctr_C-1FRYVV3"
  group_id = "grp_68817"
  product_id = "prd_Adaptive_Media_Delivery"
  name = "test_property"

  origin {
    is_secure = false
    hostname = "akamaideveloper.net"
    forward_hostname = "ORIGIN_HOSTNAME"
  }

  rule {
    name = "default"
    comment = "The default rule applies to all requests"

    behavior {
      name = "allowPost"
      option {
        name = "enabled"
        flag = true
      }
      option {
        name = "allowWithoutContentLength"
        flag = true
      }
    }

    behavior {
      name = "realUserMonitoring"
      option {
        name = "enabled"
        flag = true
      }
    }
  }

  rule {
    name = "Fixup Path"
    comment = "Prefix incoming path with /api, unless it's already there"

    criteria {
      name = "path"

      option {
        name = "matchOperator"
        value = "DOES_NOT_MATCH_ONE_OF"
      }

      option {
        name = "values"
        values = ["/api/", "/api/*/"]
      }

      option {
        name = "matchCaseSensitive"
        flag = false
      }
    }

    behavior {
      name = "rewriteUrl"

      option {
        name = "behavior"
        value = "PREPEND"
      }

      option {
        name = "targetPathPrepend"
        value = "/api/"
      }

      option {
        name = "keepQueryString"
        flag = true
      }
    }
  }
}