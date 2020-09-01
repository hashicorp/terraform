# Opinionated Golang tag parser

[![Build Status](https://travis-ci.org/vmihailenco/tagparser.png?branch=master)](https://travis-ci.org/vmihailenco/tagparser)
[![GoDoc](https://godoc.org/github.com/vmihailenco/tagparser?status.svg)](https://godoc.org/github.com/vmihailenco/tagparser)

## Installation

Install:

```shell
go get -u github.com/vmihailenco/tagparser
```

## Quickstart

```go
func ExampleParse() {
	tag := tagparser.Parse("some_name,key:value,key2:'complex value'")
	fmt.Println(tag.Name)
	fmt.Println(tag.Options)
	// Output: some_name
	// map[key:value key2:'complex value']
}
```
