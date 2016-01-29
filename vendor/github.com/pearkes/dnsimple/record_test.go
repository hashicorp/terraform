package dnsimple

import (
	"github.com/pearkes/dnsimple/testutil"
	"strings"
	"testing"

	. "github.com/motain/gocheck"
)

func Test(t *testing.T) {
	TestingT(t)
}

type S struct {
	client *Client
}

var _ = Suite(&S{})

var testServer = testutil.NewHTTPServer()

func (s *S) SetUpSuite(c *C) {
	testServer.Start()
	var err error
	s.client, err = NewClient("email", "foobar")
	s.client.URL = "http://localhost:4444"
	if err != nil {
		panic(err)
	}
}

func (s *S) TearDownTest(c *C) {
	testServer.Flush()
}

func (s *S) Test_CreateRecord(c *C) {
	testServer.Response(202, nil, recordExample)

	opts := ChangeRecord{
		Name: "foobar",
	}

	id, err := s.client.CreateRecord("example.com", &opts)

	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(id, Equals, "25")
}

func (s *S) Test_UpdateRecord(c *C) {
	testServer.Response(200, nil, recordExample)

	opts := ChangeRecord{
		Name: "foobar",
	}

	id, err := s.client.UpdateRecord("example.com", "25", &opts)

	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(id, Equals, "25")
}

func (s *S) Test_CreateRecord_fail(c *C) {
	testServer.Response(400, nil, recordExampleError)

	opts := ChangeRecord{
		Name: "foobar",
	}

	_, err := s.client.CreateRecord("example.com", &opts)

	_ = testServer.WaitRequest()

	c.Assert(strings.Contains(err.Error(), "unsupported"), Equals, true)
}

func (s *S) Test_RetrieveRecord(c *C) {
	testServer.Response(200, nil, recordExample)

	record, err := s.client.RetrieveRecord("example.com", "25")

	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(record.StringId(), Equals, "25")
	c.Assert(record.StringDomainId(), Equals, "28")
	c.Assert(record.StringTtl(), Equals, "3600")
	c.Assert(record.Name, Equals, "foobar")
	c.Assert(record.Content, Equals, "mail.example.com")
	c.Assert(record.RecordType, Equals, "MX")
}

var recordExampleError = `{
  "errors": {
    "content": ["can't be blank"],
    "record_type": ["can't be blank","unsupported"]
  }
}`

var recordExample = `{
  "record": {
    "content": "mail.example.com",
    "created_at": "2013-01-29T14:25:38Z",
    "domain_id": 28,
    "id": 25,
    "name": "foobar",
    "prio": 10,
    "record_type": "MX",
    "ttl": 3600,
    "updated_at": "2013-01-29T14:25:38Z"
  }
}`
