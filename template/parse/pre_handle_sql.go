package parse

import "strings"

func handleInsertColumns(input, left, right string, hasFunction func(name string) bool) (columns []string, body string, pos Pos) {
	bodySb := strings.Builder{}
	l := newPreLex(input, left, right)
	leftParen := false
	for l.nextItem().typ != itemEOF {
		if l.item.typ == itemLeftDelim {
			_, body, pos := handleFiledName(l.input[l.pos:], left, right, hasFunction)
			bodySb.WriteString(body)
			l.pos += pos
			l.start += pos
			continue
		}
		switch l.item.typ {
		case itemIdentifier:
			switch strings.ToLower(l.item.val) {
			case "value", "values":
				bodySb.WriteString(l.item.val)
				return columns, bodySb.String(), l.pos
			default:
				bodySb.WriteString(l.item.val)
			}
			if leftParen {
				columns = append(columns, l.item.val)
			}
		case itemLeftParen:
			leftParen = true
			bodySb.WriteString(l.item.val)
		default:
			bodySb.WriteString(l.item.val)
		}
	}
	return columns, bodySb.String(), l.pos
}

func handleInsertValues(input, left, right string, hasFunction func(name string) bool, columns []string) (body string, pos Pos) {
	bodySb := strings.Builder{}
	l := newPreLex(input, left, right)
	i := 0
	leftParen := 0
	for l.nextItem().typ != itemEOF {
		if l.item.typ == itemLeftDelim {
			_, body, pos := handleFiledName(l.input[l.pos:], left, right, hasFunction)
			bodySb.WriteString(body)
			l.pos += pos
			l.start += pos
			continue
		}
		switch l.item.typ {
		case itemChar:
			switch l.item.val {
			case ",":
				if leftParen == 1 {
					i++
				}
				bodySb.WriteString(l.item.val)
			case "?":
				bodySb.WriteString(left)
				bodySb.WriteByte('.')
				bodySb.WriteString(columns[i])
				bodySb.WriteString(right)
			}
		case itemField:
			bodySb.WriteString(left)
			bodySb.WriteString(l.item.val)
			bodySb.WriteString(right)
		case itemLeftParen:
			leftParen++
			bodySb.WriteString(l.item.val)
		case itemRightParen:
			leftParen--
			bodySb.WriteString(l.item.val)
			if leftParen == 0 {
				return bodySb.String(), l.pos
			}
		default:
			bodySb.WriteString(l.item.val)
		}
	}
	return bodySb.String(), l.pos
}
