package hclwrite

import (
	"sort"

	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hcl"
)

// Our "parser" here is actually not doing any parsing of its own. Instead,
// it leans on the native parser in zclsyntax, and then uses the source ranges
// from the AST to partition the raw token sequence to match the raw tokens
// up to AST nodes.
//
// This strategy feels somewhat counter-intuitive, since most of the work the
// parser does is thrown away here, but this strategy is chosen because the
// normal parsing work done by zclsyntax is considered to be the "main case",
// while modifying and re-printing source is more of an edge case, used only
// in ancillary tools, and so it's good to keep all the main parsing logic
// with the main case but keep all of the extra complexity of token wrangling
// out of the main parser, which is already rather complex just serving the
// use-cases it already serves.
//
// If the parsing step produces any errors, the returned File is nil because
// we can't reliably extract tokens from the partial AST produced by an
// erroneous parse.
func parse(src []byte, filename string, start hcl.Pos) (*File, hcl.Diagnostics) {
	file, diags := hclsyntax.ParseConfig(src, filename, start)
	if diags.HasErrors() {
		return nil, diags
	}

	// To do our work here, we use the "native" tokens (those from zclsyntax)
	// to match against source ranges in the AST, but ultimately produce
	// slices from our sequence of "writer" tokens, which contain only
	// *relative* position information that is more appropriate for
	// transformation/writing use-cases.
	nativeTokens, diags := hclsyntax.LexConfig(src, filename, start)
	if diags.HasErrors() {
		// should never happen, since we would've caught these diags in
		// the first call above.
		return nil, diags
	}
	writerTokens := writerTokens(nativeTokens)

	from := inputTokens{
		nativeTokens: nativeTokens,
		writerTokens: writerTokens,
	}

	before, root, after := parseBody(file.Body.(*hclsyntax.Body), from)

	return &File{
		Name:     filename,
		SrcBytes: src,

		Body: root,
		AllTokens: &TokenSeq{
			before.Seq(),
			root.AllTokens,
			after.Seq(),
		},
	}, nil
}

type inputTokens struct {
	nativeTokens hclsyntax.Tokens
	writerTokens Tokens
}

func (it inputTokens) Partition(rng hcl.Range) (before, within, after inputTokens) {
	start, end := partitionTokens(it.nativeTokens, rng)
	before = it.Slice(0, start)
	within = it.Slice(start, end)
	after = it.Slice(end, len(it.nativeTokens))
	return
}

// PartitionIncludeComments is like Partition except the returned "within"
// range includes any lead and line comments associated with the range.
func (it inputTokens) PartitionIncludingComments(rng hcl.Range) (before, within, after inputTokens) {
	start, end := partitionTokens(it.nativeTokens, rng)
	start = partitionLeadCommentTokens(it.nativeTokens[:start])
	_, afterNewline := partitionLineEndTokens(it.nativeTokens[end:])
	end += afterNewline

	before = it.Slice(0, start)
	within = it.Slice(start, end)
	after = it.Slice(end, len(it.nativeTokens))
	return

}

// PartitionBlockItem is similar to PartitionIncludeComments but it returns
// the comments as separate token sequences so that they can be captured into
// AST attributes. It makes assumptions that apply only to block items, so
// should not be used for other constructs.
func (it inputTokens) PartitionBlockItem(rng hcl.Range) (before, leadComments, within, lineComments, newline, after inputTokens) {
	before, within, after = it.Partition(rng)
	before, leadComments = before.PartitionLeadComments()
	lineComments, newline, after = after.PartitionLineEndTokens()
	return
}

func (it inputTokens) PartitionLeadComments() (before, within inputTokens) {
	start := partitionLeadCommentTokens(it.nativeTokens)
	before = it.Slice(0, start)
	within = it.Slice(start, len(it.nativeTokens))
	return
}

func (it inputTokens) PartitionLineEndTokens() (comments, newline, after inputTokens) {
	afterComments, afterNewline := partitionLineEndTokens(it.nativeTokens)
	comments = it.Slice(0, afterComments)
	newline = it.Slice(afterComments, afterNewline)
	after = it.Slice(afterNewline, len(it.nativeTokens))
	return
}

