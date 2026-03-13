// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package ast

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// makeTraversal builds an hcl.Traversal from a list of names.
// The first name becomes a TraverseRoot, the rest become TraverseAttr steps.
func makeTraversal(names ...string) hcl.Traversal {
	if len(names) == 0 {
		return nil
	}
	t := hcl.Traversal{hcl.TraverseRoot{Name: names[0]}}
	for _, n := range names[1:] {
		t = append(t, hcl.TraverseAttr{Name: n})
	}
	return t
}

func TestParseFile_roundtrip(t *testing.T) {
	tests := map[string]string{
		"empty file": ``,

		"single block": `resource "aws_instance" "example" {
  ami           = "abc-123"
  instance_type = "t2.micro"
}
`,

		"preserves comments": `# This is a top-level comment

resource "aws_instance" "example" {
  # This comment should be preserved
  ami           = "abc-123"
  instance_type = "t2.micro" // inline comment
}
`,

		"nested blocks": `resource "aws_instance" "example" {
  ami           = "abc-123"
  instance_type = "t2.micro"

  ebs_block_device {
    volume_size = 20
    volume_type = "gp2"
  }

  tags = {
    Name = "example"
  }
}
`,

		"multiple block types": `
terraform {
  required_version = "1.15"
}

variable "hello" {
  type = string
}

locals {
  greeting = "Hello, world!"
}

output "world" {
  value = "Hello, world!"
}
`,

		"modules": `
module "a" {
  source   = "./modules/module_a"
  greeting = "Hello from root module A(${var.count})"
  count    = 2
}

module "b" {
  source = "./modules/module_b"
  name   = "ModuleB"
}
`,

		"preserves formatting and comments in output": `
variable "greeting" {
  type = string
}

variable "count" {
  type    = number
  default = 1
}

output "message" {
  // This weird syntax and the comment are left here intentionally to
  // make sure that we preserve formatting and comments when writing back the AST.
  value = "${var.greeting}"
}
`,

		"complex expressions": `
variable "enable" {
  type    = bool
  default = true
}

variable "names" {
  type    = list(string)
  default = ["alice", "bob", "", "eve"]
}

locals {
  # simple numeric sequences and transformations
  numbers = [0, 1, 2, 3]
  squares = [for n in local.numbers : n * n]
  indexed = { for idx, val in local.numbers : idx => val }

  # combine lists and pick elements using modulo arithmetic
  paired = [
    for i in local.numbers : {
      index = i
      name  = var.names[i % length(var.names)]
    }
  ]

  # filter empty strings out of a list
  filtered_names = [for n in var.names : n if length(n) > 0]

  # create a map from names to a 1-based index
  mapped = zipmap(local.filtered_names, [for i in range(length(local.filtered_names)) : i + 1])

  # nested comprehensions producing a cartesian structure, then flattening
  cartesian = [
    for a in [1, 2] : [
      for b in [10, 20] : {
        a   = a
        b   = b
        sum = a + b
      }
    ]
  ]
  flattened = flatten(local.cartesian)

  # format the flattened results into a single string
  combined = join(", ", [for p in local.flattened : format("%d+%d=%d", p.a, p.b, p.sum)])

  # deeper nested objects and list comprehensions with function calls
  nested = {
    level1 = {
      level2       = upper(join("-", [for ch in ["x", "y", "z"] : ch]))
      dynamic_list = [for i in range(3) : format("item-%d", i)]
    }
  }

  # use coalesce/try/element to provide a safe fallback when indexing out of bounds
  fallback = coalesce(try(element(local.filtered_names, 10), null), "default-name")

  # nested conditional expressions
  conditional = var.enable ? (length(local.filtered_names) > 1 ? "many" : "one") : "none"

  # regex and merging maps
  regex_matches = regexall("[aeiou]", "terraform")
  combined_map  = merge({ a = 1 }, { b = 2 }, { c = length(local.filtered_names) })

  # lookup with try for safety
  complex_lookup = try(lookup(local.mapped, "alice", 42), 0)
}

output "complex_summary" {
  value = {
    squares        = local.squares
    paired         = local.paired
    filtered       = local.filtered_names
    mapped         = local.mapped
    combined       = local.combined
    nested         = local.nested
    fallback       = local.fallback
    conditional    = local.conditional
    regex_matches  = local.regex_matches
    combined_map   = local.combined_map
    complex_lookup = local.complex_lookup
  }
}
`,

		"resources and data sources": `
data "http" "example" {
  url = "https://www.example.com"
}

data "template_file" "greeting" {
  template = "Hello, ${name}!"
  vars = {
    name = "Terraform"
  }
}

resource "random_pet" "example" {
  length = 2
}

output "http_status" {
  value = data.http.example.status_code
}

output "template_rendered" {
  value = data.template_file.greeting.rendered
}
`,
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := ParseFile([]byte(input), "main.tf", nil)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			got := string(f.Bytes())
			if got != input {
				t.Errorf("roundtrip mismatch\n--- want ---\n%s\n--- got ---\n%s", input, got)
			}
		})
	}
}

