package cloudflare

import (
	"testing"

	. "github.com/motain/gocheck"
)

func TestRecord(t *testing.T) {
	TestingT(t)
}

func (s *S) Test_CreateRecord(c *C) {
	testServer.Response(200, nil, recordExample)

	opts := CreateRecord{
		Type:    "A",
		Name:    "foobar",
		Content: "10.0.0.1",
	}

	record, err := s.client.CreateRecord("example.com", &opts)

	req := testServer.WaitRequest()

	c.Assert(req.Form["type"], DeepEquals, []string{"A"})
	c.Assert(req.Form["name"], DeepEquals, []string{"foobar"})
	c.Assert(req.Form["content"], DeepEquals, []string{"10.0.0.1"})
	c.Assert(req.Form["ttl"], DeepEquals, []string{"1"})
	c.Assert(err, IsNil)
	c.Assert(record.Id, Equals, "23734516")
}

func (s *S) Test_CreateRecordWithTTL(c *C) {
	testServer.Response(200, nil, recordExample)

	opts := CreateRecord{
		Type:    "A",
		Name:    "foobar",
		Content: "10.0.0.1",
		Ttl:     "600",
	}

	record, err := s.client.CreateRecord("example.com", &opts)

	req := testServer.WaitRequest()

	c.Assert(req.Form["type"], DeepEquals, []string{"A"})
	c.Assert(req.Form["name"], DeepEquals, []string{"foobar"})
	c.Assert(req.Form["content"], DeepEquals, []string{"10.0.0.1"})
	c.Assert(req.Form["ttl"], DeepEquals, []string{"600"})
	c.Assert(err, IsNil)
	c.Assert(record.Id, Equals, "23734516")
}

func (s *S) Test_RetrieveRecord(c *C) {
	testServer.Response(200, nil, recordsExample)

	record, err := s.client.RetrieveRecord("example.com", "16606009")

	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(record.Name, Equals, "direct")
	c.Assert(record.Id, Equals, "16606009")
	c.Assert(record.Priority, Equals, "")
}

func (s *S) Test_RetrieveRecord_Bad(c *C) {
	testServer.Response(200, nil, recordsErrorExample)

	record, err := s.client.RetrieveRecord("example.com", "16606009")

	_ = testServer.WaitRequest()

	c.Assert(err.Error(), Equals, "API Error: Invalid zone.")
	c.Assert(record, IsNil)
}

func (s *S) Test_DestroyRecord(c *C) {
	testServer.Response(200, nil, recordDeleteExample)

	err := s.client.DestroyRecord("example.com", "25")

	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
}

func (s *S) Test_UpdateRecord_Bad(c *C) {
	testServer.Response(200, nil, recordErrorExample)

	opts := UpdateRecord{
		Name: "foobar",
		Type: "CNAME",
	}

	err := s.client.UpdateRecord("example.com", "16606009", &opts)

	req := testServer.WaitRequest()

	c.Assert(req.Form["type"], DeepEquals, []string{"CNAME"})
	c.Assert(req.Form["name"], DeepEquals, []string{"foobar"})
	c.Assert(err.Error(), Equals, "API Error: Invalid record id.")
}

func (s *S) Test_UpdateRecord(c *C) {
	testServer.Response(200, nil, recordExample)

	opts := UpdateRecord{
		Name: "foobar",
		Type: "CNAME",
	}

	err := s.client.UpdateRecord("example.com", "25", &opts)

	req := testServer.WaitRequest()

	c.Assert(req.Form["type"], DeepEquals, []string{"CNAME"})
	c.Assert(req.Form["name"], DeepEquals, []string{"foobar"})
	c.Assert(err, IsNil)
}

var recordErrorExample = `{
  "request": {
    "act": "rec_edit"
  },
  "result": "error",
  "msg": "Invalid record id."
}`

var recordsErrorExample = `{
  "request": {
    "act": "rec_load_all"
  },
  "result": "error",
  "msg": "Invalid zone."
}`

var recordDeleteExample = `{
  "request": {
    "act": "rec_delete",
    "a": "rec_delete",
    "tkn": "1296c62233d48a6cf0585b0c1dddc3512e4b2",
    "id": "23735515",
    "email": "sample@example.com",
    "z": "example.com"
  },
  "result": "success",
  "msg": null
}`