func (it inputTokens) Slice(start, end int) inputTokens {
	// When we slice, we create a new slice with no additional capacity because
	// we expect that these slices will be mutated in order to insert
	// new code into the AST, and we want to ensure that a new underlying
	// array gets allocated in that case, rather than writing into some
	// following slice and corrupting it.
	return inputTokens{
		nativeTokens: it.nativeTokens[start:end:end],
		writerTokens: it.writerTokens[start:end:end],
	}
}

func (it inputTokens) Len() int {
	return len(it.nativeTokens)
}

func (it inputTokens) Seq() *TokenSeq {
	return &TokenSeq{it.writerTokens}
}

func (it inputTokens) Types() []hclsyntax.TokenType {
	ret := make([]hclsyntax.TokenType, len(it.nativeTokens))
	for i, tok := range it.nativeTokens {
		ret[i] = tok.Type
	}
	return ret
}

// parseBody locates the given body within the given input tokens and returns
// the resulting *Body object as well as the tokens that appeared before and
// after it.
func parseBody(nativeBody *hclsyntax.Body, from inputTokens) (inputTokens, *Body, inputTokens) {
	before, within, after := from.PartitionIncludingComments(nativeBody.SrcRange)

	// The main AST doesn't retain the original source ordering of the
	// body items, so we need to reconstruct that ordering by inspecting
	// their source ranges.
	nativeItems := make([]hclsyntax.Node, 0, len(nativeBody.Attributes)+len(nativeBody.Blocks))
	for _, nativeAttr := range nativeBody.Attributes {
		nativeItems = append(nativeItems, nativeAttr)
	}
	for _, nativeBlock := range nativeBody.Blocks {
		nativeItems = append(nativeItems, nativeBlock)
	}
	sort.Sort(nativeNodeSorter{nativeItems})

	body := &Body{
		IndentLevel: 0, // TODO: deal with this
	}

	remain := within
	for _, nativeItem := range nativeItems {
		beforeItem, item, afterItem := parseBodyItem(nativeItem, remain)

		if beforeItem.Len() > 0 {
			body.AppendUnstructuredTokens(beforeItem.Seq())
		}
		body.AppendItem(item)

		remain = afterItem
	}

	if remain.Len() > 0 {
		body.AppendUnstructuredTokens(remain.Seq())
	}

	return before, body, after
}

func parseBodyItem(nativeItem hclsyntax.Node, from inputTokens) (inputTokens, Node, inputTokens) {
	before, leadComments, within, lineComments, newline, after := from.PartitionBlockItem(nativeItem.Range())

	var item Node

	switch tItem := nativeItem.(type) {
	case *hclsyntax.Attribute:
		item = parseAttribute(tItem, within, leadComments, lineComments, newline)
	case *hclsyntax.Block:
		item = parseBlock(tItem, within, leadComments, lineComments, newline)
	default:
		// should never happen if caller is behaving
		panic("unsupported native item type")
	}

	return before, item, after
}

func parseAttribute(nativeAttr *hclsyntax.Attribute, from, leadComments, lineComments, newline inputTokens) *Attribute {
	var allTokens TokenSeq
	attr := &Attribute{}

	if leadComments.Len() > 0 {
		attr.LeadCommentTokens = leadComments.Seq()
		allTokens = append(allTokens, attr.LeadCommentTokens)
	}

	before, nameTokens, from := from.Partition(nativeAttr.NameRange)
	if before.Len() > 0 {
		allTokens = append(allTokens, before.Seq())
	}
	attr.NameTokens = nameTokens.Seq()
	allTokens = append(allTokens, attr.NameTokens)

	before, equalsTokens, from := from.Partition(nativeAttr.EqualsRange)
	if before.Len() > 0 {
		allTokens = append(allTokens, before.Seq())
	}
	attr.EqualsTokens = equalsTokens.Seq()
	allTokens = append(allTokens, attr.EqualsTokens)

	before, exprTokens, from := from.Partition(nativeAttr.Expr.Range())
	if before.Len() > 0 {
		allTokens = append(allTokens, before.Seq())
	}
	attr.Expr = parseExpression(nativeAttr.Expr, exprTokens)
	allTokens = append(allTokens, attr.Expr.AllTokens)

	if lineComments.Len() > 0 {
		attr.LineCommentTokens = lineComments.Seq()
		allTokens = append(allTokens, attr.LineCommentTokens)
	}

	if newline.Len() > 0 {
		attr.EOLTokens = newline.Seq()
		allTokens = append(allTokens, attr.EOLTokens)
	}

	// Collect any stragglers, though there shouldn't be any
	if from.Len() > 0 {
		allTokens = append(allTokens, from.Seq())
	}

	attr.AllTokens = &allTokens

	return attr
}

