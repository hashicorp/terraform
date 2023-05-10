//go:build tools
// +build tools

package main

import (
	_ "github.com/hashicorp/copywrite"
)

//go:generate go run github.com/hashicorp/copywrite headers
