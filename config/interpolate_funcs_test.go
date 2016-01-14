package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config/lang"
	"github.com/hashicorp/terraform/config/lang/ast"
)

func TestInterpolateFuncCompact(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			// empty string within array
			{
				`${compact(split(",", "a,,b"))}`,
				NewStringList([]string{"a", "b"}).String(),
				false,
			},

			// empty string at the end of array
			{
				`${compact(split(",", "a,b,"))}`,
				NewStringList([]string{"a", "b"}).String(),
				false,
			},

			// single empty string
			{
				`${compact(split(",", ""))}`,
				NewStringList([]string{}).String(),
				false,
			},
		},
	})
}

func TestInterpolateFuncCidrHost(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			{
				`${cidrhost("192.168.1.0/24", 5)}`,
				"192.168.1.5",
				false,
			},
			{
				`${cidrhost("192.168.1.0/30", 255)}`,
				nil,
				true, // 255 doesn't fit in two bits
			},
			{
				`${cidrhost("not-a-cidr", 6)}`,
				nil,
				true, // not a valid CIDR mask
			},
			{
				`${cidrhost("10.256.0.0/8", 6)}`,
				nil,
				true, // can't have an octet >255
			},
		},
	})
}

func TestInterpolateFuncCidrNetmask(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			{
				`${cidrnetmask("192.168.1.0/24")}`,
				"255.255.255.0",
				false,
			},
			{
				`${cidrnetmask("192.168.1.0/32")}`,
				"255.255.255.255",
				false,
			},
			{
				`${cidrnetmask("0.0.0.0/0")}`,
				"0.0.0.0",
				false,
			},
			{
				// This doesn't really make sense for IPv6 networks
				// but it ought to do something sensible anyway.
				`${cidrnetmask("1::/64")}`,
				"ffff:ffff:ffff:ffff::",
				false,
			},
			{
				`${cidrnetmask("not-a-cidr")}`,
				nil,
				true, // not a valid CIDR mask
			},
			{
				`${cidrnetmask("10.256.0.0/8")}`,
				nil,
				true, // can't have an octet >255
			},
		},
	})
}

func TestInterpolateFuncCidrSubnet(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			{
				`${cidrsubnet("192.168.2.0/20", 4, 6)}`,
				"192.168.6.0/24",
				false,
			},
			{
				`${cidrsubnet("fe80::/48", 16, 6)}`,
				"fe80:0:0:6::/64",
				false,
			},
			{
				// IPv4 address encoded in IPv6 syntax gets normalized
				`${cidrsubnet("::ffff:192.168.0.0/112", 8, 6)}`,
				"192.168.6.0/24",
				false,
			},
			{
				`${cidrsubnet("192.168.0.0/30", 4, 6)}`,
				nil,
				true, // not enough bits left
			},
			{
				`${cidrsubnet("192.168.0.0/16", 2, 16)}`,
				nil,
				true, // can't encode 16 in 2 bits
			},
			{
				`${cidrsubnet("not-a-cidr", 4, 6)}`,
				nil,
				true, // not a valid CIDR mask
			},
			{
				`${cidrsubnet("10.256.0.0/8", 4, 6)}`,
				nil,
				true, // can't have an octet >255
			},
		},
	})
}

func TestInterpolateFuncCoalesce(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			{
				`${coalesce("first", "second", "third")}`,
				"first",
				false,
			},
			{
				`${coalesce("", "second", "third")}`,
				"second",
				false,
			},
			{
				`${coalesce("", "", "")}`,
				"",
				false,
			},
			{
				`${coalesce("foo")}`,
				nil,
				true,
			},
		},
	})
}

func TestInterpolateFuncDeprecatedConcat(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			{
				`${concat("foo", "bar")}`,
				"foobar",
				false,
			},

			{
				`${concat("foo")}`,
				"foo",
				false,
			},

			{
				`${concat()}`,
				nil,
				true,
			},
		},
	})
}

