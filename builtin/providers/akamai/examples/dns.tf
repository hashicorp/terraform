provider "akamai" {
    edgerc = "/Users/dshafik/.edgerc"
    fastdns_section = "dns"
}

resource "akamai_fastdns_record" "test" {
  hostname = "akamaideveloper.com"
  name = "test"
  type = "Cname"
  active = true
  targets = ["developer.akamai.com."]
  ttl = 30
}
