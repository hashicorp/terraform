provider "nsone" {
    apikey = "fF4uiAIL9wntC7Bdar0a"
}

resource "nsone_zone" "foo_com" {
    zone = "foo.com"
    hostmaster = "hostmaster@foo.com"
}

#resource "nsone_record" "www_foo_com" {
#    zone = "foo.com"
#    domain = "www.foo.com"
#    type = "A"
#}