func TestInterpolateFuncConcat(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			// String + list
			{
				`${concat("a", split(",", "b,c"))}`,
				NewStringList([]string{"a", "b", "c"}).String(),
				false,
			},

			// List + string
			{
				`${concat(split(",", "a,b"), "c")}`,
				NewStringList([]string{"a", "b", "c"}).String(),
				false,
			},

			// Single list
			{
				`${concat(split(",", ",foo,"))}`,
				NewStringList([]string{"", "foo", ""}).String(),
				false,
			},
			{
				`${concat(split(",", "a,b,c"))}`,
				NewStringList([]string{"a", "b", "c"}).String(),
				false,
			},

			// Two lists
			{
				`${concat(split(",", "a,b,c"), split(",", "d,e"))}`,
				NewStringList([]string{"a", "b", "c", "d", "e"}).String(),
				false,
			},
			// Two lists with different separators
			{
				`${concat(split(",", "a,b,c"), split(" ", "d e"))}`,
				NewStringList([]string{"a", "b", "c", "d", "e"}).String(),
				false,
			},

			// More lists
			{
				`${concat(split(",", "a,b"), split(",", "c,d"), split(",", "e,f"), split(",", "0,1"))}`,
				NewStringList([]string{"a", "b", "c", "d", "e", "f", "0", "1"}).String(),
				false,
			},
		},
	})
}

func TestInterpolateFuncFile(t *testing.T) {
	tf, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	path := tf.Name()
	tf.Write([]byte("foo"))
	tf.Close()
	defer os.Remove(path)

	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			{
				fmt.Sprintf(`${file("%s")}`, path),
				"foo",
				false,
			},

			// Invalid path
			{
				`${file("/i/dont/exist")}`,
				nil,
				true,
			},

			// Too many args
			{
				`${file("foo", "bar")}`,
				nil,
				true,
			},
		},
	})
}

func TestInterpolateFuncFormat(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			{
				`${format("hello")}`,
				"hello",
				false,
			},

			{
				`${format("hello %s", "world")}`,
				"hello world",
				false,
			},

			{
				`${format("hello %d", 42)}`,
				"hello 42",
				false,
			},

			{
				`${format("hello %05d", 42)}`,
				"hello 00042",
				false,
			},

			{
				`${format("hello %05d", 12345)}`,
				"hello 12345",
				false,
			},
		},
	})
}

func TestInterpolateFuncFormatList(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			// formatlist requires at least one list
			{
				`${formatlist("hello")}`,
				nil,
				true,
			},
			{
				`${formatlist("hello %s", "world")}`,
				nil,
				true,
			},
			// formatlist applies to each list element in turn
			{
				`${formatlist("<%s>", split(",", "A,B"))}`,
				NewStringList([]string{"<A>", "<B>"}).String(),
				false,
			},
			// formatlist repeats scalar elements
			{
				`${join(", ", formatlist("%s=%s", "x", split(",", "A,B,C")))}`,
				"x=A, x=B, x=C",
				false,
			},
			// Multiple lists are walked in parallel
			{
				`${join(", ", formatlist("%s=%s", split(",", "A,B,C"), split(",", "1,2,3")))}`,
				"A=1, B=2, C=3",
				false,
			},
			// Mismatched list lengths generate an error
			{
				`${formatlist("%s=%2s", split(",", "A,B,C,D"), split(",", "1,2,3"))}`,
				nil,
				true,
			},
			// Works with lists of length 1 [GH-2240]
			{
				`${formatlist("%s.id", split(",", "demo-rest-elb"))}`,
				NewStringList([]string{"demo-rest-elb.id"}).String(),
				false,
			},
		},
	})
}

func TestInterpolateFuncIndex(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			{
				`${index("test", "")}`,
				nil,
				true,
			},

			{
				fmt.Sprintf(`${index("%s", "foo")}`,
					NewStringList([]string{"notfoo", "stillnotfoo", "bar"}).String()),
				nil,
				true,
			},

			{
				fmt.Sprintf(`${index("%s", "foo")}`,
					NewStringList([]string{"foo"}).String()),
				"0",
				false,
			},

			{
				fmt.Sprintf(`${index("%s", "bar")}`,
					NewStringList([]string{"foo", "spam", "bar", "eggs"}).String()),
				"2",
				false,
			},
		},
	})
}

