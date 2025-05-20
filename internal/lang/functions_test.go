// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package lang

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/experiments"
	"github.com/hashicorp/terraform/internal/lang/marks"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

// TestFunctions tests that functions are callable through the functionality
// in the langs package, via HCL.
//
// These tests are primarily here to assert that the functions are properly
// registered in the functions table, rather than to test all of the details
// of the functions. Each function should only have one or two tests here,
// since the main set of unit tests for a function should live alongside that
// function either in the "funcs" subdirectory here or over in the cty
// function/stdlib package.
//
// One exception to that is we can use this test mechanism to assert common
// patterns that are used in real-world configurations which rely on behaviors
// implemented either in this lang package or in HCL itself, such as automatic
// type conversions. The function unit tests don't cover those things because
// they call directly into the functions.
//
// With that said then, this test function should contain at least one simple
// test case per function registered in the functions table (just to prove
// it really is registered correctly) and possibly a small set of additional
// functions showing real-world use-cases that rely on type conversion
// behaviors.
func TestFunctions(t *testing.T) {
	// used in `pathexpand()` test
	homePath, err := homedir.Dir()
	if err != nil {
		t.Fatalf("Error getting home directory: %v", err)
	}

	tests := map[string][]struct {
		src  string
		want cty.Value
	}{
		// Please maintain this list in alphabetical order by function, with
		// a blank line between the group of tests for each function.

		"abs": {
			{
				`abs(-1)`,
				cty.NumberIntVal(1),
			},
		},

		"abspath": {
			{
				`abspath(".")`,
				cty.StringVal((func() string {
					cwd, err := os.Getwd()
					if err != nil {
						panic(err)
					}
					return filepath.ToSlash(cwd)
				})()),
			},
		},

		"alltrue": {
			{
				`alltrue(["true", true])`,
				cty.True,
			},
		},

		"anytrue": {
			{
				`anytrue([])`,
				cty.False,
			},
		},

		"base64decode": {
			{
				`base64decode("YWJjMTIzIT8kKiYoKSctPUB+")`,
				cty.StringVal("abc123!?$*&()'-=@~"),
			},
		},

		"base64encode": {
			{
				`base64encode("abc123!?$*&()'-=@~")`,
				cty.StringVal("YWJjMTIzIT8kKiYoKSctPUB+"),
			},
		},

		"base64gzip": {
			{
				`base64gzip("test")`,
				cty.StringVal("H4sIAAAAAAAA/ypJLS4BAAAA//8BAAD//wx+f9gEAAAA"),
			},
		},

		"base64sha256": {
			{
				`base64sha256("test")`,
				cty.StringVal("n4bQgYhMfWWaL+qgxVrQFaO/TxsrC4Is0V1sFbDwCgg="),
			},
		},

		"base64sha512": {
			{
				`base64sha512("test")`,
				cty.StringVal("7iaw3Ur350mqGo7jwQrpkj9hiYB3Lkc/iBml1JQODbJ6wYX4oOHV+E+IvIh/1nsUNzLDBMxfqa2Ob1f1ACio/w=="),
			},
		},

		"basename": {
			{
				`basename("testdata/hello.txt")`,
				cty.StringVal("hello.txt"),
			},
		},

		"can": {
			{
				`can(true)`,
				cty.True,
			},
			{
				// Note: "can" only works with expressions that pass static
				// validation, because it only gets an opportunity to run in
				// that case. The following "works" (captures the error) because
				// Terraform understands it as a reference to an attribute
				// that does not exist during dynamic evaluation.
				//
				// "can" doesn't work with references that could never possibly
				// be valid and are thus caught during static validation, such
				// as an expression like "foo" alone which would be understood
				// as an invalid resource reference.
				`can({}.baz)`,
				cty.False,
			},
		},

		"ceil": {
			{
				`ceil(1.2)`,
				cty.NumberIntVal(2),
			},
		},

		"chomp": {
			{
				`chomp("goodbye\ncruel\nworld\n")`,
				cty.StringVal("goodbye\ncruel\nworld"),
			},
		},

		"chunklist": {
			{
				`chunklist(["a", "b", "c"], 1)`,
				cty.ListVal([]cty.Value{
					cty.ListVal([]cty.Value{
						cty.StringVal("a"),
					}),
					cty.ListVal([]cty.Value{
						cty.StringVal("b"),
					}),
					cty.ListVal([]cty.Value{
						cty.StringVal("c"),
					}),
				}),
			},
		},

		"cidrhost": {
			{
				`cidrhost("192.168.1.0/24", 5)`,
				cty.StringVal("192.168.1.5"),
			},
		},

		"cidrnetmask": {
			{
				`cidrnetmask("192.168.1.0/24")`,
				cty.StringVal("255.255.255.0"),
			},
		},

		"cidrsubnet": {
			{
				`cidrsubnet("192.168.2.0/20", 4, 6)`,
				cty.StringVal("192.168.6.0/24"),
			},
		},

		"cidrsubnets": {
			{
				`cidrsubnets("10.0.0.0/8", 8, 8, 16, 8)`,
				cty.ListVal([]cty.Value{
					cty.StringVal("10.0.0.0/16"),
					cty.StringVal("10.1.0.0/16"),
					cty.StringVal("10.2.0.0/24"),
					cty.StringVal("10.3.0.0/16"),
				}),
			},
		},

		"coalesce": {
			{
				`coalesce("first", "second", "third")`,
				cty.StringVal("first"),
			},

			{
				`coalescelist(["first", "second"], ["third", "fourth"])`,
				cty.TupleVal([]cty.Value{
					cty.StringVal("first"), cty.StringVal("second"),
				}),
			},
		},

		"coalescelist": {
			{
				`coalescelist(tolist(["a", "b"]), tolist(["c", "d"]))`,
				cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
				}),
			},
			{
				`coalescelist(["a", "b"], ["c", "d"])`,
				cty.TupleVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
				}),
			},
		},

		"compact": {
			{
				`compact(["test", "", "test"])`,
				cty.ListVal([]cty.Value{
					cty.StringVal("test"), cty.StringVal("test"),
				}),
			},
		},

		"concat": {
			{
				`concat(["a", ""], ["b", "c"])`,
				cty.TupleVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal(""),
					cty.StringVal("b"),
					cty.StringVal("c"),
				}),
			},
		},

		"contains": {
			{
				`contains(["a", "b"], "a")`,
				cty.True,
			},
			{ // Should also work with sets, due to automatic conversion
				`contains(toset(["a", "b"]), "a")`,
				cty.True,
			},
		},

		"csvdecode": {
			{
				`csvdecode("a,b,c\n1,2,3\n4,5,6")`,
				cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"a": cty.StringVal("1"),
						"b": cty.StringVal("2"),
						"c": cty.StringVal("3"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"a": cty.StringVal("4"),
						"b": cty.StringVal("5"),
						"c": cty.StringVal("6"),
					}),
				}),
			},
		},

		"dirname": {
			{
				`dirname("testdata/hello.txt")`,
				cty.StringVal("testdata"),
			},
		},

		"distinct": {
			{
				`distinct(["a", "b", "a", "b"])`,
				cty.ListVal([]cty.Value{
					cty.StringVal("a"), cty.StringVal("b"),
				}),
			},
		},

		"element": {
			{
				`element(["hello"], 0)`,
				cty.StringVal("hello"),
			},
		},

		"endswith": {
			{
				`endswith("hello world", "world")`,
				cty.True,
			},
			{
				`endswith("hello world", "hello")`,
				cty.False,
			},
			{
				`endswith("hello world", "")`,
				cty.True,
				// Completely empty suffix value  ( "" )
				// will always evaluate to true for all strings.
			},
			{
				`endswith("hello world", " ")`,
				cty.False,
			},
			{
				`endswith("", "")`,
				cty.True,
			},
			{
				`endswith("", " ")`,
				cty.False,
			},
			{
				`endswith(" ", "")`,
				cty.True,
			},
			{
				`endswith("", "hello")`,
				cty.False,
			},
			{
				`endswith(" ", "hello")`,
				cty.False,
			},
		},

		"ephemeralasnull": {
			{
				`ephemeralasnull(local.ephemeral)`,
				cty.NullVal(cty.String),
			},
			{
				`ephemeralasnull("not ephemeral")`,
				cty.StringVal("not ephemeral"),
			},
		},

		"file": {
			{
				`file("hello.txt")`,
				cty.StringVal("hello!"),
			},
		},

		"fileexists": {
			{
				`fileexists("hello.txt")`,
				cty.BoolVal(true),
			},
		},

		"fileset": {
			{
				`fileset(".", "*/hello.*")`,
				cty.SetVal([]cty.Value{
					cty.StringVal("subdirectory/hello.tmpl"),
					cty.StringVal("subdirectory/hello.txt"),
				}),
			},
			{
				`fileset(".", "subdirectory/hello.*")`,
				cty.SetVal([]cty.Value{
					cty.StringVal("subdirectory/hello.tmpl"),
					cty.StringVal("subdirectory/hello.txt"),
				}),
			},
			{
				`fileset(".", "hello.*")`,
				cty.SetVal([]cty.Value{
					cty.StringVal("hello.tmpl"),
					cty.StringVal("hello.txt"),
				}),
			},
			{
				`fileset("subdirectory", "hello.*")`,
				cty.SetVal([]cty.Value{
					cty.StringVal("hello.tmpl"),
					cty.StringVal("hello.txt"),
				}),
			},
		},

		"filebase64": {
			{
				`filebase64("hello.txt")`,
				cty.StringVal("aGVsbG8h"),
			},
		},

		"filebase64sha256": {
			{
				`filebase64sha256("hello.txt")`,
				cty.StringVal("zgYJL7lI2f+sfRo3bkBLJrdXW8wR7gWkYV/vT+w6MIs="),
			},
		},

		"filebase64sha512": {
			{
				`filebase64sha512("hello.txt")`,
				cty.StringVal("xvgdsOn4IGyXHJ5YJuO6gj/7saOpAPgEdlKov3jqmP38dFhVo4U6Y1Z1RY620arxIJ6I6tLRkjgrXEy91oUOAg=="),
			},
		},

		"filemd5": {
			{
				`filemd5("hello.txt")`,
				cty.StringVal("5a8dd3ad0756a93ded72b823b19dd877"),
			},
		},

		"filesha1": {
			{
				`filesha1("hello.txt")`,
				cty.StringVal("8f7d88e901a5ad3a05d8cc0de93313fd76028f8c"),
			},
		},

		"filesha256": {
			{
				`filesha256("hello.txt")`,
				cty.StringVal("ce06092fb948d9ffac7d1a376e404b26b7575bcc11ee05a4615fef4fec3a308b"),
			},
		},

		"filesha512": {
			{
				`filesha512("hello.txt")`,
				cty.StringVal("c6f81db0e9f8206c971c9e5826e3ba823ffbb1a3a900f8047652a8bf78ea98fdfc745855a3853a635675458eb6d1aaf1209e88ead2d192382b5c4cbdd6850e02"),
			},
		},

		"flatten": {
			{
				`flatten([["a", "b"], ["c", "d"]])`,
				cty.TupleVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
					cty.StringVal("c"),
					cty.StringVal("d"),
				}),
			},
		},

		"floor": {
			{
				`floor(-1.8)`,
				cty.NumberFloatVal(-2),
			},
		},

		"format": {
			{
				`format("Hello, %s!", "Ander")`,
				cty.StringVal("Hello, Ander!"),
			},
		},

		"formatlist": {
			{
				`formatlist("Hello, %s!", ["Valentina", "Ander", "Olivia", "Sam"])`,
				cty.ListVal([]cty.Value{
					cty.StringVal("Hello, Valentina!"),
					cty.StringVal("Hello, Ander!"),
					cty.StringVal("Hello, Olivia!"),
					cty.StringVal("Hello, Sam!"),
				}),
			},
		},

		"formatdate": {
			{
				`formatdate("DD MMM YYYY hh:mm ZZZ", "2018-01-04T23:12:01Z")`,
				cty.StringVal("04 Jan 2018 23:12 UTC"),
			},
		},

		"indent": {
			{
				fmt.Sprintf("indent(4, %#v)", Poem),
				cty.StringVal("Fleas:\n    Adam\n    Had'em\n    \n    E.E. Cummings"),
			},
		},

		"index": {
			{
				`index(["a", "b", "c"], "a")`,
				cty.NumberIntVal(0),
			},
		},

		"issensitive": {
			{
				`issensitive(1)`,
				cty.False,
			},
		},

		"join": {
			{
				`join(" ", ["Hello", "World"])`,
				cty.StringVal("Hello World"),
			},
		},

		"jsondecode": {
			{
				`jsondecode("{\"hello\": \"world\"}")`,
				cty.ObjectVal(map[string]cty.Value{
					"hello": cty.StringVal("world"),
				}),
			},
		},

		"jsonencode": {
			{
				`jsonencode({"hello"="world"})`,
				cty.StringVal("{\"hello\":\"world\"}"),
			},
			// We are intentionally choosing to escape <, >, and & characters
			// to preserve backwards compatibility with Terraform 0.11
			{
				`jsonencode({"hello"="<cats & kittens>"})`,
				cty.StringVal("{\"hello\":\"\\u003ccats \\u0026 kittens\\u003e\"}"),
			},
		},

		"keys": {
			{
				`keys({"hello"=1, "goodbye"=42})`,
				cty.TupleVal([]cty.Value{
					cty.StringVal("goodbye"),
					cty.StringVal("hello"),
				}),
			},
		},

		"length": {
			{
				`length(["the", "quick", "brown", "bear"])`,
				cty.NumberIntVal(4),
			},
		},

		"list": {
			// There are intentionally no test cases for "list" because
			// it is a stub that always returns an error.
		},

		"log": {
			{
				`log(1, 10)`,
				cty.NumberFloatVal(0),
			},
		},

		"lookup": {
			{
				`lookup({hello=1, goodbye=42}, "goodbye")`,
				cty.NumberIntVal(42),
			},
		},

		"lower": {
			{
				`lower("HELLO")`,
				cty.StringVal("hello"),
			},
		},

		"map": {
			// There are intentionally no test cases for "map" because
			// it is a stub that always returns an error.
		},

		"matchkeys": {
			{
				`matchkeys(["a", "b", "c"], ["ref1", "ref2", "ref3"], ["ref1"])`,
				cty.ListVal([]cty.Value{
					cty.StringVal("a"),
				}),
			},
			{ // mixing types in searchset
				`matchkeys(["a", "b", "c"], [1, 2, 3], [1, "3"])`,
				cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("c"),
				}),
			},
		},

		"max": {
			{
				`max(12, 54, 3)`,
				cty.NumberIntVal(54),
			},
		},

		"md5": {
			{
				`md5("tada")`,
				cty.StringVal("ce47d07243bb6eaf5e1322c81baf9bbf"),
			},
		},

		"merge": {
			{
				`merge({"a"="b"}, {"c"="d"})`,
				cty.ObjectVal(map[string]cty.Value{
					"a": cty.StringVal("b"),
					"c": cty.StringVal("d"),
				}),
			},
		},

		"min": {
			{
				`min(12, 54, 3)`,
				cty.NumberIntVal(3),
			},
		},

		"nonsensitive": {
			{
				// Due to how this test is set up we have no way to get
				// a sensitive value other than to generate one with
				// another function, so this is a bit odd but does still
				// meet the goal of verifying that the "nonsensitive"
				// function is correctly registered.
				`nonsensitive(sensitive(1))`,
				cty.NumberIntVal(1),
			},
		},

		"one": {
			{
				`one([])`,
				cty.NullVal(cty.DynamicPseudoType),
			},
			{
				`one([true])`,
				cty.True,
			},
		},

		"parseint": {
			{
				`parseint("100", 10)`,
				cty.NumberIntVal(100),
			},
		},

		"pathexpand": {
			{
				`pathexpand("~/test-file")`,
				cty.StringVal(filepath.Join(homePath, "test-file")),
			},
		},

		"plantimestamp": {
			{
				`plantimestamp()`,
				cty.StringVal("2004-04-25T15:00:00Z"),
			},
		},

		"pow": {
			{
				`pow(1,0)`,
				cty.NumberFloatVal(1),
			},
		},

		"range": {
			{
				`range(3)`,
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(0),
					cty.NumberIntVal(1),
					cty.NumberIntVal(2),
				}),
			},
			{
				`range(1, 4)`,
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
					cty.NumberIntVal(2),
					cty.NumberIntVal(3),
				}),
			},
			{
				`range(1, 8, 2)`,
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
					cty.NumberIntVal(3),
					cty.NumberIntVal(5),
					cty.NumberIntVal(7),
				}),
			},
		},

		"regex": {
			{
				`regex("(\\d+)([a-z]+)", "aaa111bbb222")`,
				cty.TupleVal([]cty.Value{cty.StringVal("111"), cty.StringVal("bbb")}),
			},
		},

		"regexall": {
			{
				`regexall("(\\d+)([a-z]+)", "...111aaa222bbb...")`,
				cty.ListVal([]cty.Value{
					cty.TupleVal([]cty.Value{cty.StringVal("111"), cty.StringVal("aaa")}),
					cty.TupleVal([]cty.Value{cty.StringVal("222"), cty.StringVal("bbb")}),
				}),
			},
		},

		"replace": {
			{
				`replace("hello", "hel", "bel")`,
				cty.StringVal("bello"),
			},
		},

		"reverse": {
			{
				`reverse(["a", true, 0])`,
				cty.TupleVal([]cty.Value{cty.Zero, cty.True, cty.StringVal("a")}),
			},
		},

		"rsadecrypt": {
			{
				fmt.Sprintf("rsadecrypt(%#v, %#v)", CipherBase64, PrivateKey),
				cty.StringVal("message"),
			},
		},

		"sensitive": {
			{
				`sensitive(1)`,
				cty.NumberIntVal(1).Mark(marks.Sensitive),
			},
		},

		"setintersection": {
			{
				`setintersection(["a", "b"], ["b", "c"], ["b", "d"])`,
				cty.SetVal([]cty.Value{
					cty.StringVal("b"),
				}),
			},
		},

		"setproduct": {
			{
				`setproduct(["development", "staging", "production"], ["app1", "app2"])`,
				cty.ListVal([]cty.Value{
					cty.TupleVal([]cty.Value{cty.StringVal("development"), cty.StringVal("app1")}),
					cty.TupleVal([]cty.Value{cty.StringVal("development"), cty.StringVal("app2")}),
					cty.TupleVal([]cty.Value{cty.StringVal("staging"), cty.StringVal("app1")}),
					cty.TupleVal([]cty.Value{cty.StringVal("staging"), cty.StringVal("app2")}),
					cty.TupleVal([]cty.Value{cty.StringVal("production"), cty.StringVal("app1")}),
					cty.TupleVal([]cty.Value{cty.StringVal("production"), cty.StringVal("app2")}),
				}),
			},
		},

		"setsubtract": {
			{
				`setsubtract(["a", "b", "c"], ["a", "c"])`,
				cty.SetVal([]cty.Value{
					cty.StringVal("b"),
				}),
			},
		},

		"setunion": {
			{
				`setunion(["a", "b"], ["b", "c"], ["d"])`,
				cty.SetVal([]cty.Value{
					cty.StringVal("d"),
					cty.StringVal("b"),
					cty.StringVal("a"),
					cty.StringVal("c"),
				}),
			},
		},

		"sha1": {
			{
				`sha1("test")`,
				cty.StringVal("a94a8fe5ccb19ba61c4c0873d391e987982fbbd3"),
			},
		},

		"sha256": {
			{
				`sha256("test")`,
				cty.StringVal("9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"),
			},
		},

		"sha512": {
			{
				`sha512("test")`,
				cty.StringVal("ee26b0dd4af7e749aa1a8ee3c10ae9923f618980772e473f8819a5d4940e0db27ac185f8a0e1d5f84f88bc887fd67b143732c304cc5fa9ad8e6f57f50028a8ff"),
			},
		},

		"signum": {
			{
				`signum(12)`,
				cty.NumberFloatVal(1),
			},
		},

		"slice": {
			{
				// force a list type here for testing
				`slice(tolist(["a", "b", "c", "d"]), 1, 3)`,
				cty.ListVal([]cty.Value{
					cty.StringVal("b"), cty.StringVal("c"),
				}),
			},
			{
				`slice(["a", "b", 3, 4], 1, 3)`,
				cty.TupleVal([]cty.Value{
					cty.StringVal("b"), cty.NumberIntVal(3),
				}),
			},
		},

		"sort": {
			{
				`sort(["banana", "apple"])`,
				cty.ListVal([]cty.Value{
					cty.StringVal("apple"),
					cty.StringVal("banana"),
				}),
			},
		},

		"split": {
			{
				`split(" ", "Hello World")`,
				cty.ListVal([]cty.Value{
					cty.StringVal("Hello"),
					cty.StringVal("World"),
				}),
			},
		},

		"startswith": {
			{
				`startswith("hello world", "hello")`,
				cty.True,
			},
			{
				`startswith("hello world", "world")`,
				cty.False,
			},
			{
				`startswith("hello world", "")`,
				cty.True,
				// Completely empty prefix value  ( "" )
				// will always evaluate to true for all strings.
			},
			{
				`startswith("hello world", " ")`,
				cty.False,
			},
			{
				`startswith("", "")`,
				cty.True,
			},
			{
				`startswith("", " ")`,
				cty.False,
			},
			{
				`startswith(" ", "")`,
				cty.True,
			},
			{
				`startswith("", "hello")`,
				cty.False,
			},
			{
				`startswith(" ", "hello")`,
				cty.False,
			},
		},

		"strcontains": {
			{
				`strcontains("hello", "llo")`,
				cty.BoolVal(true),
			},
			{
				`strcontains("hello", "a")`,
				cty.BoolVal(false),
			},
		},

		"strrev": {
			{
				`strrev("hello world")`,
				cty.StringVal("dlrow olleh"),
			},
		},

		"substr": {
			{
				`substr("hello world", 1, 4)`,
				cty.StringVal("ello"),
			},
		},

		"sum": {
			{
				`sum([2340.5,10,3])`,
				cty.NumberFloatVal(2353.5),
			},
		},

		"textdecodebase64": {
			{
				`textdecodebase64("dABlAHMAdAA=", "UTF-16LE")`,
				cty.StringVal("test"),
			},
		},

		"textencodebase64": {
			{
				`textencodebase64("test", "UTF-16LE")`,
				cty.StringVal("dABlAHMAdAA="),
			},
		},

		"templatefile": {
			{
				`templatefile("hello.tmpl", {name = "Jodie"})`,
				cty.StringVal("Hello, Jodie!"),
			},
			{
				`core::templatefile("hello.tmpl", {name = "Namespaced Jodie"})`,
				cty.StringVal("Hello, Namespaced Jodie!"),
			},
		},

		"templatestring": {
			{
				`templatestring(local.greeting_template, {
  name = "Arthur"
})`,
				cty.StringVal("Hello, Arthur!"),
			},
			{
				`core::templatestring(local.greeting_template, {
  name = "Namespaced Arthur"
})`,
				cty.StringVal("Hello, Namespaced Arthur!"),
			},
		},

		"timeadd": {
			{
				`timeadd("2017-11-22T00:00:00Z", "1s")`,
				cty.StringVal("2017-11-22T00:00:01Z"),
			},
		},

		"timecmp": {
			{
				`timecmp("2017-11-22T00:00:00Z", "2017-11-22T00:00:00Z")`,
				cty.Zero,
			},
		},

		"title": {
			{
				`title("hello")`,
				cty.StringVal("Hello"),
			},
		},

		"tobool": {
			{
				`tobool("false")`,
				cty.False,
			},
		},

		"tolist": {
			{
				`tolist(["a", "b", "c"])`,
				cty.ListVal([]cty.Value{
					cty.StringVal("a"), cty.StringVal("b"), cty.StringVal("c"),
				}),
			},
		},

		"tomap": {
			{
				`tomap({"a" = 1, "b" = 2})`,
				cty.MapVal(map[string]cty.Value{
					"a": cty.NumberIntVal(1),
					"b": cty.NumberIntVal(2),
				}),
			},
		},

		"tonumber": {
			{
				`tonumber("42")`,
				cty.NumberIntVal(42),
			},
		},

		"toset": {
			{
				`toset(["a", "b", "c"])`,
				cty.SetVal([]cty.Value{
					cty.StringVal("a"), cty.StringVal("b"), cty.StringVal("c"),
				}),
			},
		},

		"tostring": {
			{
				`tostring("a")`,
				cty.StringVal("a"),
			},
		},

		"transpose": {
			{
				`transpose({"a" = ["1", "2"], "b" = ["2", "3"]})`,
				cty.MapVal(map[string]cty.Value{
					"1": cty.ListVal([]cty.Value{cty.StringVal("a")}),
					"2": cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}),
					"3": cty.ListVal([]cty.Value{cty.StringVal("b")}),
				}),
			},
		},

		"trim": {
			{
				`trim("?!hello?!", "!?")`,
				cty.StringVal("hello"),
			},
		},

		"trimprefix": {
			{
				`trimprefix("helloworld", "hello")`,
				cty.StringVal("world"),
			},
		},

		"trimspace": {
			{
				`trimspace(" hello ")`,
				cty.StringVal("hello"),
			},
		},

		"trimsuffix": {
			{
				`trimsuffix("helloworld", "world")`,
				cty.StringVal("hello"),
			},
		},

		"try": {
			{
				// Note: "try" only works with expressions that pass static
				// validation, because it only gets an opportunity to run in
				// that case. The following "works" (captures the error) because
				// Terraform understands it as a reference to an attribute
				// that does not exist during dynamic evaluation.
				//
				// "try" doesn't work with references that could never possibly
				// be valid and are thus caught during static validation, such
				// as an expression like "foo" alone which would be understood
				// as an invalid resource reference. That's okay because this
				// function exists primarily to ease access to dynamically-typed
				// structures that Terraform can't statically validate by
				// definition.
				`try({}.baz, "fallback")`,
				cty.StringVal("fallback"),
			},
			{
				`try("fallback")`,
				cty.StringVal("fallback"),
			},
		},

		"upper": {
			{
				`upper("hello")`,
				cty.StringVal("HELLO"),
			},
			{
				`core::upper("hello")`,
				cty.StringVal("HELLO"),
			},
		},

		"urlencode": {
			{
				`urlencode("foo:bar@localhost?foo=bar&bar=baz")`,
				cty.StringVal("foo%3Abar%40localhost%3Ffoo%3Dbar%26bar%3Dbaz"),
			},
		},

		"uuidv5": {
			{
				`uuidv5("dns", "tada")`,
				cty.StringVal("faa898db-9b9d-5b75-86a9-149e7bb8e3b8"),
			},
			{
				`uuidv5("url", "tada")`,
				cty.StringVal("2c1ff6b4-211f-577e-94de-d978b0caa16e"),
			},
			{
				`uuidv5("oid", "tada")`,
				cty.StringVal("61eeea26-5176-5288-87fc-232d6ed30d2f"),
			},
			{
				`uuidv5("x500", "tada")`,
				cty.StringVal("7e12415e-f7c9-57c3-9e43-52dc9950d264"),
			},
			{
				`uuidv5("6ba7b810-9dad-11d1-80b4-00c04fd430c8", "tada")`,
				cty.StringVal("faa898db-9b9d-5b75-86a9-149e7bb8e3b8"),
			},
		},

		"values": {
			{
				`values({"hello"="world", "what's"="up"})`,
				cty.TupleVal([]cty.Value{
					cty.StringVal("world"),
					cty.StringVal("up"),
				}),
			},
		},

		"yamldecode": {
			{
				`yamldecode("true")`,
				cty.True,
			},
			{
				`yamldecode("key: 0ba")`,
				cty.ObjectVal(map[string]cty.Value{
					"key": cty.StringVal("0ba"),
				}),
			},
			{
				`yamldecode("~")`,
				cty.NullVal(cty.DynamicPseudoType),
			},
		},

		"yamlencode": {
			{
				`yamlencode(["foo", "bar", true])`,
				cty.StringVal("- \"foo\"\n- \"bar\"\n- true\n"),
			},
			{
				`yamlencode({a = "b", c = "d"})`,
				cty.StringVal("\"a\": \"b\"\n\"c\": \"d\"\n"),
			},
			{
				`yamlencode(true)`,
				// the ... here is an "end of document" marker, produced for implied primitive types only
				cty.StringVal("true\n...\n"),
			},
		},

		"zipmap": {
			{
				`zipmap(["hello", "bar"], ["world", "baz"])`,
				cty.ObjectVal(map[string]cty.Value{
					"hello": cty.StringVal("world"),
					"bar":   cty.StringVal("baz"),
				}),
			},
		},
		// External function dispatching tests. These ones are only here to
		// test that dispatching to externally-declared functions works
		// _at all_, using just some placeholder functions declared in the
		// test code below.
		"provider::foo::upper": {
			{
				`provider::foo::upper("hello")`,
				cty.StringVal("HELLO"),
			},
		},
	}

	experimentalFuncs := map[string]experiments.Experiment{}

	// We'll also register a few "external functions" so that we can
	// verify that registering these works. The functions actually
	// available in a real module will be determined dynamically by
	// Terraform core based on declarations in that module, so here
	// we're just aiming to test whether dispatching to these works
	// at all, not to test that any particular functions work.
	externalFuncs := ExternalFuncs{
		Provider: map[string]map[string]function.Function{
			"foo": {
				"upper": stdlib.UpperFunc,
			},
		},
	}

	t.Run("all functions are tested", func(t *testing.T) {
		data := &dataForTests{} // no variables available; we only need literals here
		scope := &Scope{
			Data:          data,
			BaseDir:       "./testdata/functions-test", // for the functions that read from the filesystem
			ExternalFuncs: externalFuncs,
		}

		// Check that there is at least one test case for each function, omitting
		// those functions that do not return consistent values
		allFunctions := scope.Functions()

		// TODO: we can test the impure functions partially by configuring the scope
		// with PureOnly: true and then verify that they return unknown values of a
		// suitable type.
		for _, impureFunc := range impureFunctions {
			delete(allFunctions, impureFunc)
		}
		for f := range scope.Functions() {
			if strings.Contains(f, "::") {
				// Only non-namespaced functions are absolutely required to
				// have at least one test. (Others _may_ have tests.)
				continue
			}
			if _, ok := tests[f]; !ok {
				t.Errorf("Missing test for function %s\n", f)
			}
		}
	})

	for funcName, funcTests := range tests {
		t.Run(funcName, func(t *testing.T) {

			// prepareScope starts as a no-op, but if a function is marked as
			// experimental in our experimentalFuncs table above then we'll
			// reassign this to be a function that activates the appropriate
			// experiment.
			prepareScope := func(t *testing.T, scope *Scope) {}

			if experiment, isExperimental := experimentalFuncs[funcName]; isExperimental {
				// First, we'll run all of the tests without the experiment
				// enabled to see that they do actually fail in that case.
				for _, test := range funcTests {
					testName := fmt.Sprintf("experimental(%s)", test.src)
					t.Run(testName, func(t *testing.T) {
						data := &dataForTests{} // no variables available; we only need literals here
						scope := &Scope{
							Data:          data,
							BaseDir:       "./testdata/functions-test", // for the functions that read from the filesystem
							ExternalFuncs: externalFuncs,
						}

						expr, parseDiags := hclsyntax.ParseExpression([]byte(test.src), "test.hcl", hcl.Pos{Line: 1, Column: 1})
						if parseDiags.HasErrors() {
							for _, diag := range parseDiags {
								t.Error(diag.Error())
							}
							return
						}

						_, diags := scope.EvalExpr(expr, cty.DynamicPseudoType)
						if !diags.HasErrors() {
							t.Errorf("experimental function %q succeeded without its experiment %s enabled\nexpr: %s", funcName, experiment.Keyword(), test.src)
						}
					})
				}

				// Now make the experiment active in the scope so that the
				// function will actually work when we test it below.
				prepareScope = func(t *testing.T, scope *Scope) {
					t.Helper()
					t.Logf("activating experiment %s to test %q", experiment.Keyword(), funcName)
					experimentsSet := experiments.NewSet()
					experimentsSet.Add(experiment)
					scope.SetActiveExperiments(experimentsSet)
				}
			}

			for _, test := range funcTests {
				t.Run(test.src, func(t *testing.T) {
					data := &dataForTests{
						LocalValues: map[string]cty.Value{
							"greeting_template": cty.StringVal("Hello, ${name}!"),
							"ephemeral":         cty.StringVal("ephemeral").Mark(marks.Ephemeral),
						},
					}
					scope := &Scope{
						Data:          data,
						ParseRef:      addrs.ParseRef,
						BaseDir:       "./testdata/functions-test", // for the functions that read from the filesystem
						PlanTimestamp: time.Date(2004, 04, 25, 15, 00, 00, 000, time.UTC),
						ExternalFuncs: externalFuncs,
					}
					prepareScope(t, scope)

					expr, parseDiags := hclsyntax.ParseExpression([]byte(test.src), "test.hcl", hcl.Pos{Line: 1, Column: 1})
					if parseDiags.HasErrors() {
						for _, diag := range parseDiags {
							t.Error(diag.Error())
						}
						return
					}

					got, diags := scope.EvalExpr(expr, cty.DynamicPseudoType)
					if diags.HasErrors() {
						for _, diag := range diags {
							t.Errorf("%s: %s", diag.Description().Summary, diag.Description().Detail)
						}
						return
					}

					if !test.want.RawEquals(got) {
						t.Errorf("wrong result\nexpr: %s\ngot:  %#v\nwant: %#v", test.src, got, test.want)
					}
				})
			}
		})
	}
}

