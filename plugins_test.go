package main

import (
	"testing"
)

func TestBuiltinProviders(t *testing.T) {
	internal := builtinProviders()
	tfProvider, err := internal["terraform"]()
	if err != nil {
		t.Fatal(err)
	}

	schema := tfProvider.GetProviderSchema()
	_, found := schema.DataSources["terraform_remote_state"]
	if !found {
		t.Errorf("didn't find terraform_remote_state in internal \"terraform\" provider")
	}
}
