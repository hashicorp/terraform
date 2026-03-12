// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package ast

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestAST_write_back_loop(t *testing.T) {
	for name, input := range map[string]map[string]string{
		"empty": {},
		"simple": {
			"main.tf": `
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
		},

		"multiple files": {
			"main.tf": `
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
			"modules/module_a/main.tf": `
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
			"modules/module_b/main.tf": `
		variable "name" {
				type = string
		}

		output "name_upper" {
				value = upper(var.name)
		}
		`,
		},
		"complex expressions": {
			"main.tf": `
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
		numbers   = [0, 1, 2, 3]
		squares   = [for n in local.numbers : n * n]
		indexed   = { for idx, val in local.numbers : idx => val }

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
		},

		"resources and data sources": {
			"main.tf": `
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
		},

		// TODO: terraform test
		// TODO: list blocks
		// TODO: preserve formatting (or require fmt to succeed) and comments
		// TODO: invalid syntax and error handling
		// TODO: handle overrides (or would we do that on an abstraction level or not at all?)
		// TODO: fail on tf.json files
	} {
		t.Run(name, func(t *testing.T) {
			parsed := map[string]*hcl.File{}
			for filename, content := range input {
				file, diags := hclparse.NewParser().ParseHCL([]byte(content), filename)
				if diags.HasErrors() {
					t.Fatalf("unexpected error parsing input in test setup: %s", diags.Error())
				}
				parsed[filename] = file
			}

			ast, diags := FromConfig(parsed)
			tfdiags.AssertNoDiagnostics(t, diags)

			output, diags := WriteAST(ast)
			tfdiags.AssertNoDiagnostics(t, diags)

			// Check if input and output are the same, which they should be since we haven't made any changes to the AST.
			for filename, content := range input {
				outputContent, ok := output[filename]
				if !ok {
					t.Errorf("expected output to contain file %q, but it was missing", filename)
					continue
				}
				if string(outputContent) != content {
					t.Errorf("expected output content for file %q to be the same as input, but it was different", filename)
				}
			}
		})
	}
}
