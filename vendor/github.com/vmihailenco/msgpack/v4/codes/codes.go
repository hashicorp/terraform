package codes

type Code byte

var (
	PosFixedNumHigh Code = 0x7f
	NegFixedNumLow  Code = 0xe0

	Nil Code = 0xc0

	False Code = 0xc2
	True  Code = 0xc3

	Float  Code = 0xca
	Double Code = 0xcb

	Uint8  Code = 0xcc
	Uint16 Code = 0xcd
	Uint32 Code = 0xce
	Uint64 Code = 0xcf

	Int8  Code = 0xd0
	Int16 Code = 0xd1
	Int32 Code = 0xd2
	Int64 Code = 0xd3

	FixedStrLow  Code = 0xa0
	FixedStrHigh Code = 0xbf
	FixedStrMask Code = 0x1f
	Str8         Code = 0xd9
	Str16        Code = 0xda
	Str32        Code = 0xdb

	Bin8  Code = 0xc4
	Bin16 Code = 0xc5
	Bin32 Code = 0xc6

	FixedArrayLow  Code = 0x90
	FixedArrayHigh Code = 0x9f
	FixedArrayMask Code = 0xf
	Array16        Code = 0xdc
	Array32        Code = 0xdd

	FixedMapLow  Code = 0x80
	FixedMapHigh Code = 0x8f
	FixedMapMask Code = 0xf
	Map16        Code = 0xde
	Map32        Code = 0xdf

	FixExt1  Code = 0xd4
	FixExt2  Code = 0xd5
	FixExt4  Code = 0xd6
	FixExt8  Code = 0xd7
	FixExt16 Code = 0xd8
	Ext8     Code = 0xc7
	Ext16    Code = 0xc8
	Ext32    Code = 0xc9
)

func IsFixedNum(c Code) bool {
	return c <= PosFixedNumHigh || c >= NegFixedNumLow
}

func IsFixedMap(c Code) bool {
	return c >= FixedMapLow && c <= FixedMapHigh
}

func IsFixedArray(c Code) bool {
	return c >= FixedArrayLow && c <= FixedArrayHigh
}

func IsFixedString(c Code) bool {
	return c >= FixedStrLow && c <= FixedStrHigh
}

func IsString(c Code) bool {
	return IsFixedString(c) || c == Str8 || c == Str16 || c == Str32
}

func IsBin(c Code) bool {
	return c == Bin8 || c == Bin16 || c == Bin32
}

func IsFixedExt(c Code) bool {
	return c >= FixExt1 && c <= FixExt16
}

func IsExt(c Code) bool {
	return IsFixedExt(c) || c == Ext8 || c == Ext16 || c == Ext32
}
