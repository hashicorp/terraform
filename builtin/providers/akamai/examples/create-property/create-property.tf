provider "akamai" {
  edgerc = "/Users/dshafik/.edgerc"
  papi_section = "global"
}

resource "akamai_property" "daveyshafikcom" {
  group = "Davey Shafik"
  contract = "ctr_C-1FRYVV3"
  contact = ["dshafik@akamai.com"]

  hostname = ["test98.randomnoise.us"]

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

    rule {
      name = "Redirect to HTTPS"
      comment = "Redirect to the same URL on HTTPS protocol, issuing a 301 response code (Moved Permanently). You may change the response code to 302 if needed."

      criteria {
        name = "requestProtocol"
        option {
          name = "value"
          value = "HTTP"
        }
      }

      criteria {
        name = "hostname"
        option {
          name = "matchOperator"
          value = "IS_NOT_ONE_OF"
        }

        option {
          name = "values"
          values = ["staging.daveyshafik.com"]
        }
      }

      behavior {
        name = "redirect"

        option {
          name = "mobileDefaultChoice"
          value = "DEFAULT"
        }
        option {
          name = "destinationProtocol"
          value = "HTTPS"
        }
        option {
          name = "destinationHostname"
          value = "OTHER"
        }
        option {
          name = "destinationHostnameOther"
          value = "www.daveyshafik.com"
        }
        option {
          name = "destinationPath"
          value = "SAME_AS_REQUEST"
        }
        option {
          name = "queryString"
          value = "APPEND"
        }
        option {
          name = "responseCode"
          value = "301"
          type = "int"
        }
      }
    }

    rule {
      name = "Performance"
      comment = "Improves the performance of delivering objects to end users. Behaviors in this rule are applied to all requests as appropriate."

      behavior {
        name = "enhancedAkamaiProtocol"

        option {
          name = "display"
          value = ""
          type = "string"
        }
      }

      behavior {
        name = "prefetch"

        option {
          name = "enabled"
          flag = true
        }
      }

      rule {
        name = "Enable HTTP2"
        comment = ""

        criteria {
          name = "hostname"

          option {
            name = "matchOperator"
            value = "IS_ONE_OF"
          }

          option {
            name = "values"
            values = ["www.daveyshafik.com", "daveyshafik.com"]
          }
        }

        behavior {
          name = "spdy"

          option {
            name = "enabled"
            value = ""
            type = "string"
          }
        }

        behavior {
          name = "http2"

          option {
            name = "enabled"
            value = ""
            type = "string"
          }
        }

        behavior {
          name = "allowTransferEncoding"

          option {
            name = "enabled"
            flag = true
          }
        }

        behavior {
          name = "adaptiveAcceleration"

          option {
            name = "titleHttp2ServerPush"
            value = ""
            type = "string"
          }

          option {
            name = "enablePush"
            flag = true
          }

          option {
            name = "titlePreconnect"
            value = ""
            type = "string"
          }

          option {
            name = "enablePreconnect"
            flag = true
          }

          option {
            name = "useDefaultPush"
            flag = true
          }

          option {
            name = "useDefaultPreconnect"
            flag = true
          }
        }
      }

      rule {
        name = "Images"
        comment = "Improves load time by applying Adaptive Image Compression (AIC) to all JPEG images. The poorer the connection quality, the more AIC compresses the image files."

        criteria {
          name = "fileExtension"

          option {
            name = "matchOperator"
            value = "IS_ONE_OF"
          }

          option {
            name = "values"
            values = ["jpg", "jpeg", "jpe", "jif", "jfif", "jfi", "png", "gif", "webp", "jxr", "jp2"]
          }

          option {
            name = "matchCaseSensitive"
            flag = false
            type = "bool"
          }
        }

        behavior {
          name = "downstreamCache"

          option {
            name = "behavior"
            value = "ALLOW"
          }

          option {
            name = "allowBehavior"
            value = "FROM_VALUE"
          }

          option {
            name = "ttl"
            value = "24h"
          }

          option {
            name = "sendHeaders"
            value = "CACHE_CONTROL_AND_EXPIRES"
          }

          option {
            name = "sendPrivate"
            flag = false
            type = "bool"
          }
        }

        behavior {
          name = "caching"

          option {
            name = "behavior"
            value = "MAX_AGE"
          }

          option {
            name = "mustRevalidate"
            flag = false
            type = "bool"
          }

          option {
            name = "ttl"
            value = "7d"
          }
        }
      }

      rule {
        name = "Compressible Objects"
        comment = "Compresses content to improve performance of clients with slow connections. Applies Last Mile Acceleration to requests when the returned object supports gzip compression."

        criteria {
          name = "contentType"

          option {
            name = "matchOperator"
            value = "IS_ONE_OF"
          }

          option {
            name = "values"
            values = [
              "text/*",
              "application/javascript",
              "application/x-javascript",
              "application/x-javascript*",
              "application/json",
              "application/x-json",
              "application/*+json",
              "application/*+xml",
              "application/text",
              "application/vnd.microsoft.icon",
              "application/vnd-ms-fontobject",
              "application/x-font-ttf",
              "application/x-font-opentype",
              "application/x-font-truetype",
              "application/xmlfont/eot",
              "application/xml",
              "font/opentype",
              "font/otf",
              "font/eot",
              "image/svg+xml",
              "image/vnd.microsoft.icon"
            ]
          }

          option {
            name = "matchWildcard"
            flag = true
          }

          option {
            name = "matchCaseSensitive"
            flag = false
            type = "bool"
          }
        }

        behavior {
          name = "gzipResponse"

          option {
            name = "behavior"
            value = "ALWAYS"
          }
        }
      }

      rule {
        name = "Offload"
        comment = "Controls caching, which offloads traffic away from the origin. Most objects types are not cached. However, the child rules override this behavior for certain subsets of requests."

        behavior {
          name = "removeVary"

          option {
            name = "enabled"
            flag = true
          }
        }

        behavior {
          name = "caching"

          option {
            name = "behavior"
            value = "MAX_AGE"
          }

          option {
            name = "mustRevalidate"
            flag = false
            type = "bool"
          }

          option {
            name = "ttl"
            value = "1d"
          }
        }

        behavior {
          name = "cacheError"

          option {
            name = "enabled"
            flag = true
          }

          option {
            name = "ttl"
            value = "10s"
          }

          option {
            name = "preserveStale"
            flag = true
          }
        }

        behavior {
          name = "downstreamCache"

          option {
            name = "behavior"
            value = "ALLOW"
          }

          option {
            name = "allowBehavior"
            value = "GREATER"
          }

          option {
            name = "sendHeaders"
            value = "CACHE_CONTROL_AND_EXPIRES"
          }

          option {
            name = "sendPrivate"
            flag = false
            type = "bool"
          }
        }
      }

      behavior {
        name = "tieredDistribution"

        option {
          name = "enabled"
          flag = true
        }
      }

      rule {
        name = "CSS and JavaScript"
        comment = "Overrides the default caching behavior for CSS and JavaScript objects that are cached on the edge server. Because these object types are dynamic, the TTL is brief."

        criteria {
          name = "fileExtension"

          option {
            name = "matchOperator"
            value = "IS_ONE_OF"
          }

          option {
            name = "values"
            values = ["css", "js", "ico"]
          }

          option {
            name = "matchCaseSensitive"
            flag = false
            type = "bool"
          }
        }

        behavior {
          name = "caching"

          option {
            name = "behavior"
            value = "MAX_AGE"
          }

          option {
            name = "mustRevalidate"
            flag = false
            type = "bool"
          }

          option {
            name = "ttl"
            value = "365d"
          }
        }

        behavior {
          name = "prefreshCache"

          option {
            name = "enabled"
            flag = true
          }

          option {
            name = "prefreshval"
            value = "90"
          }
        }

        behavior {
          name = "prefetchable"

          option {
            name = "enabled"
            flag = true
          }
        }

        behavior {
          name = "cacheKeyQueryParams"

          option {
            name = "behavior"
            value = "INCLUDE_ALL_ALPHABETIZE_ORDER"
          }
        }
      }
      rule {
        name = "Static Objects"
        comment = "Overrides the default caching behavior for images, music, and similar objects that are cached on the edge server. Because these object types are static, the TTL is long."

        criteria_match = "any"

        criteria {
          name = "fileExtension"

          option {
            name = "matchOperator"
            value = "IS_ONE_OF"
          }

          option {
            name = "values"
            values = [
              "aif",
              "aiff",
              "au",
              "avi",
              "bin",
              "bmp",
              "cab",
              "carb",
              "cct",
              "cdf",
              "class",
              "doc",
              "dcr",
              "dtd",
              "exe",
              "flv",
              "gcf",
              "gff",
              "gif",
              "grv",
              "hdml",
              "hqx",
              "ini",
              "jpeg",
              "jpg",
              "mov",
              "mp3",
              "nc",
              "pct",
              "pdf",
              "png",
              "ppc",
              "pws",
              "swa",
              "swf",
              "txt",
              "vbs",
              "w32",
              "wav",
              "wbmp",
              "wml",
              "wmlc",
              "wmls",
              "wmlsc",
              "xsd",
              "zip",
              "pict",
              "tif",
              "tiff",
              "mid",
              "midi",
              "ttf",
              "eot",
              "woff",
              "woff2",
              "otf",
              "svg",
              "svgz",
              "webp",
              "jxr",
              "jar",
              "jp2"
            ]
          }

          option {
            name = "matchCaseSensitive"
            flag = false
            type = "bool"
          }
        }

        behavior {
          name = "caching"

          option {
            name = "behavior"
            value = "MAX_AGE"
          }

          option {
            name = "mustRevalidate"
            flag = false
            type = "bool"
          }

          option {
            name = "ttl"
            value = "365d"
          }
        }

        behavior {
          name = "prefreshCache"

          option {
            name = "enabled"
            flag = true
          }

          option {
            name = "prefreshval"
            value = "90"
          }
        }

        behavior {
          name = "prefetchable"

          option {
            name = "enabled"
            flag = true
          }
        }

        behavior {
          name = "downstreamCache"

          option {
            name = "behavior"
            value = "ALLOW"
          }

          option {
            name = "allowBehavior"
            value = "GREATER"
          }

          option {
            name = "sendHeaders"
            value = "CACHE_CONTROL_AND_EXPIRES"
          }

          option {
            name = "sendPrivate"
            flag = false
            type = "bool"
          }
        }
      }
      rule {
        name = "Uncacheable Responses"
        comment = "Overrides the default downstream caching behavior for uncacheable object types. Instructs the edge server to pass Cache-Control and/or Expire headers from the origin to the client."

        criteria {
          name = "cacheability"

          option {
            name = "value"
            value = "CACHEABLE"
          }

          option {
            name = "matchOperator"
            value = "IS_NOT"
          }
        }
      }
    }
  }
}