func parseBlock(nativeBlock *hclsyntax.Block, from, leadComments, lineComments, newline inputTokens) *Block {
	var allTokens TokenSeq
	block := &Block{}

	if leadComments.Len() > 0 {
		block.LeadCommentTokens = leadComments.Seq()
		allTokens = append(allTokens, block.LeadCommentTokens)
	}

	before, typeTokens, from := from.Partition(nativeBlock.TypeRange)
	if before.Len() > 0 {
		allTokens = append(allTokens, before.Seq())
	}
	block.TypeTokens = typeTokens.Seq()
	allTokens = append(allTokens, block.TypeTokens)

	for _, rng := range nativeBlock.LabelRanges {
		var labelTokens inputTokens
		before, labelTokens, from = from.Partition(rng)
		if before.Len() > 0 {
			allTokens = append(allTokens, before.Seq())
		}
		seq := labelTokens.Seq()
		block.LabelTokens = append(block.LabelTokens, seq)
		*(block.LabelTokensFlat) = append(*(block.LabelTokensFlat), seq)
		allTokens = append(allTokens, seq)
	}

	before, oBrace, from := from.Partition(nativeBlock.OpenBraceRange)
	if before.Len() > 0 {
		allTokens = append(allTokens, before.Seq())
	}
	block.OBraceTokens = oBrace.Seq()
	allTokens = append(allTokens, block.OBraceTokens)

	// We go a bit out of order here: we go hunting for the closing brace
	// so that we have a delimited body, but then we'll deal with the body
	// before we actually append the closing brace and any straggling tokens
	// that appear after it.
	bodyTokens, cBrace, from := from.Partition(nativeBlock.CloseBraceRange)
	before, body, after := parseBody(nativeBlock.Body, bodyTokens)

	if before.Len() > 0 {
		allTokens = append(allTokens, before.Seq())
	}
	block.Body = body
	allTokens = append(allTokens, body.AllTokens)
	if after.Len() > 0 {
		allTokens = append(allTokens, after.Seq())
	}

	block.CBraceTokens = cBrace.Seq()
	allTokens = append(allTokens, block.CBraceTokens)

	// stragglers
	if after.Len() > 0 {
		allTokens = append(allTokens, from.Seq())
	}
	if lineComments.Len() > 0 {
		// blocks don't actually have line comments, so we'll just treat
		// them as extra stragglers
		allTokens = append(allTokens, lineComments.Seq())
	}
	if newline.Len() > 0 {
		block.EOLTokens = newline.Seq()
		allTokens = append(allTokens, block.EOLTokens)
	}

	block.AllTokens = &allTokens
	return block
}

func parseExpression(nativeExpr hclsyntax.Expression, from inputTokens) *Expression {
	// TODO: Populate VarRefs by analyzing the result of nativeExpr.Variables()
	return &Expression{
		AllTokens: from.Seq(),
	}
}

// writerTokens takes a sequence of tokens as produced by the main zclsyntax
// package and transforms it into an equivalent sequence of tokens using
// this package's own token model.
//
// The resulting list contains the same number of tokens and uses the same
// indices as the input, allowing the two sets of tokens to be correlated
// by index.
func writerTokens(nativeTokens hclsyntax.Tokens) Tokens {
	// Ultimately we want a slice of token _pointers_, but since we can
	// predict how much memory we're going to devote to tokens we'll allocate
	// it all as a single flat buffer and thus give the GC less work to do.
	tokBuf := make([]Token, len(nativeTokens))
	var lastByteOffset int
	for i, mainToken := range nativeTokens {
		// Create a copy of the bytes so that we can mutate without
		// corrupting the original token stream.
		bytes := make([]byte, len(mainToken.Bytes))
		copy(bytes, mainToken.Bytes)

		tokBuf[i] = Token{
			Type:  mainToken.Type,
			Bytes: bytes,

			// We assume here that spaces are always ASCII spaces, since
			// that's what the scanner also assumes, and thus the number
			// of bytes skipped is also the number of space characters.
			SpacesBefore: mainToken.Range.Start.Byte - lastByteOffset,
		}

		lastByteOffset = mainToken.Range.End.Byte
	}

	// Now make a slice of pointers into the previous slice.
	ret := make(Tokens, len(tokBuf))
	for i := range ret {
		ret[i] = &tokBuf[i]
	}

	return ret
}

