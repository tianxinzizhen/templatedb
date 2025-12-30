package parse

import (
	"fmt"
	"strconv"
	"strings"
)

func newPreLex(input, left, right string) *sqlLexer {
	return &sqlLexer{
		input:     input,
		leftDelim: left,
	}
}

func newLex(input, left, right string) *lexer {
	return &lexer{
		input:      input,
		leftDelim:  left,
		rightDelim: right,
	}
}

func handleFiledName(input, left, right string, hasFunction func(name string) bool) (cond, body string, pos Pos) {
	condSb := strings.Builder{}
	bodySb := strings.Builder{}
	l := newLex(input, left, right)
	useMuiltiFieldOutput := true
	useFunc := false
	for l.nextItem().typ != itemEOF {
		if l.item.typ == itemError {
			panic(fmt.Errorf("template: %s:%d: %s", l.name, l.line, l.item.val))
		} else if l.item.typ == itemRightDelim {
			bodySb.WriteString(l.item.val)
			break
		} else if l.item.typ > itemKeyword {
			useMuiltiFieldOutput = false
		}
		if useMuiltiFieldOutput {
			switch l.item.typ {
			case itemIdentifier:
				if !hasFunction(l.item.val) {
					condSb.WriteString(" ." + l.item.val)
					bodySb.WriteRune('.')
				} else {
					useMuiltiFieldOutput = false
					useFunc = true
				}
			case itemChar:
				if l.item.val == "," {
					bodySb.WriteString(right)
					bodySb.WriteString(l.item.val)
					bodySb.WriteString(left)
					continue
				}
			case itemCharConstant:
				_, _, tail, err := strconv.UnquoteChar(l.item.val[1:], l.item.val[0])
				if err != nil {
					return "", "", 0
				}
				if tail != "'" {
					l.item.val = fmt.Sprintf(`"%s"`, l.item.val[1:len(l.item.val)-1])
				}
			case itemField:
				condSb.WriteRune(' ')
				condSb.WriteString(l.item.val)
			}
		} else if useFunc {
			switch l.item.typ {
			case itemIdentifier:
				if !hasFunction(l.item.val) {
					condSb.WriteString(" ." + l.item.val)
					bodySb.WriteRune('.')
				}
			case itemField:
				condSb.WriteRune(' ')
				condSb.WriteString(l.item.val)
			case itemChar:
				if l.item.val == "," {
					l.item.val = " "
				}
			}
		}
		bodySb.WriteString(l.item.val)
	}
	if !useMuiltiFieldOutput {
		condSb.Reset()
	}
	return condSb.String(), bodySb.String(), l.pos
}

func handleOption(input string, left, right string, hasFunction func(name string) bool) (cond, body string, pos Pos) {
	condSb := strings.Builder{}
	bodySb := strings.Builder{}
	l := newPreLex(input, left, right)
	leftParen := 0
	preKey := ""
	for l.nextItem().typ != itemEOF {
		if l.item.typ == itemLeftDelim {
			cond, body, pos := handleFiledName(l.input[l.item.pos:], left, right, hasFunction)
			condSb.WriteString(cond)
			bodySb.WriteString(body)
			l.pos += pos
			l.start += pos
			continue
		}
		switch l.item.typ {
		case itemChar:
			switch l.item.val {
			case "[":
				leftParen++
			case "]":
				leftParen--
				if leftParen == 0 {
					return condSb.String(), bodySb.String(), l.pos
				}
			case "?":
				condSb.WriteString(" ." + preKey)
				bodySb.WriteString(left)
				bodySb.WriteByte('.')
				bodySb.WriteString(preKey)
				bodySb.WriteString(right)
			default:
				bodySb.WriteString(l.item.val)
			}
		case itemField:
			condSb.WriteString(l.item.val)
			bodySb.WriteString(left)
			bodySb.WriteString(l.item.val)
			bodySb.WriteString(right)
		case itemIdentifier:
			preKey = l.item.val
			bodySb.WriteString(l.item.val)
		default:
			bodySb.WriteString(l.item.val)
		}
	}
	return condSb.String(), bodySb.String(), l.pos
}

// 处理input中@信息，将其替换为left+filedName+right
func handleAtsign(input, left, right string, hasFunction func(name string) bool) (body string) {
	bodySb := strings.Builder{}
	l := newPreLex(input, left, right)
	preKey := ""
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
			case "insert":
				bodySb.WriteString(l.item.val)
				columns, body, pos := handleInsertColumns(l.input[l.pos:], left, right, hasFunction)
				l.pos += pos
				l.start += pos
				bodySb.WriteString(body)
				if len(columns) > 0 {
					body, pos := handleInsertValues(l.input[l.pos:], left, right, hasFunction, columns)
					l.pos += pos
					l.start += pos
					bodySb.WriteString(body)
				}
			default:
				preKey = l.item.val
				bodySb.WriteString(l.item.val)
			}
		case itemField:
			bodySb.WriteString(left)
			bodySb.WriteString(l.item.val)
			bodySb.WriteString(right)
		case itemChar:
			switch l.item.val {
			case "?":
				bodySb.WriteString(left)
				bodySb.WriteString(" ." + preKey)
				bodySb.WriteString(right)
			case "[":
				cond, body, pos := handleOption(l.input[l.pos:], left, right, hasFunction)
				l.pos += pos
				l.start += pos
				hasCond := strings.TrimSpace(cond) != ""
				if hasCond {
					bodySb.WriteString(left)
					bodySb.WriteString("if and ")
					bodySb.WriteString(cond)
					bodySb.WriteString(right)
				}
				bodySb.WriteString(body)
				if hasCond {
					bodySb.WriteString(left)
					bodySb.WriteString("end")
					bodySb.WriteString(right)
				}
			default:
				bodySb.WriteString(l.item.val)
			}
		case itemSpace:
			bodySb.WriteString(" ")
		default:
			bodySb.WriteString(l.item.val)
		}
	}
	return bodySb.String()
}