func TestFile_FindBlocks(t *testing.T) {
	input := `resource "aws_s3_bucket" "a" {
  bucket = "bucket-a"
}

resource "aws_s3_bucket" "b" {
  bucket = "bucket-b"
}

resource "aws_instance" "web" {
  ami = "abc-123"
}

data "aws_ami" "latest" {
  most_recent = true
}
`

	f, err := ParseFile([]byte(input), "main.tf", nil)
	if err != nil {
		t.Fatalf("unexpected parse error: %s", err)
	}

	tests := map[string]struct {
		blockType    string
		resourceType string
		wantCount    int
		wantLabels   [][]string
	}{
		"find all resources of a type": {
			blockType:    "resource",
			resourceType: "aws_s3_bucket",
			wantCount:    2,
			wantLabels: [][]string{
				{"aws_s3_bucket", "a"},
				{"aws_s3_bucket", "b"},
			},
		},
		"no matches": {
			blockType:    "resource",
			resourceType: "aws_lambda_function",
			wantCount:    0,
			wantLabels:   nil,
		},
		"find data sources": {
			blockType:    "data",
			resourceType: "aws_ami",
			wantCount:    1,
			wantLabels: [][]string{
				{"aws_ami", "latest"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			blocks := f.FindBlocks(tc.blockType, tc.resourceType)
			if len(blocks) != tc.wantCount {
				t.Fatalf("got %d blocks, want %d", len(blocks), tc.wantCount)
			}
			for i, b := range blocks {
				if tc.wantLabels != nil {
					gotLabels := b.Labels()
					if len(gotLabels) != len(tc.wantLabels[i]) {
						t.Errorf("block %d: got %d labels, want %d", i, len(gotLabels), len(tc.wantLabels[i]))
						continue
					}
					for j := range gotLabels {
						if gotLabels[j] != tc.wantLabels[i][j] {
							t.Errorf("block %d label %d: got %q, want %q", i, j, gotLabels[j], tc.wantLabels[i][j])
						}
					}
				}
				if b.Type() != tc.blockType {
					t.Errorf("block %d: Type() = %q, want %q", i, b.Type(), tc.blockType)
				}
			}
		})
	}
}

func TestBlock_BlockAtPath(t *testing.T) {
	input := `resource "aws_instance" "example" {
  ami           = "abc-123"
  instance_type = "t2.micro"

  network_interface {
    device_index = 0

    access_config {
      nat_ip = "1.2.3.4"
    }
  }

  ebs_block_device {
    volume_size = 20
  }
}
`

	f, err := ParseFile([]byte(input), "main.tf", nil)
	if err != nil {
		t.Fatalf("unexpected parse error: %s", err)
	}

	blocks := f.FindBlocks("resource", "aws_instance")
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	root := blocks[0]

	tests := map[string]struct {
		path     cty.Path
		wantNil  bool
		wantType string
	}{
		"direct child": {
			path:     cty.Path{cty.GetAttrStep{Name: "ebs_block_device"}},
			wantNil:  false,
			wantType: "ebs_block_device",
		},
		"nested two levels": {
			path:     cty.Path{cty.GetAttrStep{Name: "network_interface"}, cty.GetAttrStep{Name: "access_config"}},
			wantNil:  false,
			wantType: "access_config",
		},
		"missing path": {
			path:    cty.Path{cty.GetAttrStep{Name: "nonexistent"}},
			wantNil: true,
		},
		"partial path missing": {
			path:    cty.Path{cty.GetAttrStep{Name: "network_interface"}, cty.GetAttrStep{Name: "nonexistent"}},
			wantNil: true,
		},
		"single segment": {
			path:     cty.Path{cty.GetAttrStep{Name: "network_interface"}},
			wantNil:  false,
			wantType: "network_interface",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := root.BlockAtPath(tc.path)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got block of type %q", got.Type())
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil block, got nil")
			}
			if got.Type() != tc.wantType {
				t.Errorf("Type() = %q, want %q", got.Type(), tc.wantType)
			}
		})
	}
}

