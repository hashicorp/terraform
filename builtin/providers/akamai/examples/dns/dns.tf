provider "akamai" {
    edgerc = "/Users/Johanna/.edgerc"
    fastdns_section = "dns"
}

resource "akamai_fastdns_record" "test" {
  hostname = "akamaideveloper.net"
  name = "testing"
  type = "Cname"
  active = true
  targets = ["akamaideveloper.net."]
  ttl = 30
}