func TestPlanTimeStampUnknown(t *testing.T) {
	// plantimestamp should return an unknown if there is no timestamp, which
	// happens during validation
	expr, parseDiags := hclsyntax.ParseExpression([]byte("plantimestamp()"), "test.hcl", hcl.Pos{Line: 1, Column: 1})
	if parseDiags.HasErrors() {
		t.Fatal(parseDiags)
	}

	scope := &Scope{}
	got, diags := scope.EvalExpr(expr, cty.DynamicPseudoType)
	if diags.HasErrors() {
		t.Fatal(diags.Err())

	}

	if got.IsKnown() {
		t.Fatalf("plantimestamp() should be unknown, got %#v\n", got)
	}
}

const (
	CipherBase64 = "eczGaDhXDbOFRZGhjx2etVzWbRqWDlmq0bvNt284JHVbwCgObiuyX9uV0LSAMY707IEgMkExJqXmsB4OWKxvB7epRB9G/3+F+pcrQpODlDuL9oDUAsa65zEpYF0Wbn7Oh7nrMQncyUPpyr9WUlALl0gRWytOA23S+y5joa4M34KFpawFgoqTu/2EEH4Xl1zo+0fy73fEto+nfkUY+meuyGZ1nUx/+DljP7ZqxHBFSlLODmtuTMdswUbHbXbWneW51D7Jm7xB8nSdiA2JQNK5+Sg5x8aNfgvFTt/m2w2+qpsyFa5Wjeu6fZmXSl840CA07aXbk9vN4I81WmJyblD/ZA=="
	PrivateKey   = `
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAgUElV5mwqkloIrM8ZNZ72gSCcnSJt7+/Usa5G+D15YQUAdf9
c1zEekTfHgDP+04nw/uFNFaE5v1RbHaPxhZYVg5ZErNCa/hzn+x10xzcepeS3KPV
Xcxae4MR0BEegvqZqJzN9loXsNL/c3H/B+2Gle3hTxjlWFb3F5qLgR+4Mf4ruhER
1v6eHQa/nchi03MBpT4UeJ7MrL92hTJYLdpSyCqmr8yjxkKJDVC2uRrr+sTSxfh7
r6v24u/vp/QTmBIAlNPgadVAZw17iNNb7vjV7Gwl/5gHXonCUKURaV++dBNLrHIZ
pqcAM8wHRph8mD1EfL9hsz77pHewxolBATV+7QIDAQABAoIBAC1rK+kFW3vrAYm3
+8/fQnQQw5nec4o6+crng6JVQXLeH32qXShNf8kLLG/Jj0vaYcTPPDZw9JCKkTMQ
0mKj9XR/5DLbBMsV6eNXXuvJJ3x4iKW5eD9WkLD4FKlNarBRyO7j8sfPTqXW7uat
NxWdFH7YsSRvNh/9pyQHLWA5OituidMrYbc3EUx8B1GPNyJ9W8Q8znNYLfwYOjU4
Wv1SLE6qGQQH9Q0WzA2WUf8jklCYyMYTIywAjGb8kbAJlKhmj2t2Igjmqtwt1PYc
pGlqbtQBDUiWXt5S4YX/1maIQ/49yeNUajjpbJiH3DbhJbHwFTzP3pZ9P9GHOzlG
kYR+wSECgYEAw/Xida8kSv8n86V3qSY/I+fYQ5V+jDtXIE+JhRnS8xzbOzz3v0WS
Oo5H+o4nJx5eL3Ghb3Gcm0Jn46dHrxinHbm+3RjXv/X6tlbxIYjRSQfHOTSMCTvd
qcliF5vC6RCLXuc7R+IWR1Ky6eDEZGtrvt3DyeYABsp9fRUFR/6NluUCgYEAqNsw
1aSl7WJa27F0DoJdlU9LWerpXcazlJcIdOz/S9QDmSK3RDQTdqfTxRmrxiYI9LEs
mkOkvzlnnOBMpnZ3ZOU5qIRfprecRIi37KDAOHWGnlC0EWGgl46YLb7/jXiWf0AG
Y+DfJJNd9i6TbIDWu8254/erAS6bKMhW/3q7f2kCgYAZ7Id/BiKJAWRpqTRBXlvw
BhXoKvjI2HjYP21z/EyZ+PFPzur/lNaZhIUlMnUfibbwE9pFggQzzf8scM7c7Sf+
mLoVSdoQ/Rujz7CqvQzi2nKSsM7t0curUIb3lJWee5/UeEaxZcmIufoNUrzohAWH
BJOIPDM4ssUTLRq7wYM9uQKBgHCBau5OP8gE6mjKuXsZXWUoahpFLKwwwmJUp2vQ
pOFPJ/6WZOlqkTVT6QPAcPUbTohKrF80hsZqZyDdSfT3peFx4ZLocBrS56m6NmHR
UYHMvJ8rQm76T1fryHVidz85g3zRmfBeWg8yqT5oFg4LYgfLsPm1gRjOhs8LfPvI
OLlRAoGBAIZ5Uv4Z3s8O7WKXXUe/lq6j7vfiVkR1NW/Z/WLKXZpnmvJ7FgxN4e56
RXT7GwNQHIY8eDjDnsHxzrxd+raOxOZeKcMHj3XyjCX3NHfTscnsBPAGYpY/Wxzh
T8UYnFu6RzkixElTf2rseEav7rkdKkI3LAeIZy7B0HulKKsmqVQ7
-----END RSA PRIVATE KEY-----
`
	Poem = `Fleas:
Adam
Had'em

E.E. Cummings`
)
