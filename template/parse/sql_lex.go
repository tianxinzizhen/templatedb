package parse

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type sqlLexer struct {
	input     string // the string being scanned
	pos       Pos    // current position in the input
	start     Pos    // start position of this item
	atEOF     bool   // we have hit the end of input and returned eof
	item      item
	leftDelim string // start of action marker
}

func (l *sqlLexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.atEOF = true
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += Pos(w)
	return r
}

func (l *sqlLexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *sqlLexer) backup() {
	if !l.atEOF && l.pos > 0 {
		_, w := utf8.DecodeLastRuneInString(l.input[:l.pos])
		l.pos -= Pos(w)
	}
}

func (l *sqlLexer) thisItem(t itemType) item {
	i := item{typ: t, pos: l.start, val: l.input[l.start:l.pos]}
	l.start = l.pos
	return i
}

func (l *sqlLexer) emit(t itemType) stateSqlFn {
	return l.emitItem(l.thisItem(t))
}

// emitItem passes the specified item to the parser.
func (l *sqlLexer) emitItem(i item) stateSqlFn {
	l.item = i
	return nil
}

type stateSqlFn func(*sqlLexer) stateSqlFn

func (l *sqlLexer) nextItem() item {
	l.item = item{typ: itemEOF, pos: l.pos, val: "EOF"}
	state := lexSql
	for {
		state = state(l)
		if state == nil {
			return l.item
		}
	}
}

func lexSql(l *sqlLexer) stateSqlFn {
	if strings.HasPrefix(l.input[l.pos:], l.leftDelim) {
		return l.emit(itemLeftDelim)
	}
	switch r := l.next(); {
	case r == eof:
		return l.emit(itemEOF)
	case isSpace(r):
		for isSpace(l.peek()) {
			l.next()
		}
		return l.emit(itemSpace)
	case r == '"':
		for l.peek() == '"' {
			l.next()
		}
		return l.emit(itemString)
	case r == '`':
		for l.peek() == '`' {
			l.next()
		}
		return l.emit(itemRawString)
	case r == '\'':
		for l.peek() == '\'' {
			l.next()
		}
		return l.emit(itemString)
	case r == '@':
		// special look-ahead for ".field" so we don't break l.backup().
		if l.pos < Pos(len(l.input)) {
			r := l.input[l.pos]
			if r < '0' || '9' < r {
				hasAt := false
				if r == '@' {
					hasAt = true
					l.next()
				}
				for isAlphaNumeric(l.peek()) {
					l.next()
				}
				if hasAt {
					return l.emit(itemIdentifier)
				} else {
					return l.emit(itemField)
				}
			}
		}
	case isAlphaNumeric(r):
		for isAlphaNumeric(l.peek()) {
			l.next()
		}
		return l.emit(itemIdentifier)
	case r == '(':
		return l.emit(itemLeftParen)
	case r == ')':
		return l.emit(itemRightParen)
	case r <= unicode.MaxASCII && unicode.IsPrint(r):
		return l.emit(itemChar)
	default:
		return l.emit(itemChar)
	}
	return nil
}
