// The "playground" main package is an alternative entry point to Terraform
// designed to be compiled for the js_wasm target and thus produce a
// WebAssembly module that exports Terraform Core functionality so that it
// can be used as part of a client-side web application for experimenting
// with the Terraform language using mock infrastructure.
//
// The API exposed by the resulting WebAssembly module is, for now at least,
// considered to be tightly coupled with the single playground web application
// also maintained in this codebase. The API is subject to change at any time
// and so we recommend against using the module in other web applications.
package main

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/tfdiags"
)

func main() {
	select {}
	realMain()
}

func realMain() {
	config, diags := config()
	if diags.HasErrors() {
		fmt.Println(diags.Err().Error())
		return
	}
	fmt.Printf("Config is %s", spew.Sdump(config))

	/*ctx, diags := terraform.NewContext(&terraform.ContextOpts{
		Config: config,
	})
	if diags.HasErrors() {
		fmt.Println(diags.Err().Error())
	}
	fmt.Println("Hello, WebAssembly!", ctx)*/
}

func configSnapshot() *configload.Snapshot {
	snap := &configload.Snapshot{
		Modules: map[string]*configload.SnapshotModule{
			"": {
				Dir: ".",
				Files: map[string][]byte{
					"outputs.tf": []byte(`
output "test" {
	value = "baz"
}
`),
				},
			},
		},
	}
	return snap
}

func config() (*configs.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	snap := configSnapshot()
	loader := configload.NewLoaderFromSnapshot(snap)
	config, confDiags := loader.LoadConfig(".")
	diags = diags.Append(confDiags)
	return config, diags
}