func TestInterpolateFuncJoin(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			{
				`${join(",")}`,
				nil,
				true,
			},

			{
				fmt.Sprintf(`${join(",", "%s")}`,
					NewStringList([]string{"foo"}).String()),
				"foo",
				false,
			},

			/*
				TODO
				{
					`${join(",", "foo", "bar")}`,
					"foo,bar",
					false,
				},
			*/

			{
				fmt.Sprintf(`${join(".", "%s")}`,
					NewStringList([]string{"foo", "bar", "baz"}).String()),
				"foo.bar.baz",
				false,
			},
		},
	})
}

func TestInterpolateFuncReplace(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			// Regular search and replace
			{
				`${replace("hello", "hel", "bel")}`,
				"bello",
				false,
			},

			// Search string doesn't match
			{
				`${replace("hello", "nope", "bel")}`,
				"hello",
				false,
			},

			// Regular expression
			{
				`${replace("hello", "/l/", "L")}`,
				"heLLo",
				false,
			},

			{
				`${replace("helo", "/(l)/", "$1$1")}`,
				"hello",
				false,
			},

			// Bad regexp
			{
				`${replace("helo", "/(l/", "$1$1")}`,
				nil,
				true,
			},
		},
	})
}

func TestInterpolateFuncLength(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			// Raw strings
			{
				`${length("")}`,
				"0",
				false,
			},
			{
				`${length("a")}`,
				"1",
				false,
			},
			{
				`${length(" ")}`,
				"1",
				false,
			},
			{
				`${length(" a ,")}`,
				"4",
				false,
			},
			{
				`${length("aaa")}`,
				"3",
				false,
			},

			// Lists
			{
				`${length(split(",", "a"))}`,
				"1",
				false,
			},
			{
				`${length(split(",", "foo,"))}`,
				"2",
				false,
			},
			{
				`${length(split(",", ",foo,"))}`,
				"3",
				false,
			},
			{
				`${length(split(",", "foo,bar"))}`,
				"2",
				false,
			},
			{
				`${length(split(".", "one.two.three.four.five"))}`,
				"5",
				false,
			},
			// Want length 0 if we split an empty string then compact
			{
				`${length(compact(split(",", "")))}`,
				"0",
				false,
			},
		},
	})
}

func TestInterpolateFuncSplit(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			{
				`${split(",")}`,
				nil,
				true,
			},

			{
				`${split(",", "")}`,
				NewStringList([]string{""}).String(),
				false,
			},

			{
				`${split(",", "foo")}`,
				NewStringList([]string{"foo"}).String(),
				false,
			},

			{
				`${split(",", ",,,")}`,
				NewStringList([]string{"", "", "", ""}).String(),
				false,
			},

			{
				`${split(",", "foo,")}`,
				NewStringList([]string{"foo", ""}).String(),
				false,
			},

			{
				`${split(",", ",foo,")}`,
				NewStringList([]string{"", "foo", ""}).String(),
				false,
			},

			{
				`${split(".", "foo.bar.baz")}`,
				NewStringList([]string{"foo", "bar", "baz"}).String(),
				false,
			},
		},
	})
}

func TestInterpolateFuncLookup(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Vars: map[string]ast.Variable{
			"var.foo.bar": ast.Variable{
				Value: "baz",
				Type:  ast.TypeString,
			},
		},
		Cases: []testFunctionCase{
			{
				`${lookup("foo", "bar")}`,
				"baz",
				false,
			},

			// Invalid key
			{
				`${lookup("foo", "baz")}`,
				nil,
				true,
			},

			// Too many args
			{
				`${lookup("foo", "bar", "baz")}`,
				nil,
				true,
			},
		},
	})
}

func TestInterpolateFuncKeys(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Vars: map[string]ast.Variable{
			"var.foo.bar": ast.Variable{
				Value: "baz",
				Type:  ast.TypeString,
			},
			"var.foo.qux": ast.Variable{
				Value: "quack",
				Type:  ast.TypeString,
			},
			"var.str": ast.Variable{
				Value: "astring",
				Type:  ast.TypeString,
			},
		},
		Cases: []testFunctionCase{
			{
				`${keys("foo")}`,
				NewStringList([]string{"bar", "qux"}).String(),
				false,
			},

			// Invalid key
			{
				`${keys("not")}`,
				nil,
				true,
			},

			// Too many args
			{
				`${keys("foo", "bar")}`,
				nil,
				true,
			},

			// Not a map
			{
				`${keys("str")}`,
				nil,
				true,
			},
		},
	})
}

