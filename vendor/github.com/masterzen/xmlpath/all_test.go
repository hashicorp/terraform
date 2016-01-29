package xmlpath_test

import (
	"bytes"
	"encoding/xml"
	. "launchpad.net/gocheck"
	"launchpad.net/xmlpath"
	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&BasicSuite{})

type BasicSuite struct{}

var trivialXml = []byte(`<root>a<foo>b</foo>c<bar>d</bar>e<bar>f</bar>g</root>`)

func (s *BasicSuite) TestRootText(c *C) {
	node, err := xmlpath.Parse(bytes.NewBuffer(trivialXml))
	c.Assert(err, IsNil)
	path := xmlpath.MustCompile("/")
	result, ok := path.String(node)
	c.Assert(ok, Equals, true)
	c.Assert(result, Equals, "abcdefg")
}

var trivialHtml = []byte(`<root><foo>&lt;a&gt;</root>`)

func (s *BasicSuite) TestHTML(c *C) {
	node, err := xmlpath.ParseHTML(bytes.NewBuffer(trivialHtml))
	c.Assert(err, IsNil)
	path := xmlpath.MustCompile("/root/foo")
	result, ok := path.String(node)
	c.Assert(ok, Equals, true)
	c.Assert(result, Equals, "<a>")
}

func (s *BasicSuite) TestLibraryTable(c *C) {
	node, err := xmlpath.Parse(bytes.NewBuffer(libraryXml))
	c.Assert(err, IsNil)
	for _, test := range libraryTable {
		cmt := Commentf("xml path: %s", test.path)
		path, err := xmlpath.Compile(test.path)
		if want, ok := test.result.(cerror); ok {
			c.Assert(err, ErrorMatches, string(want), cmt)
			c.Assert(path, IsNil, cmt)
			continue
		}
		c.Assert(err, IsNil)
		switch want := test.result.(type) {
		case string:
			got, ok := path.String(node)
			c.Assert(ok, Equals, true, cmt)
			c.Assert(got, Equals, want, cmt)
			c.Assert(path.Exists(node), Equals, true, cmt)
			iter := path.Iter(node)
			iter.Next()
			node := iter.Node()
			c.Assert(node.String(), Equals, want, cmt)
			c.Assert(string(node.Bytes()), Equals, want, cmt)
		case []string:
			var alls []string
			var allb []string
			iter := path.Iter(node)
			for iter.Next() {
				alls = append(alls, iter.Node().String())
				allb = append(allb, string(iter.Node().Bytes()))
			}
			c.Assert(alls, DeepEquals, want, cmt)
			c.Assert(allb, DeepEquals, want, cmt)
			s, sok := path.String(node)
			b, bok := path.Bytes(node)
			if len(want) == 0 {
				c.Assert(sok, Equals, false, cmt)
				c.Assert(bok, Equals, false, cmt)
				c.Assert(s, Equals, "")
				c.Assert(b, IsNil)
			} else {
				c.Assert(sok, Equals, true, cmt)
				c.Assert(bok, Equals, true, cmt)
				c.Assert(s, Equals, alls[0], cmt)
				c.Assert(string(b), Equals, alls[0], cmt)
				c.Assert(path.Exists(node), Equals, true, cmt)
			}
		case exists:
			wantb := bool(want)
			ok := path.Exists(node)
			c.Assert(ok, Equals, wantb, cmt)
			_, ok = path.String(node)
			c.Assert(ok, Equals, wantb, cmt)
		}
	}
}

type cerror string
type exists bool

