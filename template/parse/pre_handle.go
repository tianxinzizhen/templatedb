package parse

import (
	"strings"
	"unicode/utf8"
)

func handleFiledName(input, left, right string, hasFunction func(name string) bool) (cond, body string) {
	condSb := strings.Builder{}
	sb := strings.Builder{}
	for i := 0; i < len(input); {
		v, size := utf8.DecodeRuneInString(input[i:])
		if isAlphaNumeric(v) {
			identifier := strings.Builder{}
			for isAlphaNumeric(v) {
				identifier.WriteRune(v)
				i += size
				v, size = utf8.DecodeRuneInString(input[i:])
			}
			name := identifier.String()
			if !hasFunction(name) {
				// not key
				if _, ok := key[name]; !ok {
					sb.WriteString(".")
					sb.WriteString(name)
					condSb.WriteString(".")
					condSb.WriteString(name)
					continue
				} else {
					return "", input
				}
			}
			condSb.WriteRune(' ')
			sb.WriteString(name)
		} else if v == '.' || v == '@' {
			sb.WriteRune(v)
			condSb.WriteRune(v)
			i += size
			v, size = utf8.DecodeRuneInString(input[i:])
			for isAlphaNumeric(v) {
				sb.WriteRune(v)
				condSb.WriteRune(v)
				i += size
				v, size = utf8.DecodeRuneInString(input[i:])
			}
			condSb.WriteRune(' ')
		} else if v == ',' {
			i += size
			sb.WriteString(right)
			sb.WriteRune(',')
			sb.WriteString(left)
		} else if isSpace(v) {
			sb.WriteRune(' ')
			for isSpace(v) {
				i += size
				v, size = utf8.DecodeRuneInString(input[i:])
			}
		} else {
			sb.WriteRune(v)
			i += size
		}
	}
	return condSb.String(), sb.String()
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
				cond, body := handleFiledName(input[i+len(left):i+x], left, right, hasFunction)
				condSb.WriteString(cond)
				bodySb.WriteString(left)
				bodySb.WriteString(body)
				bodySb.WriteString(right)
				i += x + len(right)
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
			x := strings.Index(input[i:], right)
			if x != -1 {
				_, body := handleFiledName(input[i+len(left):i+x], left, right, hasFunction)
				isb.WriteString(left)
				isb.WriteString(body)
				isb.WriteString(right)
				i += x + len(right)
			}
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
				isb.WriteString(left)
				isb.WriteString("if and ")
				isb.WriteString(cond)
				isb.WriteString(right)
				isb.WriteString(body)
				isb.WriteString(left)
				isb.WriteString("end")
				isb.WriteString(right)
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
