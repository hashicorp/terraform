package dnsimple

import (
	"time"

	. "github.com/motain/gocheck"
)

func (s *S) Test_GetDomains(c *C) {
	testServer.Response(202, nil, domainsExample)

	domains, err := s.client.GetDomains()

	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(domains, DeepEquals, []Domain{
		Domain{
			228,
			19,
			0,
			"example.it",
			"example.it",
			"domain-token",
			"hosted",
			"",
			true,
			false,
			false,
			5,
			0,
			"",
			Jan15_3,
			Jan15_3,
		},
		Domain{
			227,
			19,
			28,
			"example.com",
			"example.com",
			"domain-token",
			"registered",
			"",
			true,
			true,
			false,
			7,
			0,
			"2015-01-16",
			Jan15_1,
			Jan16,
		},
	})
}

var domainsExample = `[
  {
    "domain": {
      "id": 228,
      "user_id": 19,
      "registrant_id": null,
      "name": "example.it",
      "unicode_name": "example.it",
      "token": "domain-token",
      "state": "hosted",
      "language": null,
      "lockable": true,
      "auto_renew": false,
      "whois_protected": false,
      "record_count": 5,
      "service_count": 0,
      "expires_on": null,
      "created_at": "2014-01-15T22:03:49Z",
      "updated_at": "2014-01-15T22:03:49Z"
    }
  },
  {
    "domain": {
      "id": 227,
      "user_id": 19,
      "registrant_id": 28,
      "name": "example.com",
      "unicode_name": "example.com",
      "token": "domain-token",
      "state": "registered",
      "language": null,
      "lockable": true,
      "auto_renew": true,
      "whois_protected": false,
      "record_count": 7,
      "service_count": 0,
      "expires_on": "2015-01-16",
      "created_at": "2014-01-15T22:01:55Z",
      "updated_at": "2014-01-16T22:56:22Z"
    }
  }
]`

var Jan15_3, _ = time.Parse("2006-01-02T15:04:05Z", "2014-01-15T22:03:49Z")
var Jan15_1, _ = time.Parse("2006-01-02T15:04:05Z", "2014-01-15T22:01:55Z")
var Jan16, _ = time.Parse("2006-01-02T15:04:05Z", "2014-01-16T22:56:22Z")
