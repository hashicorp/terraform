// Fuzzy searching allows for flexibly matching a string with partial input,
// useful for filtering data very quickly based on lightweight user input.
package fuzzy

import (
	"unicode"
	"unicode/utf8"
)

var noop = func(r rune) rune { return r }

// Match returns true if source matches target using a fuzzy-searching
// algorithm. Note that it doesn't implement Levenshtein distance (see
// RankMatch instead), but rather a simplified version where there's no
// approximation. The method will return true only if each character in the
// source can be found in the target and occurs after the preceding matches.
func Match(source, target string) bool {
	return match(source, target, noop)
}

// MatchFold is a case-insensitive version of Match.
func MatchFold(source, target string) bool {
	return match(source, target, unicode.ToLower)
}

func match(source, target string, fn func(rune) rune) bool {
	lenDiff := len(target) - len(source)

	if lenDiff < 0 {
		return false
	}

	if lenDiff == 0 && source == target {
		return true
	}

Outer:
	for _, r1 := range source {
		for i, r2 := range target {
			if fn(r1) == fn(r2) {
				target = target[i+utf8.RuneLen(r2):]
				continue Outer
			}
		}
		return false
	}

	return true
}

// Find will return a list of strings in targets that fuzzy matches source.
func Find(source string, targets []string) []string {
	return find(source, targets, noop)
}

// FindFold is a case-insensitive version of Find.
func FindFold(source string, targets []string) []string {
	return find(source, targets, unicode.ToLower)
}

func find(source string, targets []string, fn func(rune) rune) []string {
	var matches []string

	for _, target := range targets {
		if match(source, target, fn) {
			matches = append(matches, target)
		}
	}

	return matches
}

// RankMatch is similar to Match except it will measure the Levenshtein
// distance between the source and the target and return its result. If there
// was no match, it will return -1.
// Given the requirements of match, RankMatch only needs to perform a subset of
// the Levenshtein calculation, only deletions need be considered, required
// additions and substitutions would fail the match test.
func RankMatch(source, target string) int {
	return rank(source, target, noop)
}

// RankMatchFold is a case-insensitive version of RankMatch.
func RankMatchFold(source, target string) int {
	return rank(source, target, unicode.ToLower)
}

func rank(source, target string, fn func(rune) rune) int {
	lenDiff := len(target) - len(source)

	if lenDiff < 0 {
		return -1
	}

	if lenDiff == 0 && source == target {
		return 0
	}

	runeDiff := 0

Outer:
	for _, r1 := range source {
		for i, r2 := range target {
			if fn(r1) == fn(r2) {
				target = target[i+utf8.RuneLen(r2):]
				continue Outer
			} else {
				runeDiff++
			}
		}
		return -1
	}

	// Count up remaining char
	for len(target) > 0 {
		target = target[utf8.RuneLen(rune(target[0])):]
		runeDiff++
	}

	return runeDiff
}

// RankFind is similar to Find, except it will also rank all matches using
// Levenshtein distance.
func RankFind(source string, targets []string) Ranks {
	var r Ranks
	for _, target := range find(source, targets, noop) {
		distance := LevenshteinDistance(source, target)
		r = append(r, Rank{source, target, distance})
	}
	return r
}

// RankFindFold is a case-insensitive version of RankFind.
func RankFindFold(source string, targets []string) Ranks {
	var r Ranks
	for _, target := range find(source, targets, unicode.ToLower) {
		distance := LevenshteinDistance(source, target)
		r = append(r, Rank{source, target, distance})
	}
	return r
}

type Rank struct {
	// Source is used as the source for matching.
	Source string

	// Target is the word matched against.
	Target string

	// Distance is the Levenshtein distance between Source and Target.
	Distance int
}

type Ranks []Rank

func (r Ranks) Len() int {
	return len(r)
}

func (r Ranks) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r Ranks) Less(i, j int) bool {
	return r[i].Distance < r[j].Distance
}
