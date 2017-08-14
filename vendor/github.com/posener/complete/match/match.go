package match

// Match matches two strings
// it is used for comparing a term to the last typed
// word, the prefix, and see if it is a possible auto complete option.
type Match func(term, prefix string) bool