func TestInterpolateFuncValues(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Vars: map[string]ast.Variable{
			"var.foo.bar": ast.Variable{
				Value: "quack",
				Type:  ast.TypeString,
			},
			"var.foo.qux": ast.Variable{
				Value: "baz",
				Type:  ast.TypeString,
			},
			"var.str": ast.Variable{
				Value: "astring",
				Type:  ast.TypeString,
			},
		},
		Cases: []testFunctionCase{
			{
				`${values("foo")}`,
				NewStringList([]string{"quack", "baz"}).String(),
				false,
			},

			// Invalid key
			{
				`${values("not")}`,
				nil,
				true,
			},

			// Too many args
			{
				`${values("foo", "bar")}`,
				nil,
				true,
			},

			// Not a map
			{
				`${values("str")}`,
				nil,
				true,
			},
		},
	})
}

func TestInterpolateFuncElement(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			{
				fmt.Sprintf(`${element("%s", "1")}`,
					NewStringList([]string{"foo", "baz"}).String()),
				"baz",
				false,
			},

			{
				fmt.Sprintf(`${element("%s", "0")}`,
					NewStringList([]string{"foo"}).String()),
				"foo",
				false,
			},

			// Invalid index should wrap vs. out-of-bounds
			{
				fmt.Sprintf(`${element("%s", "2")}`,
					NewStringList([]string{"foo", "baz"}).String()),
				"foo",
				false,
			},

			// Too many args
			{
				fmt.Sprintf(`${element("%s", "0", "2")}`,
					NewStringList([]string{"foo", "baz"}).String()),
				nil,
				true,
			},
		},
	})
}

func TestInterpolateFuncBase64Encode(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			// Regular base64 encoding
			{
				`${base64encode("abc123!?$*&()'-=@~")}`,
				"YWJjMTIzIT8kKiYoKSctPUB+",
				false,
			},
		},
	})
}

func TestInterpolateFuncBase64Decode(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			// Regular base64 decoding
			{
				`${base64decode("YWJjMTIzIT8kKiYoKSctPUB+")}`,
				"abc123!?$*&()'-=@~",
				false,
			},

			// Invalid base64 data decoding
			{
				`${base64decode("this-is-an-invalid-base64-data")}`,
				nil,
				true,
			},
		},
	})
}

func TestInterpolateFuncLower(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			{
				`${lower("HELLO")}`,
				"hello",
				false,
			},

			{
				`${lower("")}`,
				"",
				false,
			},

			{
				`${lower()}`,
				nil,
				true,
			},
		},
	})
}

func TestInterpolateFuncUpper(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			{
				`${upper("hello")}`,
				"HELLO",
				false,
			},

			{
				`${upper("")}`,
				"",
				false,
			},

			{
				`${upper()}`,
				nil,
				true,
			},
		},
	})
}

func TestInterpolateFuncSha1(t *testing.T) {
	testFunction(t, testFunctionConfig{
		Cases: []testFunctionCase{
			{
				`${sha1("test")}`,
				"a94a8fe5ccb19ba61c4c0873d391e987982fbbd3",
				false,
			},
		},
	})
}

type testFunctionConfig struct {
	Cases []testFunctionCase
	Vars  map[string]ast.Variable
}

type testFunctionCase struct {
	Input  string
	Result interface{}
	Error  bool
}

func testFunction(t *testing.T, config testFunctionConfig) {
	for i, tc := range config.Cases {
		ast, err := lang.Parse(tc.Input)
		if err != nil {
			t.Fatalf("Case #%d: input: %#v\nerr: %s", i, tc.Input, err)
		}

		out, _, err := lang.Eval(ast, langEvalConfig(config.Vars))
		if err != nil != tc.Error {
			t.Fatalf("Case #%d:\ninput: %#v\nerr: %s", i, tc.Input, err)
		}

		if !reflect.DeepEqual(out, tc.Result) {
			t.Fatalf(
				"%d: bad output for input: %s\n\nOutput: %#v\nExpected: %#v",
				i, tc.Input, out, tc.Result)
		}
	}
}
