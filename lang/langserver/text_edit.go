package langserver

import (
	"bytes"

	"github.com/agext/levenshtein"

	"github.com/hashicorp/terraform/internal/lsp"
)

// makeTextEdits constructs a slice of LSP text edits representing the
// edits required to change old into new, trying to minimize the number of
// changes.
//
// It primarily deals with whole-line insertions and deletes but will use
// a similarity heuristic to replace the content of some lines in-place where
// sensible, which helps text editor clients to keep other annotations (like
// diagnostics) attached to where they ought to be.
func makeTextEdits(old, new sourceLines, simThreshold float64) []lsp.TextEdit {
	var ret []lsp.TextEdit
	lcs := longestCommonSubsequence(old, new)
	var oldI, newI, lcsI int
	for oldI < len(old) || newI < len(new) || lcsI < len(lcs) {
		for oldI < len(old) && (lcsI >= len(lcs) || !bytes.Equal(old[oldI].content, lcs[lcsI].content)) {
			if newI < len(new) {
				// See if the next "new" is similar enough to our "old" that
				// we'll treat this as an Update rather than a Delete/Create.
				if lineContentsSimilar(old[oldI], new[newI], simThreshold) {
					ret = append(ret, makeTextEditUpdate(old[oldI], new[newI]))
					oldI++
					newI++ // we also consume the next "new" in this case
					continue
				}
			}

			ret = append(ret, makeTextEditDelete(old[oldI]))
			oldI++
		}
		for newI < len(new) && (lcsI >= len(lcs) || !bytes.Equal(new[newI].content, lcs[lcsI].content)) {
			ret = append(ret, makeTextEditInsert(new[newI]))
			newI++
		}
		if lcsI < len(lcs) {
			// All of our indexes advance together now, since the line
			// is common to all three sequences.
			lcsI++
			oldI++
			newI++
		}
	}

	// We've built each of our changes as unaware of those before it, so we'll
	// apply them in reverse order so they don't trample.
	reverseTextEditSlice(ret)

	return ret
}

func makeTextEditUpdate(old, new sourceLine) lsp.TextEdit {
	// For our purposes here, an "update" is still a whole-line replacement
	// but (unlike for a delete and an insert) we do it only for the content
	// of the line and don't touch any line-ending characters, thus allowing
	// the editor to understand that this is still the same line we had before
	// in case it wants to preserve any line-oriented state, such as whether
	// the line begins a folded region.
	// To do this, though, we need to know how long the line is in UTF-16 code
	// units, because that's how LSP counts columns.
	l := old.lspLen()
	return lsp.TextEdit{
		Range: lsp.Range{
			Start: lsp.Position{Line: float64(new.rng.Start.Line - 1), Character: 0},
			End:   lsp.Position{Line: float64(new.rng.Start.Line - 1), Character: float64(l)},
		},
		NewText: string(new.content),
	}
}

func makeTextEditInsert(new sourceLine) lsp.TextEdit {
	pos := lsp.Position{Line: float64(new.rng.Start.Line - 1), Character: 0}
	return lsp.TextEdit{
		Range: lsp.Range{
			Start: pos,
			End:   pos,
		},
		NewText: string(new.content) + "\n",
	}
}

func makeTextEditDelete(old sourceLine) lsp.TextEdit {
	// Delete from the start of this line to the start of the next one, which'll
	// also (helpfully) remove any newline characters we can't otherwise see from here.
	return lsp.TextEdit{
		Range: lsp.Range{
			Start: lsp.Position{Line: float64(old.rng.Start.Line - 1), Character: 0},
			End:   lsp.Position{Line: float64(old.rng.Start.Line), Character: 0},
		},
		NewText: "",
	}
}

// longestCommonSequence is the first step of producing an edit diff: we
// find the longest sequence of consecutive lines that have common content
// in both xs and ys.
func longestCommonSubsequence(xs, ys sourceLines) []sourceLine {
	if len(xs) == 0 || len(ys) == 0 {
		return nil
	}

	c := make([]int, len(xs)*len(ys))
	eqs := make([]bool, len(xs)*len(ys))
	w := len(xs)

	for y := 0; y < len(ys); y++ {
		for x := 0; x < len(xs); x++ {
			if bytes.Equal(xs[x].content, ys[y].content) {
				eqs[(w*y)+x] = true // equality tests can be expensive, so cache it

				// Sequence gets one longer than for the cell at top left,
				// since we'd append a new item to the sequence here.
				if x == 0 || y == 0 {
					c[(w*y)+x] = 1
				} else {
					c[(w*y)+x] = c[(w*(y-1))+(x-1)] + 1
				}
			} else {
				// We follow the longest of the sequence above and the sequence
				// to the left of us in the matrix.
				l := 0
				u := 0
				if x > 0 {
					l = c[(w*y)+(x-1)]
				}
				if y > 0 {
					u = c[(w*(y-1))+x]
				}
				if l > u {
					c[(w*y)+x] = l
				} else {
					c[(w*y)+x] = u
				}
			}
		}
	}

	// The bottom right cell tells us how long our longest sequence will be
	// The result is generic []sourceLine rather than our "sourceLines" named
	// type because part of the contract for a "sourceLines" is that the
	// source line numbers correlate with the line numbers in our ranges.
	seq := make([]sourceLine, c[len(c)-1])

	// Now we will walk back from the bottom right cell, finding again all
	// of the equal pairs to construct our sequence.
	x := len(xs) - 1
	y := len(ys) - 1
	i := len(seq) - 1

	for x > -1 && y > -1 {
		if eqs[(w*y)+x] {
			// Add the value to our result list and then walk diagonally
			// up and to the left.
			seq[i] = xs[x]
			x--
			y--
			i--
		} else {
			// Take the path with the greatest sequence length in the matrix.
			l := 0
			u := 0
			if x > 0 {
				l = c[(w*y)+(x-1)]
			}
			if y > 0 {
				u = c[(w*(y-1))+x]
			}
			if l > u {
				x--
			} else {
				y--
			}
		}
	}

	if i > -1 {
		// should never happen if the matrix was constructed properly
		panic("not enough elements in sequence")
	}

	return seq
}

func lineContentsSimilar(a, b sourceLine, threshold float64) bool {
	sim := levenshtein.Similarity(
		string(a.content), string(b.content),
		levenshtein.NewParams().MinScore(threshold),
	)
	return sim >= threshold
}

// reverseTextEditSlice reverses a slice of text edits in-place
func reverseTextEditSlice(edits []lsp.TextEdit) {
	max := len(edits) / 2
	for i := 0; i < max; i++ {
		j := len(edits) - 1 - i
		edits[i], edits[j] = edits[j], edits[i]
	}
}
