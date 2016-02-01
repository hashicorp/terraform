package mailgun

import (
	"testing"

	. "github.com/motain/gocheck"
)

func TestDomain(t *testing.T) {
	TestingT(t)
}

func (s *S) Test_CreateDomain(c *C) {
	testServer.Response(202, nil, domainExample)

	opts := CreateDomain{
		Name:         "example.com",
		SpamAction:   "disabled",
		Wildcard:     true,
		SmtpPassword: "foobar",
	}

	id, err := s.client.CreateDomain(&opts)

	req := testServer.WaitRequest()

	c.Assert(req.Form["name"], DeepEquals, []string{"example.com"})
	c.Assert(req.Form["spam_action"], DeepEquals, []string{"disabled"})
	c.Assert(req.Form["wildcard"], DeepEquals, []string{"true"})
	c.Assert(req.Form["smtp_password"], DeepEquals, []string{"foobar"})
	c.Assert(err, IsNil)
	c.Assert(id, Equals, "domain.com")
}

func (s *S) Test_RetrieveDomain(c *C) {
	testServer.Response(200, nil, domainExample)

	resp, err := s.client.RetrieveDomain("example.com")

	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(resp.Domain.Name, Equals, "domain.com")
	c.Assert(resp.Domain.SmtpPassword, Equals, "4rtqo4p6rrx9")
	c.Assert(resp.Domain.StringWildcard(), Equals, "false")
	c.Assert(resp.ReceivingRecords[0].Priority, Equals, "10")
	c.Assert(resp.SendingRecords[0].Value, Equals, "v=spf1 include:mailgun.org ~all")
}

func (s *S) Test_DestroyDomain(c *C) {
	testServer.Response(204, nil, "")

	err := s.client.DestroyDomain("example.com")

	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
}

var domainErrorExample = `{
  "message": "Domain name format is bad"
}`

var domainExample = `
{
  "domain": {
    "created_at": "Wed, 10 Jul 2013 19:26:52 GMT",
    "smtp_login": "postmaster@domain.com",
    "name": "domain.com",
    "smtp_password": "4rtqo4p6rrx9",
    "wildcard": false,
    "spam_action": "tag"
  },
  "receiving_dns_records": [
    {
      "priority": "10",
      "record_type": "MX",
      "valid": "valid",
      "value": "mxa.mailgun.org"
    },
    {
      "priority": "10",
      "record_type": "MX",
      "valid": "valid",
      "value": "mxb.mailgun.org"
    }
  ],
  "sending_dns_records": [
    {
      "record_type": "TXT",
      "valid": "valid",
      "name": "domain.com",
      "value": "v=spf1 include:mailgun.org ~all"
    },
    {
      "record_type": "TXT",
      "valid": "valid",
      "name": "domain.com",
      "value": "k=rsa; p=MIGfMA0GCSqGSIb3DQEBAQUA...."
    },
    {
      "record_type": "CNAME",
      "valid": "valid",
      "name": "email.domain.com",
      "value": "mailgun.org"
    }
  ]
}
`