var recordsExample = `{
  "request": {
    "act": "rec_load_all",
    "a": "rec_load_all",
    "email": "sample@example.com",
    "tkn": "8afbe6dea02407989af4dd4c97bb6e25",
    "z": "example.com"
  },
  "response": {
    "recs": {
      "has_more": false,
      "count": 7,
      "objs": [
        {
          "rec_id": "16606009",
          "rec_tag": "7f8e77bac02ba65d34e20c4b994a202c",
          "zone_name": "example.com",
          "name": "direct.example.com",
          "display_name": "direct",
          "type": "A",
          "prio": null,
          "content": "[server IP]",
          "display_content": "[server IP]",
          "ttl": "1",
          "ttl_ceil": 86400,
          "ssl_id": null,
          "ssl_status": null,
          "ssl_expires_on": null,
          "auto_ttl": 1,
          "service_mode": "0",
          "props": {
            "proxiable": 1,
            "cloud_on": 0,
            "cf_open": 1,
            "ssl": 0,
            "expired_ssl": 0,
            "expiring_ssl": 0,
            "pending_ssl": 0
          }
        },
        {
          "rec_id": "16606003",
          "rec_tag": "d5315634e9f5660d3670e62fa176e5de",
          "zone_name": "example.com",
          "name": "home.example.com",
          "display_name": "home",
          "type": "A",
          "prio": null,
          "content": "[server IP]",
          "display_content": "[server IP]",
          "ttl": "1",
          "ttl_ceil": 86400,
          "ssl_id": null,
          "ssl_status": null,
          "ssl_expires_on": null,
          "auto_ttl": 1,
          "service_mode": "0",
          "props": {
            "proxiable": 1,
            "cloud_on": 0,
            "cf_open": 1,
            "ssl": 0,
            "expired_ssl": 0,
            "expiring_ssl": 0,
            "pending_ssl": 0
          }
        },
        {
          "rec_id": "16606000",
          "rec_tag": "23b26c051884e94e18711742942760b1",
          "zone_name": "example.com",
          "name": "example.com",
          "display_name": "example.com",
          "type": "A",
          "prio": null,
          "content": "[server IP]",
          "display_content": "[server IP]",
          "ttl": "1",
          "ttl_ceil": 86400,
          "ssl_id": null,
          "ssl_status": null,
          "ssl_expires_on": null,
          "auto_ttl": 1,
          "service_mode": "1",
          "props": {
            "proxiable": 1,
            "cloud_on": 1,
            "cf_open": 1,
            "ssl": 0,
            "expired_ssl": 0,
            "expiring_ssl": 0,
            "pending_ssl": 0
          }
        },
        {
          "rec_id": "18136402",
          "rec_tag": "3bcef45cdf5b7638b13cfb89f1b6e716",
          "zone_name": "example.com",
          "name": "test.example.com",
          "display_name": "test",
          "type": "A",
          "prio": null,
          "content": "[server IP]",
          "display_content": "[server IP]",
          "ttl": "1",
          "ttl_ceil": 86400,
          "ssl_id": null,
          "ssl_status": null,
          "ssl_expires_on": null,
          "auto_ttl": 1,
          "service_mode": "0",
          "props": {
            "proxiable": 1,
            "cloud_on": 0,
            "cf_open": 1,
            "ssl": 0,
            "expired_ssl": 0,
            "expiring_ssl": 0,
            "pending_ssl": 0
          }
        },
        {
          "rec_id": "16606018",
          "rec_tag": "c0b453b2d94213a7930d342114cbda86",
          "zone_name": "example.com",
          "name": "www.example.com",
          "display_name": "www",
          "type": "CNAME",
          "prio": null,
          "content": "example.com",
          "display_content": "example.com",
          "ttl": "1",
          "ttl_ceil": 86400,
          "ssl_id": null,
          "ssl_status": null,
          "ssl_expires_on": null,
          "auto_ttl": 1,
          "service_mode": "0",
          "props": {
            "proxiable": 1,
            "cloud_on": 0,
            "cf_open": 1,
            "ssl": 0,
            "expired_ssl": 0,
            "expiring_ssl": 0,
            "pending_ssl": 0
          }
        },
        {
          "rec_id": "17119732",
          "rec_tag": "1faa40f85c78bccb69ee8116e84f3b40",
          "zone_name": "example.com",
          "name": "xn--vii.example.com",
          "display_name": "⟵",
          "type": "CNAME",
          "prio": null,
          "content": "example.com",
          "display_content": "example.com",
          "ttl": "1",
          "ttl_ceil": 86400,
          "ssl_id": null,
          "ssl_status": null,
          "ssl_expires_on": null,
          "auto_ttl": 1,
          "service_mode": "1",
          "props": {
            "proxiable": 1,
            "cloud_on": 1,
            "cf_open": 1,
            "ssl": 0,
            "expired_ssl": 0,
            "expiring_ssl": 0,
            "pending_ssl": 0
          }
        },
        {
          "rec_id": "16606030",
          "rec_tag": "2012b3a2e49978ef18ee13dd98e6b6f7",
          "zone_name": "example.com",
          "name": "yay.example.com",
          "display_name": "yay",
          "type": "CNAME",
          "prio": null,
          "content": "domains.tumblr.com",
          "display_content": "domains.tumblr.com",
          "ttl": "1",
          "ttl_ceil": 86400,
          "ssl_id": null,
          "ssl_status": null,
          "ssl_expires_on": null,
          "auto_ttl": 1,
          "service_mode": "0",
          "props": {
            "proxiable": 1,
            "cloud_on": 0,
            "cf_open": 1,
            "ssl": 0,
            "expired_ssl": 0,
            "expiring_ssl": 0,
            "pending_ssl": 0
          }
        }
      ]
    }
  },
  "result": "success",
  "msg": null
}`

var recordExample = `{
  "request": {
    "act": "rec_new",
    "a": "rec_new",
    "tkn": "8afbe6dea02407989af4dd4c97bb6e25",
    "email": "sample@example.com",
    "type": "A",
    "z": "example.com",
    "name": "test",
    "content": "96.126.126.36",
    "ttl": "1",
    "service_mode": "1"
  },
  "response": {
    "rec": {
      "obj": {
        "rec_id": "23734516",
        "rec_tag": "b3db8b8ad50389eb4abae7522b22852f",
        "zone_name": "example.com",
        "name": "test.example.com",
        "display_name": "test",
        "type": "A",
        "prio": null,
        "content": "96.126.126.36",
        "display_content": "96.126.126.36",
        "ttl": "1",
        "ttl_ceil": 86400,
        "ssl_id": "12805",
        "ssl_status": "V",
        "ssl_expires_on": null,
        "auto_ttl": 1,
        "service_mode": "0",
        "props": {
          "proxiable": 1,
          "cloud_on": 0,
          "cf_open": 1,
          "ssl": 1,
          "expired_ssl": 0,
          "expiring_ssl": 0,
          "pending_ssl": 0,
          "vanity_lock": 0
        }
      }
    }
  },
  "result": "success",
  "msg": null
}`