var libraryTable = []struct{ path string; result interface{} }{
	// These are the examples in the package documentation:
	{"/library/book/isbn", "0836217462"},
	{"library/*/isbn", "0836217462"},
	{"/library/book/../book/./isbn", "0836217462"},
	{"/library/book/character[2]/name", "Snoopy"},
	{"/library/book/character[born='1950-10-04']/name", "Snoopy"},
	{"/library/book//node()[@id='PP']/name", "Peppermint Patty"},
	{"//book[author/@id='CMS']/title", "Being a Dog Is a Full-Time Job"},
	{"/library/book/preceding::comment()", " Great book. "},

	// A few simple
	{"/library/book/isbn", exists(true)},
	{"/library/isbn", exists(false)},
	{"/library/book/isbn/bad", exists(false)},
	{"/library/book/bad", exists(false)},
	{"/library/bad/isbn", exists(false)},
	{"/bad/book/isbn", exists(false)},

	// Simple paths.
	{"/library/book/isbn", "0836217462"},
	{"/library/book/author/name", "Charles M Schulz"},
	{"/library/book/author/born", "1922-11-26"},
	{"/library/book/character/name", "Peppermint Patty"},
	{"/library/book/character/qualification", "bold, brash and tomboyish"},

	// Unrooted path with root node as context.
	{"library/book/isbn", "0836217462"},

	// Multiple entries from simple paths.
	{"/library/book/isbn", []string{"0836217462", "0883556316"}},
	{"/library/book/character/name", []string{"Peppermint Patty", "Snoopy", "Schroeder", "Lucy", "Barney Google", "Spark Plug", "Snuffy Smith"}},

	// Handling of wildcards.
	{"/library/book/author/*", []string{"Charles M Schulz", "1922-11-26", "2000-02-12", "Charles M Schulz", "1922-11-26", "2000-02-12"}},

	// Unsupported axis and note test.
	{"/foo()", cerror(`compiling xml path "/foo\(\)":5: unsupported expression: foo\(\)`)},
	{"/foo::node()", cerror(`compiling xml path "/foo::node\(\)":6: unsupported axis: "foo"`)},

	// The attribute axis.
	{"/library/book/title/attribute::lang", "en"},
	{"/library/book/title/@lang", "en"},
	{"/library/book/@available/parent::node()/@id", "b0836217462"},
	{"/library/book/attribute::*", []string{"b0836217462", "true", "b0883556316", "true"}},
	{"/library/book/attribute::text()", cerror(`.*: text\(\) cannot succeed on axis "attribute"`)},

	// The self axis.
	{"/library/book/isbn/./self::node()", "0836217462"},

	// The descendant axis.
	{"/library/book/isbn/descendant::isbn", exists(false)},
	{"/library/descendant::isbn", []string{"0836217462", "0883556316"}},
	{"/descendant::*/isbn", []string{"0836217462", "0883556316"}},
	{"/descendant::isbn", []string{"0836217462", "0883556316"}},

	// The descendant-or-self axis.
	{"/library/book/isbn/descendant-or-self::isbn", "0836217462"},
	{"/library//isbn", []string{"0836217462", "0883556316"}},
	{"//isbn", []string{"0836217462", "0883556316"}},
	{"/descendant-or-self::node()/child::book/child::*", "0836217462"},

	// The parent axis.
	{"/library/book/isbn/../isbn/parent::node()//title", "Being a Dog Is a Full-Time Job"},

	// The ancestor axis.
	{"/library/book/isbn/ancestor::book/title", "Being a Dog Is a Full-Time Job"},
	{"/library/book/ancestor::book/title", exists(false)},

	// The ancestor-or-self axis.
	{"/library/book/isbn/ancestor-or-self::book/title", "Being a Dog Is a Full-Time Job"},
	{"/library/book/ancestor-or-self::book/title", "Being a Dog Is a Full-Time Job"},

	// The following axis.
	// The first author name must not be included, as it's within the context
	// node (author) rather than following it. These queries exercise de-duping
	// of nodes, since the following axis runs to the end multiple times.
	{"/library/book/author/following::name", []string{"Peppermint Patty", "Snoopy", "Schroeder", "Lucy", "Charles M Schulz", "Barney Google", "Spark Plug", "Snuffy Smith"}},
	{"//following::book/author/name", []string{"Charles M Schulz", "Charles M Schulz"}},

	// The following-sibling axis.
	{"/library/book/quote/following-sibling::node()/name", []string{"Charles M Schulz", "Peppermint Patty", "Snoopy", "Schroeder", "Lucy"}},

	// The preceding axis.
	{"/library/book/author/born/preceding::name", []string{"Charles M Schulz", "Charles M Schulz", "Lucy", "Schroeder", "Snoopy", "Peppermint Patty"}},
	{"/library/book/author/born/preceding::author/name", []string{"Charles M Schulz"}},
	{"/library/book/author/born/preceding::library", exists(false)},

	// The preceding-sibling axis.
	{"/library/book/author/born/preceding-sibling::name", []string{"Charles M Schulz", "Charles M Schulz"}},
	{"/library/book/author/born/preceding::author/name", []string{"Charles M Schulz"}},

	// Comments.
	{"/library/comment()", []string{" Great book. ", " Another great book. "}},
	{"//self::comment()", []string{" Great book. ", " Another great book. "}},
	{`comment("")`, cerror(`.*: comment\(\) has no arguments`)},


	// Processing instructions.
	{`/library/book/author/processing-instruction()`, `"go rocks"`},
	{`/library/book/author/processing-instruction("echo")`, `"go rocks"`},
	{`/library//processing-instruction("echo")`, `"go rocks"`},
	{`/library/book/author/processing-instruction("foo")`, exists(false)},
	{`/library/book/author/processing-instruction(")`, cerror(`.*: missing '"'`)},

	// Predicates.
	{"library/book[@id='b0883556316']/isbn", []string{"0883556316"}},
	{"library/book[isbn='0836217462']/character[born='1950-10-04']/name", []string{"Snoopy"}},
	{"library/book[quote]/@id", []string{"b0836217462"}},
	{"library/book[./character/born='1922-07-17']/@id", []string{"b0883556316"}},
	{"library/book[2]/isbn", []string{"0883556316"}},
	{"library/book[0]/isbn", cerror(".*: positions start at 1")},
	{"library/book[-1]/isbn", cerror(".*: positions must be positive")},

	// Bogus expressions.
	{"/foo)", cerror(`compiling xml path "/foo\)":4: unexpected '\)'`)},
}