// partitionTokens takes a sequence of tokens and a hcl.Range and returns
// two indices within the token sequence that correspond with the range
// boundaries, such that the slice operator could be used to produce
// three token sequences for before, within, and after respectively:
//
//     start, end := partitionTokens(toks, rng)
//     before := toks[:start]
//     within := toks[start:end]
//     after := toks[end:]
//
// This works best when the range is aligned with token boundaries (e.g.
// because it was produced in terms of the scanner's result) but if that isn't
// true then it will make a best effort that may produce strange results at
// the boundaries.
//
// Native zclsyntax tokens are used here, because they contain the necessary
// absolute position information. However, since writerTokens produces a
// correlatable sequence of writer tokens, the resulting indices can be
// used also to index into its result, allowing the partitioning of writer
// tokens to be driven by the partitioning of native tokens.
//
// The tokens are assumed to be in source order and non-overlapping, which
// will be true if the token sequence from the scanner is used directly.
func partitionTokens(toks hclsyntax.Tokens, rng hcl.Range) (start, end int) {
	// We us a linear search here because we assume tha in most cases our
	// target range is close to the beginning of the sequence, and the seqences
	// are generally small for most reasonable files anyway.
	for i := 0; ; i++ {
		if i >= len(toks) {
			// No tokens for the given range at all!
			return len(toks), len(toks)
		}

		if toks[i].Range.Start.Byte >= rng.Start.Byte {
			start = i
			break
		}
	}

	for i := start; ; i++ {
		if i >= len(toks) {
			// The range "hangs off" the end of the token sequence
			return start, len(toks)
		}

		if toks[i].Range.Start.Byte >= rng.End.Byte {
			end = i // end marker is exclusive
			break
		}
	}

	return start, end
}

// partitionLeadCommentTokens takes a sequence of tokens that is assumed
// to immediately precede a construct that can have lead comment tokens,
// and returns the index into that sequence where the lead comments begin.
//
// Lead comments are defined as whole lines containing only comment tokens
// with no blank lines between. If no such lines are found, the returned
// index will be len(toks).
func partitionLeadCommentTokens(toks hclsyntax.Tokens) int {
	// single-line comments (which is what we're interested in here)
	// consume their trailing newline, so we can just walk backwards
	// until we stop seeing comment tokens.
	for i := len(toks) - 1; i >= 0; i-- {
		if toks[i].Type != hclsyntax.TokenComment {
			return i + 1
		}
	}
	return 0
}

// partitionLineEndTokens takes a sequence of tokens that is assumed
// to immediately follow a construct that can have a line comment, and
// returns first the index where any line comments end and then second
// the index immediately after the trailing newline.
//
// Line comments are defined as comments that appear immediately after
// a construct on the same line where its significant tokens ended.
//
// Since single-line comment tokens (# and //) include the newline that
// terminates them, in the presence of these the two returned indices
// will be the same since the comment itself serves as the line end.
func partitionLineEndTokens(toks hclsyntax.Tokens) (afterComment, afterNewline int) {
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if tok.Type != hclsyntax.TokenComment {
			switch tok.Type {
			case hclsyntax.TokenNewline:
				return i, i + 1
			case hclsyntax.TokenEOF:
				// Although this is valid, we mustn't include the EOF
				// itself as our "newline" or else strange things will
				// happen when we try to append new items.
				return i, i
			default:
				// If we have well-formed input here then nothing else should be
				// possible. This path should never happen, because we only try
				// to extract tokens from the sequence if the parser succeeded,
				// and it should catch this problem itself.
				panic("malformed line trailers: expected only comments and newlines")
			}
		}

		if len(tok.Bytes) > 0 && tok.Bytes[len(tok.Bytes)-1] == '\n' {
			// Newline at the end of a single-line comment serves both as
			// the end of comments *and* the end of the line.
			return i + 1, i + 1
		}
	}
	return len(toks), len(toks)
}

// lexConfig uses the zclsyntax scanner to get a token stream and then
// rewrites it into this package's token model.
//
// Any errors produced during scanning are ignored, so the results of this
// function should be used with care.
func lexConfig(src []byte) Tokens {
	mainTokens, _ := hclsyntax.LexConfig(src, "", hcl.Pos{Byte: 0, Line: 1, Column: 1})
	return writerTokens(mainTokens)
}
