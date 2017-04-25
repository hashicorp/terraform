package scanner

import (
	"fmt"

	"github.com/hashicorp/hil/ast"
)

type Token struct {
	Type    TokenType
	Content string
	Pos     ast.Pos
}

//go:generate stringer -type=TokenType
type TokenType rune

const (
	// Raw string data outside of ${ .. } sequences
	LITERAL TokenType = 'o'

	// STRING is like a LITERAL but it's inside a quoted string
	// within a ${ ... } sequence, and so it can contain backslash
	// escaping.
	STRING TokenType = 'S'

	// Other Literals
	INTEGER TokenType = 'I'
	FLOAT   TokenType = 'F'
	BOOL    TokenType = 'B'

	BEGIN    TokenType = '$' // actually "${"
	END      TokenType = '}'
	OQUOTE   TokenType = '“' // Opening quote of a nested quoted sequence
	CQUOTE   TokenType = '”' // Closing quote of a nested quoted sequence
	OPAREN   TokenType = '('
	CPAREN   TokenType = ')'
	OBRACKET TokenType = '['
	CBRACKET TokenType = ']'
	COMMA    TokenType = ','

	IDENTIFIER TokenType = 'i'

	PERIOD  TokenType = '.'
	PLUS    TokenType = '+'
	MINUS   TokenType = '-'
	STAR    TokenType = '*'
	SLASH   TokenType = '/'
	PERCENT TokenType = '%'

	AND  TokenType = '∧'
	OR   TokenType = '∨'
	BANG TokenType = '!'

	EQUAL    TokenType = '='
	NOTEQUAL TokenType = '≠'
	GT       TokenType = '>'
	LT       TokenType = '<'
	GTE      TokenType = '≥'
	LTE      TokenType = '≤'

	QUESTION TokenType = '?'
	COLON    TokenType = ':'

	EOF TokenType = '␄'

	// Produced for sequences that cannot be understood as valid tokens
	// e.g. due to use of unrecognized punctuation.
	INVALID TokenType = '�'
)

func (t *Token) String() string {
	switch t.Type {
	case EOF:
		return "end of string"
	case INVALID:
		return fmt.Sprintf("invalid sequence %q", t.Content)
	case INTEGER:
		return fmt.Sprintf("integer %s", t.Content)
	case FLOAT:
		return fmt.Sprintf("float %s", t.Content)
	case STRING:
		return fmt.Sprintf("string %q", t.Content)
	case LITERAL:
		return fmt.Sprintf("literal %q", t.Content)
	case OQUOTE:
		return fmt.Sprintf("opening quote")
	case CQUOTE:
		return fmt.Sprintf("closing quote")
	case AND:
		return "&&"
	case OR:
		return "||"
	case NOTEQUAL:
		return "!="
	case GTE:
		return ">="
	case LTE:
		return "<="
	default:
		// The remaining token types have content that
		// speaks for itself.
		return fmt.Sprintf("%q", t.Content)
	}
}