var libraryXml = []byte(
`<?xml version="1.0"?> 
<library>
  <!-- Great book. -->
  <book id="b0836217462" available="true">
    <isbn>0836217462</isbn>
    <title lang="en">Being a Dog Is a Full-Time Job</title>
    <quote>I'd dog paddle the deepest ocean.</quote>
    <author id="CMS">
      <?echo "go rocks"?>
      <name>Charles M Schulz</name>
      <born>1922-11-26</born>
      <dead>2000-02-12</dead>
    </author>
    <character id="PP">
      <name>Peppermint Patty</name>
      <born>1966-08-22</born>
      <qualification>bold, brash and tomboyish</qualification>
    </character>
    <character id="Snoopy">
      <name>Snoopy</name>
      <born>1950-10-04</born>
      <qualification>extroverted beagle</qualification>
    </character>
    <character id="Schroeder">
      <name>Schroeder</name>
      <born>1951-05-30</born>
      <qualification>brought classical music to the Peanuts strip</qualification>
    </character>
    <character id="Lucy">
      <name>Lucy</name>
      <born>1952-03-03</born>
      <qualification>bossy, crabby and selfish</qualification>
    </character>
  </book>
  <!-- Another great book. -->
  <book id="b0883556316" available="true">
    <isbn>0883556316</isbn>
    <title lang="en">Barney Google and Snuffy Smith</title>
    <author id="CMS">
      <name>Charles M Schulz</name>
      <born>1922-11-26</born>
      <dead>2000-02-12</dead>
    </author>
    <character id="Barney">
      <name>Barney Google</name>
      <born>1919-01-01</born>
      <qualification>goggle-eyed, moustached, gloved and top-hatted, bulbous-nosed, cigar-chomping shrimp</qualification>
    </character>
    <character id="Spark">
      <name>Spark Plug</name>
      <born>1922-07-17</born>
      <qualification>brown-eyed, bow-legged nag, seldom races, patched blanket</qualification>
    </character>
    <character id="Snuffy">
      <name>Snuffy Smith</name>
      <born>1934-01-01</born>
      <qualification>volatile and diminutive moonshiner, ornery little cuss, sawed-off and shiftless</qualification>
    </character>
  </book>
</library>
`)

