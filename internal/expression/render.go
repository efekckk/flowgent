package expression

import (
	"fmt"
	"regexp"
	"strings"
)

var exprPattern = regexp.MustCompile(`\{\{\s*(.+?)\s*\}\}`)

// RenderValue renders a value:
//   - non-string → returned as-is
//   - string with a single "{{ ... }}" and nothing else → returns the raw
//     evaluation result (preserves the expression's type, e.g. int / map)
//   - string with embedded "{{ ... }}" expressions → returns a string
//     concatenation, stringifying each piece with fmt.Sprint
func RenderValue(template any, ctx EvalContext) (any, error) {
	s, ok := template.(string)
	if !ok {
		return template, nil
	}
	matches := exprPattern.FindAllStringSubmatchIndex(s, -1)
	if len(matches) == 0 {
		return s, nil
	}
	// single-expression: entire string is one match
	if len(matches) == 1 && matches[0][0] == 0 && matches[0][1] == len(s) {
		inner := s[matches[0][2]:matches[0][3]]
		return Evaluate(inner, ctx)
	}
	// multi-piece: stringify
	var b strings.Builder
	pos := 0
	for _, m := range matches {
		b.WriteString(s[pos:m[0]])
		inner := s[m[2]:m[3]]
		v, err := Evaluate(inner, ctx)
		if err != nil {
			return nil, err
		}
		b.WriteString(fmt.Sprint(v))
		pos = m[1]
	}
	b.WriteString(s[pos:])
	return b.String(), nil
}
