provider "akamai" {
    edgerc = "/Users/Johanna/.edgerc"
    fastdns_section = "dns"
}

resource "akamai_fastdns_record" "test_soa_record" {
  hostname = "akamaideveloper.net"
  soa {
    type = "soa"
    ttl = 900
    originserver = "akamaideveloper.net."
    contact = "hostmaster.akamaideveloper.net"
    refresh = 900
    retry = 300
    expire = 604800
    minimum = 180
  }
  a {
    type = "a"
    name = "test"
    ttl = 900
    active = true
    target = "akamaideveloper.net"
  }
  a {
    type = "a"
    name = "test2"
    ttl = 600
    active = true
    target = "aloper.net"
  }
}
