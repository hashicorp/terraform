// Package luacty is an adapter library that wraps values from cty -- a
// configuration-oriented dynamic type system in Go -- so that they can be
// used in Lua programs executed by GopherLua.
//
// cty and Lua are both, in different ways, tailored for configuration
// use-cases. This bridge allows applications that are using cty for other
// reasons, such as more declarative configuration formats that produce cty
// results, to interchange those same results with Lua programs in situations
// where a scripting language is more appropriate.
//
// luacty is a two-way bridge: it allows cty values to be wrapped in Lua's
// "userdata" type for direct use in Lua, and it also allows many of Lua's
// native types to be converted to cty values so that script results can be
// consumed.
//
// Conversion of cty values to Lua values is via wrapping: Lua features are
// used to implement normal Lua operators against the real underlying cty
// values, via the cty API. This causes operations on cty values to behave
// like they would within the Go API, but also means that certain cty semantics
// "leak in" to Lua: the list vs. map vs. object vs. tuple distinction is
// retained, for example, rather than converting to a generic Lua table, and
// cty lists and tuples retain their zero-indexing rather than adopting the
// one-indexing that Lua uses for its own indexed tables.
//
// Conversion of Lua values out to cty is done by actual conversion rather
// than wrapping, producing new cty values that start with equivalent content
// to the given Lua value but using cty semantics rather than Lua semantics.
// Not all Lua values can convert to cty values, and conversions may be
// "lossy" in the sense that the original Lua value cannot be easily recovered.
// The available automatic conversions are:
//
//     LNilType    cty.NullVal(cty.DynamicPseudoType)
//     LBool       cty.Bool
//     LNumber     cty.Number
//     LString     cty.String
//     LTable      cty.Object
//     LUserData   cty.Value (when userdata was created by this package)
//
// When cty type information can be provided for conversion, additional
// conversions are possible for LTable, allowing conversions to cty.List,
// cty.Map and cty.Tuple where the table content meets the constraints of
// these types.
//
// No value conversions are available for Lua functions or userdata that
// was created by other packages. A wrapper is provided to allow Lua functions
// to be used as cty functions within applications that make use
// of the cty function extension, but cty functions are not cty values.
package luacty
