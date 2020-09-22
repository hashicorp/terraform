package tagparser

import (
	"github.com/vmihailenco/tagparser/internal/parser"
)

type Tag struct {
	Name    string
	Options map[string]string
}

func (t *Tag) HasOption(name string) bool {
	_, ok := t.Options[name]
	return ok
}

func Parse(s string) *Tag {
	p := &tagParser{
		Parser: parser.NewString(s),
	}
	p.parseKey()
	return &p.Tag
}

type tagParser struct {
	*parser.Parser

	Tag     Tag
	hasName bool
	key     string
}

func (p *tagParser) setTagOption(key, value string) {
	if !p.hasName {
		p.hasName = true
		if key == "" {
			p.Tag.Name = value
			return
		}
	}
	if p.Tag.Options == nil {
		p.Tag.Options = make(map[string]string)
	}
	if key == "" {
		p.Tag.Options[value] = ""
	} else {
		p.Tag.Options[key] = value
	}
}

func (p *tagParser) parseKey() {
	p.key = ""

	var b []byte
	for p.Valid() {
		c := p.Read()
		switch c {
		case ',':
			p.Skip(' ')
			p.setTagOption("", string(b))
			p.parseKey()
			return
		case ':':
			p.key = string(b)
			p.parseValue()
			return
		case '\'':
			p.parseQuotedValue()
			return
		default:
			b = append(b, c)
		}
	}

	if len(b) > 0 {
		p.setTagOption("", string(b))
	}
}

func (p *tagParser) parseValue() {
	const quote = '\''

	c := p.Peek()
	if c == quote {
		p.Skip(quote)
		p.parseQuotedValue()
		return
	}

	var b []byte
	for p.Valid() {
		c = p.Read()
		switch c {
		case '\\':
			b = append(b, p.Read())
		case '(':
			b = append(b, c)
			b = p.readBrackets(b)
		case ',':
			p.Skip(' ')
			p.setTagOption(p.key, string(b))
			p.parseKey()
			return
		default:
			b = append(b, c)
		}
	}
	p.setTagOption(p.key, string(b))
}

func (p *tagParser) readBrackets(b []byte) []byte {
	var lvl int
loop:
	for p.Valid() {
		c := p.Read()
		switch c {
		case '\\':
			b = append(b, p.Read())
		case '(':
			b = append(b, c)
			lvl++
		case ')':
			b = append(b, c)
			lvl--
			if lvl < 0 {
				break loop
			}
		default:
			b = append(b, c)
		}
	}
	return b
}

func (p *tagParser) parseQuotedValue() {
	const quote = '\''

	var b []byte
	b = append(b, quote)

	for p.Valid() {
		bb, ok := p.ReadSep(quote)
		if !ok {
			b = append(b, bb...)
			break
		}

		if len(bb) > 0 && bb[len(bb)-1] == '\\' {
			b = append(b, bb[:len(bb)-1]...)
			b = append(b, quote)
			continue
		}

		b = append(b, bb...)
		b = append(b, quote)
		break
	}

	p.setTagOption(p.key, string(b))
	if p.Skip(',') {
		p.Skip(' ')
	}
	p.parseKey()
}

func Unquote(s string) (string, bool) {
	const quote = '\''

	if len(s) < 2 {
		return s, false
	}
	if s[0] == quote && s[len(s)-1] == quote {
		return s[1 : len(s)-1], true
	}
	return s, false
}
