provider "akamai" {
  edgerc = "/Users/dshafik/.edgerc"
  papi_section = "default"
}

resource "akamai_property" "daveyshafikcom" {
  group = "Davey Shafik"
  hostname = ["www.daveyshafik.com"]
  cpcode = 40455

  origin {
    is_secure = true
    hostname = "origin.daveyshafik.com"
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

  cache {
    match {
      extensions = ["jpg", "jpeg", "png", "gif", "svg", "webp", "jp2", "jxr"]
    }
    max_age = "365d"
    prefreshing = true
    prefetch = true
  }

  cache {
    match {
      path = ["/admin", "/admin/*"]
    }
    cache = false
  }
}