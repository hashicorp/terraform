provider "akamai" {
    edgerc = "/Users/Johanna/.edgerc"
    fastdns_section = "dns"
}

resource "akamai_fastdns_record" "test_zone" {
  hostname = "akamaideveloper.net"
  soa {
    ttl = 900
    originserver = "akamaideveloper.net."
    contact = "hostmaster.akamaideveloper.net."
    refresh = 900
    retry = 300
    expire = 604800
    minimum = 180
  }
  a {
    name = "web"
    ttl = 900
    active = true
    target = "1.2.3.4"
  }
  a {
    name = "www"
    ttl = 600
    active = true
    target = "5.6.7.8"
  }
}