func TestBlock_RenameAttribute(t *testing.T) {
	tests := map[string]struct {
		input      string
		from       string
		to         string
		wantResult bool
		wantSubstr string
		wantAbsent string
	}{
		"simple rename": {
			input: `resource "aws_instance" "example" {
  ami           = "abc-123"
  instance_type = "t2.micro"
}
`,
			from:       "ami",
			to:         "image_id",
			wantResult: true,
			wantSubstr: "image_id",
			wantAbsent: "ami",
		},
		"attribute not found": {
			input: `resource "aws_instance" "example" {
  ami = "abc-123"
}
`,
			from:       "nonexistent",
			to:         "something",
			wantResult: false,
			wantSubstr: "ami",
		},
		"preserves comments around renamed attribute": {
			input: `resource "aws_instance" "example" {
  # This is the AMI to use
  ami = "abc-123" # inline comment
  instance_type = "t2.micro"
}
`,
			from:       "ami",
			to:         "image_id",
			wantResult: true,
			wantSubstr: "image_id",
			wantAbsent: `ami`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := ParseFile([]byte(tc.input), "main.tf", nil)
			if err != nil {
				t.Fatalf("unexpected parse error: %s", err)
			}

			blocks := f.FindBlocks("resource", "aws_instance")
			if len(blocks) != 1 {
				t.Fatalf("expected 1 block, got %d", len(blocks))
			}

			got := blocks[0].RenameAttribute(tc.from, tc.to)
			if got != tc.wantResult {
				t.Errorf("RenameAttribute() = %v, want %v", got, tc.wantResult)
			}

			output := string(f.Bytes())
			if !strings.Contains(output, tc.wantSubstr) {
				t.Errorf("output missing %q\n%s", tc.wantSubstr, output)
			}
			if tc.wantAbsent != "" && strings.Contains(output, tc.wantAbsent) {
				t.Errorf("output should not contain %q\n%s", tc.wantAbsent, output)
			}
		})
	}
}

func TestBlock_RemoveAttribute(t *testing.T) {
	tests := map[string]struct {
		input      string
		name       string
		wantResult bool
		wantAbsent string
	}{
		"remove existing": {
			input: `resource "aws_instance" "example" {
  ami           = "abc-123"
  instance_type = "t2.micro"
}
`,
			name:       "ami",
			wantResult: true,
			wantAbsent: "ami",
		},
		"not found returns false": {
			input: `resource "aws_instance" "example" {
  ami = "abc-123"
}
`,
			name:       "nonexistent",
			wantResult: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := ParseFile([]byte(tc.input), "main.tf", nil)
			if err != nil {
				t.Fatalf("unexpected parse error: %s", err)
			}

			blocks := f.FindBlocks("resource", "aws_instance")
			if len(blocks) != 1 {
				t.Fatalf("expected 1 block, got %d", len(blocks))
			}

			got := blocks[0].RemoveAttribute(tc.name)
			if got != tc.wantResult {
				t.Errorf("RemoveAttribute() = %v, want %v", got, tc.wantResult)
			}

			if tc.wantAbsent != "" {
				output := string(f.Bytes())
				if strings.Contains(output, tc.wantAbsent) {
					t.Errorf("output should not contain %q\n%s", tc.wantAbsent, output)
				}
			}
		})
	}
}