func (s *BasicSuite) TestNamespace(c *C) {
	node, err := xmlpath.Parse(bytes.NewBuffer(namespaceXml))
	c.Assert(err, IsNil)
	for _, test := range namespaceTable {
		cmt := Commentf("xml path: %s", test.path)
		path, err := xmlpath.CompileWithNamespace(test.path, namespaces)
		if want, ok := test.result.(cerror); ok {
			c.Assert(err, ErrorMatches, string(want), cmt)
			c.Assert(path, IsNil, cmt)
			continue
		}
		c.Assert(err, IsNil)
		switch want := test.result.(type) {
		case string:
			got, ok := path.String(node)
			c.Assert(ok, Equals, true, cmt)
			c.Assert(got, Equals, want, cmt)
			c.Assert(path.Exists(node), Equals, true, cmt)
			iter := path.Iter(node)
			iter.Next()
			node := iter.Node()
			c.Assert(node.String(), Equals, want, cmt)
			c.Assert(string(node.Bytes()), Equals, want, cmt)
		case []string:
			var alls []string
			var allb []string
			iter := path.Iter(node)
			for iter.Next() {
				alls = append(alls, iter.Node().String())
				allb = append(allb, string(iter.Node().Bytes()))
			}
			c.Assert(alls, DeepEquals, want, cmt)
			c.Assert(allb, DeepEquals, want, cmt)
			s, sok := path.String(node)
			b, bok := path.Bytes(node)
			if len(want) == 0 {
				c.Assert(sok, Equals, false, cmt)
				c.Assert(bok, Equals, false, cmt)
				c.Assert(s, Equals, "")
				c.Assert(b, IsNil)
			} else {
				c.Assert(sok, Equals, true, cmt)
				c.Assert(bok, Equals, true, cmt)
				c.Assert(s, Equals, alls[0], cmt)
				c.Assert(string(b), Equals, alls[0], cmt)
				c.Assert(path.Exists(node), Equals, true, cmt)
			}
		case exists:
			wantb := bool(want)
			ok := path.Exists(node)
			c.Assert(ok, Equals, wantb, cmt)
			_, ok = path.String(node)
			c.Assert(ok, Equals, wantb, cmt)
		}
	}
}

var namespaceXml = []byte(`<s:Envelope xml:lang="en-US" xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:w="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell" xmlns:p="http://schemas.microsoft.com/wbem/wsman/1/wsman.xsd"><s:Header><a:Action>http://schemas.microsoft.com/wbem/wsman/1/windows/shell/ReceiveResponse</a:Action><a:MessageID>uuid:AAD46BD4-6315-4C3C-93D4-94A55773287D</a:MessageID><a:To>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:To><a:RelatesTo>uuid:18A52A06-9027-41DC-8850-3F244595AF62</a:RelatesTo></s:Header><s:Body><rsp:ReceiveResponse><rsp:Stream Name="stdout" CommandId="1A6DEE6B-EC68-4DD6-87E9-030C0048ECC4">VGhhdCdzIGFsbCBmb2xrcyEhIQ==</rsp:Stream><rsp:Stream Name="stderr" CommandId="1A6DEE6B-EC68-4DD6-87E9-030C0048ECC4">VGhpcyBpcyBzdGRlcnIsIEknbSBwcmV0dHkgc3VyZSE=</rsp:Stream><rsp:CommandState CommandId="1A6DEE6B-EC68-4DD6-87E9-030C0048ECC4" State="http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandState/Running"></rsp:CommandState></rsp:ReceiveResponse></s:Body></s:Envelope>`)

