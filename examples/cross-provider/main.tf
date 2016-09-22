# Create our Heroku application. Heroku will
# automatically assign a name.
resource "heroku_app" "web" {}

# Create our DNSimple record to point to the
# heroku application.
resource "dnsimple_record" "web" {
  domain = "${var.dnsimple_domain}"

  name = "terraform"

  # heroku_hostname is a computed attribute on the heroku
  # application we can use to determine the hostname
  value = "${heroku_app.web.heroku_hostname}"

  type = "CNAME"
  ttl  = 3600
}

# The Heroku domain, which will be created and added
# to the heroku application after we have assigned the domain
# in DNSimple
resource "heroku_domain" "foobar" {
  app      = "${heroku_app.web.name}"
  hostname = "${dnsimple_record.web.hostname}"
}
