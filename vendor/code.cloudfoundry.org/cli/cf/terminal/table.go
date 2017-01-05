package terminal

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// PrintableTable is an implementation of the Table interface. It
// remembers the headers, the added rows, the column widths, and a
// number of other things.
type Table struct {
	ui            UI
	headers       []string
	headerPrinted bool
	columnWidth   []int
	rowHeight     []int
	rows          [][]string
	colSpacing    string
	transformer   []Transformer
}

// Transformer is the type of functions used to modify the content of
// a table cell for actual display. For multi-line content of a cell
// the transformation is applied to each individual line.
type Transformer func(s string) string

// NewTable is the constructor function creating a new printable table
// from a list of headers. The table is also connected to a UI, which
// is where it will print itself to on demand.
func NewTable(headers []string) *Table {
	pt := &Table{
		headers:     headers,
		columnWidth: make([]int, len(headers)),
		colSpacing:  "   ",
		transformer: make([]Transformer, len(headers)),
	}
	// Standard colorization, column 0 is auto-highlighted as some
	// name. Everything else has no transformation (== identity
	// transform)
	for i := range pt.transformer {
		pt.transformer[i] = nop
	}
	if 0 < len(headers) {
		pt.transformer[0] = TableContentHeaderColor
	}
	return pt
}

// NoHeaders disables the printing of the header row for the specified
// table.
func (t *Table) NoHeaders() {
	// Fake the Print() code into the belief that the headers have
	// been printed already.
	t.headerPrinted = true
}

// SetTransformer specifies a string transformer to apply to the
// content of the given column in the specified table.
func (t *Table) SetTransformer(columnIndex int, tr Transformer) {
	t.transformer[columnIndex] = tr
}

// Add extends the table by another row.
func (t *Table) Add(row ...string) {
	t.rows = append(t.rows, row)
}

// PrintTo is the core functionality for printing the table, placing
// the formatted table into the writer given to it as argument. The
// exported Print() is just a wrapper around this which redirects the
// result into CF datastructures.
func (t *Table) PrintTo(result io.Writer) error {
	t.rowHeight = make([]int, len(t.rows)+1)

	rowIndex := 0
	if !t.headerPrinted {
		// row transformer header row
		err := t.calculateMaxSize(transHeader, rowIndex, t.headers)
		if err != nil {
			return err
		}
		rowIndex++
	}

	for _, row := range t.rows {
		// table is row transformer itself, for content rows
		err := t.calculateMaxSize(t, rowIndex, row)
		if err != nil {
			return err
		}
		rowIndex++
	}

	rowIndex = 0
	if !t.headerPrinted {
		err := t.printRow(result, transHeader, rowIndex, t.headers)
		if err != nil {
			return err
		}
		t.headerPrinted = true
		rowIndex++
	}

	for row := range t.rows {
		err := t.printRow(result, t, rowIndex, t.rows[row])
		if err != nil {
			return err
		}
		rowIndex++
	}

	// Note, printing a table clears it.
	t.rows = [][]string{}
	return nil
}

// calculateMaxSize iterates over the collected rows of the specified
// table, and their strings, determining the height of each row (in
// lines), and the width of each column (in characters). The results
// are stored in the table for use by Print.
func (t *Table) calculateMaxSize(transformer rowTransformer, rowIndex int, row []string) error {

	// Iterate columns
	for columnIndex := range row {
		// Truncate long row, ignore the additional fields.
		if columnIndex >= len(t.headers) {
			break
		}

		// Note that the length of the cell in characters is
		// __not__ equivalent to its width.  Because it may be
		// a multi-line value. We have to split the cell into
		// lines and check the width of each such fragment.
		// The number of lines founds also goes into the row
		// height.

		lines := strings.Split(row[columnIndex], "\n")
		height := len(lines)

		if t.rowHeight[rowIndex] < height {
			t.rowHeight[rowIndex] = height
		}

		for i := range lines {
			// (**) See also 'printCellValue' (pCV). Here
			// and there we have to apply identical
			// transformations to the cell value to get
			// matching cell width information. If they do
			// not match then pCV may compute a cell width
			// larger than the max width found here, a
			// negative padding length from that, and
			// subsequently return an error.  What
			// was further missing is trimming before
			// entering the user-transform. Especially
			// with color transforms any trailing space
			// going in will not be removable for print.
			//
			// This happened for
			// https://www.pivotaltracker.com/n/projects/892938/stories/117404629

			value := trim(Decolorize(transformer.Transform(columnIndex, trim(lines[i]))))
			width, err := visibleSize(value)
			if err != nil {
				return err
			}
			if t.columnWidth[columnIndex] < width {
				t.columnWidth[columnIndex] = width
			}
		}
	}
	return nil
}

