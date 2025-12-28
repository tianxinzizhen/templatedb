package parse

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

func handleFiledName(input, left, right string, hasFunction func(name string) bool) (cond, body string, pos int) {
	condSb := strings.Builder{}
	bodySb := strings.Builder{}
	l := &lexer{
		name:         "",
		input:        input,
		leftDelim:    left,
		rightDelim:   right,
		line:         1,
		startLine:    1,
		insideAction: false,
	}
	useMuiltiFieldOutput := true
	useFunc := false
	for l.nextItem().typ != itemEOF {
		if l.item.typ == itemRightDelim {
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
				condSb.WriteString(l.item.val)
				condSb.WriteRune(' ')
			}
		} else if useFunc {
			switch l.item.typ {
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
	return condSb.String(), bodySb.String(), int(l.item.pos)
}

func handleOption(input string, left, right string, hasFunction func(name string) bool) (cond, body string) {
	condSb := strings.Builder{}
	bodySb := strings.Builder{}
	pk := ""
	psb := strings.Builder{}
	for i := 0; i < len(input); {
		if strings.HasPrefix(input[i:], left) {
			x := strings.Index(input[i:], right)
			if x != -1 {
				cond, body, pos := handleFiledName(input[i:], left, right, hasFunction)
				condSb.WriteString(cond)
				bodySb.WriteString(body)
				i += pos + len(right)
			}
		}
		v, size := utf8.DecodeRuneInString(input[i:])
		i += size
		switch {
		case v == '@':
			// skip @@
			v, size = utf8.DecodeRuneInString(input[i:])
			if v == '@' {
				bodySb.WriteString("@@")
				i += size
				continue
			}
			dot := true
			bodySb.WriteString(left)
			for i < len(input) {
				if v == ':' {
					bodySb.WriteString("json ")
					i += size
				} else if isAlphaNumeric(v) {
					if dot {
						dot = false
						condSb.WriteByte('.')
						bodySb.WriteByte('.')
					}
					condSb.WriteRune(v)
					bodySb.WriteRune(v)
					i += size
				} else {
					break
				}
				v, size = utf8.DecodeRuneInString(input[i:])
			}
			condSb.WriteRune(' ')
			bodySb.WriteString(right)
		case v == '?':
			bodySb.WriteString(left)
			v, size = utf8.DecodeRuneInString(input[i:])
			if v == ':' {
				i += size
				bodySb.WriteString("json ")
			}
			bodySb.WriteString("." + pk)
			condSb.WriteString("." + pk)
			condSb.WriteRune(' ')
			bodySb.WriteString(right)
		case isAlphaNumeric(v):
			bodySb.WriteRune(v)
			psb.WriteRune(v)
		default:
			if isSpace(v) {
				bodySb.WriteRune(' ')
				v, size = utf8.DecodeRuneInString(input[i:])
				for isSpace(v) {
					i += size
					v, size = utf8.DecodeRuneInString(input[i:])
				}
			} else {
				if psb.String() != "" {
					pk = psb.String()
				}
				psb.Reset()
				bodySb.WriteRune(v)
			}
		}
	}
	return condSb.String(), bodySb.String()
}

// 处理input中@信息，将其替换为left+filedName+right
func handleAtsign(input, left, right string, hasFunction func(name string) bool) string {
	isb := strings.Builder{}
	pk := ""
	psb := strings.Builder{}
	for i := 0; i < len(input); {
		if strings.HasPrefix(input[i:], left) {
			_, body, pos := handleFiledName(input[i:], left, right, hasFunction)
			isb.WriteString(body)
			i += pos + len(right)
		}
		v, size := utf8.DecodeRuneInString(input[i:])
		i += size
		switch {
		case v == '@':
			// skip @@
			v, size = utf8.DecodeRuneInString(input[i:])
			if v == '@' {
				isb.WriteString("@@")
				i += size
				continue
			}
			dot := true
			isb.WriteString(left)
			for i < len(input) {
				if v == ':' {
					isb.WriteString("json ")
					i += size
				} else if isAlphaNumeric(v) {
					if dot {
						dot = false
						isb.WriteByte('.')
					}
					isb.WriteRune(v)
					i += size
				} else {
					break
				}
				v, size = utf8.DecodeRuneInString(input[i:])
			}
			isb.WriteString(right)
		case v == '[':
			x := strings.IndexByte(input[i:], ']')
			if x == -1 {
				isb.WriteRune(v)
			} else {
				cond, body := handleOption(input[i:i+x], left, right, hasFunction)
				hasCond := strings.TrimSpace(cond) != ""
				if hasCond {
					isb.WriteString(left)
					isb.WriteString("if and ")
					isb.WriteString(cond)
					isb.WriteString(right)
				}
				isb.WriteString(body)
				if hasCond {
					isb.WriteString(left)
					isb.WriteString("end")
					isb.WriteString(right)
				}
				i += x + 1 // skip ']'
			}
		case v == '?':
			isb.WriteString(left)
			v, size = utf8.DecodeRuneInString(input[i:])
			if v == ':' {
				i += size
				isb.WriteString("json ")
			}
			isb.WriteString("." + pk)
			isb.WriteString(right)
		case isAlphaNumeric(v):
			isb.WriteRune(v)
			psb.WriteRune(v)
		default:
			if isSpace(v) {
				isb.WriteRune(' ')
				v, size = utf8.DecodeRuneInString(input[i:])
				for isSpace(v) {
					i += size
					v, size = utf8.DecodeRuneInString(input[i:])
				}
			} else {
				if psb.String() != "" {
					pk = psb.String()
				}
				psb.Reset()
				isb.WriteRune(v)
			}
		}
	}
	return isb.String()
}