func TestBlock_SetAttributeValue(t *testing.T) {
	tests := map[string]struct {
		input      string
		attrName   string
		val        cty.Value
		wantSubstr string
	}{
		"change string value": {
			input: `resource "aws_instance" "example" {
  ami = "abc-123"
}
`,
			attrName:   "ami",
			val:        cty.StringVal("new-ami-456"),
			wantSubstr: `"new-ami-456"`,
		},
		"change number value": {
			input: `resource "aws_instance" "example" {
  count = 1
}
`,
			attrName:   "count",
			val:        cty.NumberIntVal(3),
			wantSubstr: "3",
		},
		"add new attribute": {
			input: `resource "aws_instance" "example" {
  ami = "abc-123"
}
`,
			attrName:   "instance_type",
			val:        cty.StringVal("t2.micro"),
			wantSubstr: `"t2.micro"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := ParseFile([]byte(tc.input), "main.tf", nil)
			if err != nil {
				t.Fatalf("unexpected parse error: %s", err)
			}

			blocks := f.FindBlocks("resource", "aws_instance")
			if len(blocks) != 1 {
				t.Fatalf("expected 1 block, got %d", len(blocks))
			}

			blocks[0].SetAttributeValue(tc.attrName, tc.val)

			output := string(f.Bytes())
			if !strings.Contains(output, tc.wantSubstr) {
				t.Errorf("output missing %q\n%s", tc.wantSubstr, output)
			}
		})
	}
}

func TestBlock_SetAttributeRaw(t *testing.T) {
	input := `resource "aws_instance" "example" {
  ami = "abc-123"
}
`
	f, err := ParseFile([]byte(input), "main.tf", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	blocks := f.FindBlocks("resource", "aws_instance")
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	tokens := hclwrite.TokensForTraversal(hcl.Traversal{
		hcl.TraverseRoot{Name: "var"},
		hcl.TraverseAttr{Name: "ami_id"},
	})
	blocks[0].SetAttributeRaw("ami", tokens)

	output := string(f.Bytes())
	if !strings.Contains(output, "var.ami_id") {
		t.Errorf("output missing var.ami_id\n%s", output)
	}
}

func TestBlock_SetLabels(t *testing.T) {
	input := `
/**
    * This is a comment that should be preserved when changing labels.
    * It is intentionally formatted in a weird way to make sure that we preserve formatting as well.
    */
resource "aws_instance" "old_name" { // yet another comment
  ami = "abc-123"                    // inline commment
}                                    // we have high quality, we comment a lot!
// and one to finish the file with a bang!

`
	f, err := ParseFile([]byte(input), "main.tf", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	blocks := f.FindBlocks("resource", "aws_instance")
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	blocks[0].SetLabels([]string{"aws_instance", "new_name"})

	output := string(f.Bytes())

	expected := `
/**
    * This is a comment that should be preserved when changing labels.
    * It is intentionally formatted in a weird way to make sure that we preserve formatting as well.
    */
resource "aws_instance" "new_name" { // yet another comment
  ami = "abc-123"                    // inline commment
}                                    // we have high quality, we comment a lot!
// and one to finish the file with a bang!

`

	if diff := cmp.Diff(expected, output); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

}

func TestBlock_RemoveNestedBlock(t *testing.T) {
	tests := map[string]struct {
		input      string
		blockType  string
		wantResult bool
		wantAbsent string
	}{
		"remove existing nested block": {
			input: `resource "aws_instance" "example" {
  ami           = "abc-123"
  instance_type = "t2.micro"

  ebs_block_device {
    volume_size = 20
    volume_type = "gp2"
  }
}
`,
			blockType:  "ebs_block_device",
			wantResult: true,
			wantAbsent: "ebs_block_device",
		},
		"not found returns false": {
			input: `resource "aws_instance" "example" {
  ami = "abc-123"
}
`,
			blockType:  "nonexistent",
			wantResult: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := ParseFile([]byte(tc.input), "main.tf", nil)
			if err != nil {
				t.Fatalf("unexpected parse error: %s", err)
			}

			blocks := f.FindBlocks("resource", "aws_instance")
			if len(blocks) != 1 {
				t.Fatalf("expected 1 block, got %d", len(blocks))
			}

			got := blocks[0].RemoveBlock(tc.blockType)
			if got != tc.wantResult {
				t.Errorf("RemoveNestedBlock() = %v, want %v", got, tc.wantResult)
			}

			if tc.wantAbsent != "" {
				output := string(f.Bytes())
				if strings.Contains(output, tc.wantAbsent) {
					t.Errorf("output should not contain %q\n%s", tc.wantAbsent, output)
				}
			}
		})
	}
}

func TestBlock_AddNestedBlock(t *testing.T) {
	tests := map[string]struct {
		input         string
		blockType     string
		attrName      string
		attrVal       cty.Value
		wantBlockType string
		wantSubstr    string
	}{
		"add new nested block with attribute": {
			input: `resource "aws_instance" "example" {
  ami = "abc-123"
}
`,
			blockType:     "ebs_block_device",
			attrName:      "volume_size",
			attrVal:       cty.NumberIntVal(50),
			wantBlockType: "ebs_block_device",
			wantSubstr:    "volume_size",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := ParseFile([]byte(tc.input), "main.tf", nil)
			if err != nil {
				t.Fatalf("unexpected parse error: %s", err)
			}

			blocks := f.FindBlocks("resource", "aws_instance")
			if len(blocks) != 1 {
				t.Fatalf("expected 1 block, got %d", len(blocks))
			}

			nb := blocks[0].AddBlock(tc.blockType)
			if nb == nil {
				t.Fatal("AddNestedBlock returned nil")
			}
			if nb.Type() != tc.wantBlockType {
				t.Errorf("Type() = %q, want %q", nb.Type(), tc.wantBlockType)
			}

			nb.SetAttributeValue(tc.attrName, tc.attrVal)

			output := string(f.Bytes())
			if !strings.Contains(output, tc.blockType) {
				t.Errorf("output missing block type %q\n%s", tc.blockType, output)
			}
			if !strings.Contains(output, tc.wantSubstr) {
				t.Errorf("output missing %q\n%s", tc.wantSubstr, output)
			}
		})
	}
}

func TestBlock_RenameReferencePrefix(t *testing.T) {
	tests := map[string]struct {
		input string
		old   hcl.Traversal
		new   hcl.Traversal
		want  string
	}{
		"rename resource reference in attribute": {
			input: `resource "aws_instance" "example" {
  subnet_id = aws_s3_bucket.main.id
}
`,
			old: makeTraversal("aws_s3_bucket"), new: makeTraversal("aws_bucket"),
			want: `resource "aws_instance" "example" {
  subnet_id = aws_bucket.main.id
}
`,
		},
		"rename in nested block": {
			input: `resource "aws_instance" "example" {
  ami = "abc"

  ebs_block_device {
    kms_key_id = aws_kms_key.old.arn
  }
}
`,
			old: makeTraversal("aws_kms_key", "old"), new: makeTraversal("aws_kms_key", "new"),
			want: `resource "aws_instance" "example" {
  ami = "abc"

  ebs_block_device {
    kms_key_id = aws_kms_key.new.arn
  }
}
`,
		},
		"no matching references": {
			input: `resource "aws_instance" "example" {
  ami = "abc-123"
}
`,
			old: makeTraversal("aws_s3_bucket"), new: makeTraversal("aws_bucket"),
			want: `resource "aws_instance" "example" {
  ami = "abc-123"
}
`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := ParseFile([]byte(tc.input), "main.tf", nil)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			blocks := f.FindBlocks("resource", "aws_instance")
			blocks[0].RenameReferencePrefix(tc.old, tc.new)
			got := string(f.Bytes())
			if got != tc.want {
				t.Errorf("output mismatch\n--- want ---\n%s\n--- got ---\n%s", tc.want, got)
			}
		})
	}
}

func TestFile_RenameBlockType(t *testing.T) {
	tests := map[string]struct {
		input      string
		oldType    string
		newType    string
		wantSubstr string
		wantAbsent string
	}{
		"rename all matching blocks": {
			input: `resource "aws_instance" "a" {
  ami = "abc"
}

resource "aws_instance" "b" {
  ami = "def"
}

data "aws_ami" "latest" {
  most_recent = true
}
`,
			oldType:    "resource",
			newType:    "moved",
			wantSubstr: "moved",
			wantAbsent: "resource",
		},
		"no matches is no-op": {
			input: `resource "aws_instance" "a" {
  ami = "abc"
}
`,
			oldType:    "data",
			newType:    "moved",
			wantSubstr: "resource",
		},
		"leaves non-matching blocks untouched": {
			input: `resource "aws_instance" "a" {
  ami = "abc"
}

data "aws_ami" "latest" {
  most_recent = true
}
`,
			oldType:    "resource",
			newType:    "moved",
			wantSubstr: "data",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := ParseFile([]byte(tc.input), "main.tf", nil)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			f.RenameBlockType(tc.oldType, tc.newType)
			output := string(f.Bytes())
			if !strings.Contains(output, tc.wantSubstr) {
				t.Errorf("output missing %q\n%s", tc.wantSubstr, output)
			}
			if tc.wantAbsent != "" && strings.Contains(output, tc.wantAbsent) {
				t.Errorf("output should not contain %q\n%s", tc.wantAbsent, output)
			}
		})
	}
}

func TestFile_RemoveBlock(t *testing.T) {
	tests := map[string]struct {
		input      string
		blockType  string
		labels     []string
		wantResult bool
		wantAbsent string
	}{
		"remove matching block": {
			input: `resource "aws_instance" "web" {
  ami = "abc"
}

resource "aws_s3_bucket" "data" {
  bucket = "my-bucket"
}
`,
			blockType:  "resource",
			labels:     []string{"aws_instance", "web"},
			wantResult: true,
			wantAbsent: "aws_instance",
		},
		"no match returns false": {
			input: `resource "aws_instance" "web" {
  ami = "abc"
}
`,
			blockType:  "resource",
			labels:     []string{"aws_s3_bucket", "data"},
			wantResult: false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := ParseFile([]byte(tc.input), "main.tf", nil)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			got := f.RemoveBlock(tc.blockType, tc.labels)
			if got != tc.wantResult {
				t.Errorf("RemoveBlock() = %v, want %v", got, tc.wantResult)
			}
			if tc.wantAbsent != "" {
				output := string(f.Bytes())
				if strings.Contains(output, tc.wantAbsent) {
					t.Errorf("output should not contain %q\n%s", tc.wantAbsent, output)
				}
			}
		})
	}
}

func TestFile_AddBlock(t *testing.T) {
	tests := map[string]struct {
		input      string
		blockType  string
		labels     []string
		attrName   string
		attrVal    cty.Value
		wantSubstr string
	}{
		"add new block with attribute": {
			input:      ``,
			blockType:  "resource",
			labels:     []string{"aws_instance", "new"},
			attrName:   "ami",
			attrVal:    cty.StringVal("abc-123"),
			wantSubstr: `resource "aws_instance" "new"`,
		},
		"append to existing file": {
			input: `resource "aws_instance" "existing" {
  ami = "old"
}
`,
			blockType:  "resource",
			labels:     []string{"aws_s3_bucket", "added"},
			attrName:   "bucket",
			attrVal:    cty.StringVal("my-bucket"),
			wantSubstr: `resource "aws_s3_bucket" "added"`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := ParseFile([]byte(tc.input), "main.tf", nil)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			b := f.AddBlock(tc.blockType, tc.labels)
			if b == nil {
				t.Fatal("AddBlock returned nil")
			}
			b.SetAttributeValue(tc.attrName, tc.attrVal)
			output := string(f.Bytes())
			if !strings.Contains(output, tc.wantSubstr) {
				t.Errorf("output missing %q\n%s", tc.wantSubstr, output)
			}
		})
	}
}

func TestFile_FindBlocks_noLabel(t *testing.T) {
	input := `terraform {
  required_version = ">= 1.0"
}

resource "aws_instance" "web" {
  ami = "abc-123"
}
`

	f, err := ParseFile([]byte(input), "main.tf", nil)
	if err != nil {
		t.Fatalf("unexpected parse error: %s", err)
	}

	blocks := f.FindBlocks("terraform", "")
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(blocks))
	}
	if blocks[0].Type() != "terraform" {
		t.Errorf("Type() = %q, want %q", blocks[0].Type(), "terraform")
	}
}

func TestFile_RenameReferencePrefix(t *testing.T) {
	tests := map[string]struct {
		input string
		old   hcl.Traversal
		new   hcl.Traversal
		want  string
	}{
		"rename across multiple blocks": {
			input: `resource "aws_instance" "a" {
  bucket = aws_s3_bucket.main.id
}

resource "aws_instance" "b" {
  bucket = aws_s3_bucket.other.arn
}
`,
			old: makeTraversal("aws_s3_bucket"), new: makeTraversal("aws_bucket"),
			want: `resource "aws_instance" "a" {
  bucket = aws_bucket.main.id
}

resource "aws_instance" "b" {
  bucket = aws_bucket.other.arn
}
`,
		},
		"rename in top-level locals": {
			input: `locals {
  id = aws_s3_bucket.main.id
}
`,
			old: makeTraversal("aws_s3_bucket"), new: makeTraversal("aws_bucket"),
			want: `locals {
  id = aws_bucket.main.id
}
`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := ParseFile([]byte(tc.input), "main.tf", nil)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			f.RenameReferencePrefix(tc.old, tc.new)
			got := string(f.Bytes())
			if got != tc.want {
				t.Errorf("output mismatch\n--- want ---\n%s\n--- got ---\n%s", tc.want, got)
			}
		})
	}
}
