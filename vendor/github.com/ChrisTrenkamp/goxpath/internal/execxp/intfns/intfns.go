package intfns

import (
	"encoding/xml"

	"github.com/ChrisTrenkamp/goxpath/tree"
)

//BuiltIn contains the list of built-in XPath functions
var BuiltIn = map[xml.Name]tree.Wrap{
	//String functions
	{Local: "string"}:           {Fn: _string, NArgs: 1, LastArgOpt: tree.Optional},
	{Local: "concat"}:           {Fn: concat, NArgs: 3, LastArgOpt: tree.Variadic},
	{Local: "starts-with"}:      {Fn: startsWith, NArgs: 2},
	{Local: "contains"}:         {Fn: contains, NArgs: 2},
	{Local: "substring-before"}: {Fn: substringBefore, NArgs: 2},
	{Local: "substring-after"}:  {Fn: substringAfter, NArgs: 2},
	{Local: "substring"}:        {Fn: substring, NArgs: 3, LastArgOpt: tree.Optional},
	{Local: "string-length"}:    {Fn: stringLength, NArgs: 1, LastArgOpt: tree.Optional},
	{Local: "normalize-space"}:  {Fn: normalizeSpace, NArgs: 1, LastArgOpt: tree.Optional},
	{Local: "translate"}:        {Fn: translate, NArgs: 3},
	//Node set functions
	{Local: "last"}:          {Fn: last},
	{Local: "position"}:      {Fn: position},
	{Local: "count"}:         {Fn: count, NArgs: 1},
	{Local: "local-name"}:    {Fn: localName, NArgs: 1, LastArgOpt: tree.Optional},
	{Local: "namespace-uri"}: {Fn: namespaceURI, NArgs: 1, LastArgOpt: tree.Optional},
	{Local: "name"}:          {Fn: name, NArgs: 1, LastArgOpt: tree.Optional},
	//boolean functions
	{Local: "boolean"}: {Fn: boolean, NArgs: 1},
	{Local: "not"}:     {Fn: not, NArgs: 1},
	{Local: "true"}:    {Fn: _true},
	{Local: "false"}:   {Fn: _false},
	{Local: "lang"}:    {Fn: lang, NArgs: 1},
	//number functions
	{Local: "number"}:  {Fn: number, NArgs: 1, LastArgOpt: tree.Optional},
	{Local: "sum"}:     {Fn: sum, NArgs: 1},
	{Local: "floor"}:   {Fn: floor, NArgs: 1},
	{Local: "ceiling"}: {Fn: ceiling, NArgs: 1},
	{Local: "round"}:   {Fn: round, NArgs: 1},
}