// printRow is responsible for the layouting, transforming and
// printing of the string in a single row
func (t *Table) printRow(result io.Writer, transformer rowTransformer, rowIndex int, row []string) error {

	height := t.rowHeight[rowIndex]

	// Compute the index of the last column as the min number of
	// cells in the header and cells in the current row.
	// Note: math.Min seems to be for float only :(
	last := len(t.headers) - 1
	lastr := len(row) - 1
	if lastr < last {
		last = lastr
	}

	// Note how we always print into a line buffer before placing
	// the assembled line into the result. This allows us to trim
	// superfluous trailing whitespace from the line before making
	// it final.

	if height <= 1 {
		// Easy case, all cells in the row are single-line
		line := &bytes.Buffer{}

		for columnIndex := range row {
			// Truncate long row, ignore the additional fields.
			if columnIndex >= len(t.headers) {
				break
			}

			err := t.printCellValue(line, transformer, columnIndex, last, row[columnIndex])
			if err != nil {
				return err
			}
		}

		fmt.Fprintf(result, "%s\n", trim(string(line.Bytes())))
		return nil
	}

	// We have at least one multi-line cell in this row.
	// Treat it a bit like a mini-table.

	// Step I. Fill the mini-table. Note how it is stored
	//         column-major, not row-major.

	// [column][row]string
	sub := make([][]string, len(t.headers))
	for columnIndex := range row {
		// Truncate long row, ignore the additional fields.
		if columnIndex >= len(t.headers) {
			break
		}
		sub[columnIndex] = strings.Split(row[columnIndex], "\n")
		// (*) Extend the column to the full height.
		for len(sub[columnIndex]) < height {
			sub[columnIndex] = append(sub[columnIndex], "")
		}
	}

	// Step II. Iterate over the rows, then columns to
	//          collect the output. This assumes that all
	//          the rows in sub are the same height. See
	//          (*) above where that is made true.

	for rowIndex := range sub[0] {
		line := &bytes.Buffer{}

		for columnIndex := range sub {
			err := t.printCellValue(line, transformer, columnIndex, last, sub[columnIndex][rowIndex])
			if err != nil {
				return err
			}
		}

		fmt.Fprintf(result, "%s\n", trim(string(line.Bytes())))
	}
	return nil
}

// printCellValue pads the specified string to the width of the given
// column, adds the spacing bewtween columns, and returns the result.
func (t *Table) printCellValue(result io.Writer, transformer rowTransformer, col, last int, value string) error {
	value = trim(transformer.Transform(col, trim(value)))
	fmt.Fprint(result, value)

	// Pad all columns, but the last in this row (with the size of
	// the header row limiting this). This ensures that most of
	// the irrelevant spacing is not printed. At the moment
	// irrelevant spacing can only occur when printing a row with
	// multi-line cells, introducing a physical short line for a
	// long logical row. Getting rid of that requires fixing in
	// printRow.
	//
	//  Note how the inter-column spacing is also irrelevant for
	//  that last column.

	if col < last {
		// (**) See also 'calculateMaxSize' (cMS). Here and
		// there we have to apply identical transformations to
		// the cell value to get matching cell width
		// information. If they do not match then we may here
		// compute a cell width larger than the max width
		// found by cMS, derive a negative padding length from
		// that, and subsequently return an error. What was
		// further missing is trimming before entering the
		// user-transform. Especially with color transforms
		// any trailing space going in will not be removable
		// for print.
		//
		// This happened for
		// https://www.pivotaltracker.com/n/projects/892938/stories/117404629

		decolorizedLength, err := visibleSize(trim(Decolorize(value)))
		if err != nil {
			return err
		}
		padlen := t.columnWidth[col] - decolorizedLength
		padding := strings.Repeat(" ", padlen)
		fmt.Fprint(result, padding)
		fmt.Fprint(result, t.colSpacing)
	}
	return nil
}

// rowTransformer is an interface behind which we can specify how to
// transform the strings of an entire row on a column-by-column basis.
type rowTransformer interface {
	Transform(column int, s string) string
}

// transformHeader is an implementation of rowTransformer which
// highlights all columns as a Header.
type transformHeader struct{}

// transHeader holds a package-global transformHeader to prevent us
// from continuously allocating a literal of the type whenever we
// print a header row. Instead all tables use this value.
var transHeader = &transformHeader{}

// Transform performs the Header highlighting for transformHeader
func (th *transformHeader) Transform(column int, s string) string {
	return HeaderColor(s)
}

// Transform makes a PrintableTable an implementation of
// rowTransformer. It performs the per-column transformation for table
// content, as specified during construction and/or overridden by the
// user of the table, see SetTransformer.
func (t *Table) Transform(column int, s string) string {
	return t.transformer[column](s)
}

// nop is the identity transformation which does not transform the
// string at all.
func nop(s string) string {
	return s
}

// trim is a helper to remove trailing whitespace from a string.
func trim(s string) string {
	return strings.TrimRight(s, " \t")
}

// visibleSize returns the number of columns the string will cover
// when displayed in the terminal. This is the number of runes,
// i.e. characters, not the number of bytes it consists of.
func visibleSize(s string) (int, error) {
	// This code re-implements the basic functionality of
	// RuneCountInString to account for special cases. Namely
	// UTF-8 characters taking up 3 bytes (**) appear as double-width.
	//
	// (**) I wonder if that is the set of characters outside of
	// the BMP <=> the set of characters requiring surrogates (2
	// slots) when encoded in UCS-2.

	r := strings.NewReader(s)

	var size int
	for range s {
		_, runeSize, err := r.ReadRune()
		if err != nil {
			return -1, fmt.Errorf("error when calculating visible size of: %s", s)
		}

		if runeSize == 3 {
			size += 2 // Kanji and Katakana characters appear as double-width
		} else {
			size++
		}
	}

	return size, nil
}
