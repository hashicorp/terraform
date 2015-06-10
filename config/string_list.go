package config

import "strings"

// StringList represents the "poor man's list" that terraform uses
// internally
type StringList string

// This is the delimiter used to recognize and split StringLists
const StringListDelim = `B780FFEC-B661-4EB8-9236-A01737AD98B6`

// Build a StringList from a slice
func NewStringList(parts []string) StringList {
	// FOR NOW:
	return StringList(strings.Join(parts, StringListDelim))
	// EVENTUALLY:
	// var sl StringList
	// for _, p := range parts {
	// 	sl = sl.Append(p)
	// }
	// return sl
}

// Returns a new StringList with the item appended
func (sl StringList) Append(s string) StringList {
	// FOR NOW:
	return StringList(strings.Join(append(sl.Slice(), s), StringListDelim))
	// EVENTUALLY:
	// return StringList(fmt.Sprintf("%s%s%s", sl, s, StringListDelim))
}

// Returns an element at the index, wrapping around the length of the string
// when index > list length
func (sl StringList) Element(index int) string {
	return sl.Slice()[index%sl.Length()]
}

// Returns the length of the StringList
func (sl StringList) Length() int {
	return len(sl.Slice())
}

// Returns a slice of strings as represented by this StringList
func (sl StringList) Slice() []string {
	parts := strings.Split(string(sl), StringListDelim)

	// FOR NOW:
	if sl.String() == "" {
		return []string{}
	} else {
		return parts
	}
	// EVENTUALLY:
	// StringLists always have a trailing StringListDelim
	// return parts[:len(parts)-1]
}

func (sl StringList) String() string {
	return string(sl)
}

// Determines if a given string represents a StringList
func IsStringList(s string) bool {
	return strings.Contains(s, StringListDelim)
}