var namespaces = []xmlpath.Namespace {
	{ "a", "http://schemas.xmlsoap.org/ws/2004/08/addressing" },
	{ "rsp", "http://schemas.microsoft.com/wbem/wsman/1/windows/shell" },
}

var namespaceTable = []struct{ path string; result interface{} }{
	{ "//a:To", "http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous" },
	{ "//rsp:Stream[@Name='stdout']", "VGhhdCdzIGFsbCBmb2xrcyEhIQ==" },
	{ "//rsp:CommandState/@CommandId", "1A6DEE6B-EC68-4DD6-87E9-030C0048ECC4" },
	{ "//*[@State='http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandState/Done']", exists(false) },
	{ "//rsp:Stream", []string{ "VGhhdCdzIGFsbCBmb2xrcyEhIQ==", "VGhpcyBpcyBzdGRlcnIsIEknbSBwcmV0dHkgc3VyZSE=" }},
	{ "//s:Header", cerror(`.*: unknown namespace prefix: s`) },
}

func (s *BasicSuite) BenchmarkParse(c *C) {
	for i := 0; i < c.N; i++ {
		_, err := xmlpath.Parse(bytes.NewBuffer(instancesXml))
		c.Assert(err, IsNil)
	}
}

func (s *BasicSuite) BenchmarkSimplePathCompile(c *C) {
	var err error
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		_, err = xmlpath.Compile("/DescribeInstancesResponse/reservationSet/item/groupSet/item/groupId")
	}
	c.StopTimer()
	c.Assert(err, IsNil)
}

func (s *BasicSuite) BenchmarkSimplePathString(c *C) {
	node, err := xmlpath.Parse(bytes.NewBuffer(instancesXml))
	c.Assert(err, IsNil)
	path := xmlpath.MustCompile("/DescribeInstancesResponse/reservationSet/item/instancesSet/item/instanceType")
	var str string
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		str, _ = path.String(node)
	}
	c.StopTimer()
	c.Assert(str, Equals, "m1.small")
}

func (s *BasicSuite) BenchmarkSimplePathStringUnmarshal(c *C) {
	// For a vague comparison.
	var result struct{ Str string `xml:"reservationSet>item>instancesSet>item>instanceType"` }
	for i := 0; i < c.N; i++ {
		xml.Unmarshal(instancesXml, &result)
	}
	c.StopTimer()
	c.Assert(result.Str, Equals, "m1.large")
}

func (s *BasicSuite) BenchmarkSimplePathExists(c *C) {
	node, err := xmlpath.Parse(bytes.NewBuffer(instancesXml))
	c.Assert(err, IsNil)
	path := xmlpath.MustCompile("/DescribeInstancesResponse/reservationSet/item/instancesSet/item/instanceType")
	var exists bool
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		exists = path.Exists(node)
	}
	c.StopTimer()
	c.Assert(exists, Equals, true)
}



var instancesXml = []byte(
`<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2011-12-15/">
  <requestId>98e3c9a4-848c-4d6d-8e8a-b1bdEXAMPLE</requestId>
  <reservationSet>
    <item>
      <reservationId>r-b27e30d9</reservationId>
      <ownerId>999988887777</ownerId>
      <groupSet>
        <item>
          <groupId>sg-67ad940e</groupId>
          <groupName>default</groupName>
        </item>
      </groupSet>
      <instancesSet>
        <item>
          <instanceId>i-c5cd56af</instanceId>
          <imageId>ami-1a2b3c4d</imageId>
          <instanceState>
            <code>16</code>
            <name>running</name>
          </instanceState>
          <privateDnsName>domU-12-31-39-10-56-34.compute-1.internal</privateDnsName>
          <dnsName>ec2-174-129-165-232.compute-1.amazonaws.com</dnsName>
          <reason/>
          <keyName>GSG_Keypair</keyName>
          <amiLaunchIndex>0</amiLaunchIndex>
          <productCodes/>
          <instanceType>m1.small</instanceType>
          <launchTime>2010-08-17T01:15:18.000Z</launchTime>
          <placement>
            <availabilityZone>us-east-1b</availabilityZone>
            <groupName/>
          </placement>
          <kernelId>aki-94c527fd</kernelId>
          <ramdiskId>ari-96c527ff</ramdiskId>
          <monitoring>
            <state>disabled</state>
          </monitoring>
          <privateIpAddress>10.198.85.190</privateIpAddress>
          <ipAddress>174.129.165.232</ipAddress>
          <architecture>i386</architecture>
          <rootDeviceType>ebs</rootDeviceType>
          <rootDeviceName>/dev/sda1</rootDeviceName>
          <blockDeviceMapping>
            <item>
              <deviceName>/dev/sda1</deviceName>
              <ebs>
                <volumeId>vol-a082c1c9</volumeId>
                <status>attached</status>
                <attachTime>2010-08-17T01:15:21.000Z</attachTime>
                <deleteOnTermination>false</deleteOnTermination>
              </ebs>
            </item>
          </blockDeviceMapping>
          <instanceLifecycle>spot</instanceLifecycle>
          <spotInstanceRequestId>sir-7a688402</spotInstanceRequestId>
          <virtualizationType>paravirtual</virtualizationType>
          <clientToken/>
          <tagSet/>
          <hypervisor>xen</hypervisor>
       </item>
      </instancesSet>
      <requesterId>854251627541</requesterId>
    </item>
    <item>
      <reservationId>r-b67e30dd</reservationId>
      <ownerId>999988887777</ownerId>
      <groupSet>
        <item>
          <groupId>sg-67ad940e</groupId>
          <groupName>default</groupName>
        </item>
      </groupSet>
      <instancesSet>
        <item>
          <instanceId>i-d9cd56b3</instanceId>
          <imageId>ami-1a2b3c4d</imageId>
          <instanceState>
            <code>16</code>
            <name>running</name>
          </instanceState>
          <privateDnsName>domU-12-31-39-10-54-E5.compute-1.internal</privateDnsName>
          <dnsName>ec2-184-73-58-78.compute-1.amazonaws.com</dnsName>
          <reason/>
          <keyName>GSG_Keypair</keyName>
          <amiLaunchIndex>0</amiLaunchIndex>
          <productCodes/>
          <instanceType>m1.large</instanceType>
          <launchTime>2010-08-17T01:15:19.000Z</launchTime>
          <placement>
            <availabilityZone>us-east-1b</availabilityZone>
            <groupName/>
          </placement>
          <kernelId>aki-94c527fd</kernelId>
          <ramdiskId>ari-96c527ff</ramdiskId>
          <monitoring>
            <state>disabled</state>
          </monitoring>
          <privateIpAddress>10.198.87.19</privateIpAddress>
          <ipAddress>184.73.58.78</ipAddress>
          <architecture>i386</architecture>
          <rootDeviceType>ebs</rootDeviceType>
          <rootDeviceName>/dev/sda1</rootDeviceName>
          <blockDeviceMapping>
            <item>
              <deviceName>/dev/sda1</deviceName>
              <ebs>
                <volumeId>vol-a282c1cb</volumeId>
                <status>attached</status>
                <attachTime>2010-08-17T01:15:23.000Z</attachTime>
                <deleteOnTermination>false</deleteOnTermination>
              </ebs>
            </item>
          </blockDeviceMapping>
          <instanceLifecycle>spot</instanceLifecycle>
          <spotInstanceRequestId>sir-55a3aa02</spotInstanceRequestId>
          <virtualizationType>paravirtual</virtualizationType>
          <clientToken/>
          <tagSet/>
          <hypervisor>xen</hypervisor>
       </item>
      </instancesSet>
      <requesterId>854251627541</requesterId>
    </item>
  </reservationSet>
</DescribeInstancesResponse>
`)
