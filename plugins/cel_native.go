package plugins

import (
	"net"
	"strconv"
	"strings"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	"github.com/threatwinds/go-sdk/jsondoc"
	"github.com/tidwall/gjson"
)

// nativePred is an expression compiled to Go. It answers exactly what the CEL overloads answer
// for the same expression and document.
type nativePred func(d *jsondoc.Doc) bool

// compileNative turns an expression made only of the string-keyed overloads, literals and the
// boolean operators into a Go predicate. It reports false for anything else (event variables,
// arithmetic, member calls, overloads not covered here), and the caller then keeps using CEL.
func compileNative(expression string) (nativePred, bool) {
	env, err := cel.NewEnv()
	if err != nil {
		return nil, false
	}

	parsed, issues := env.Parse(expression)
	if issues != nil && issues.Err() != nil {
		return nil, false
	}

	return buildPred(parsed.NativeRep().Expr())
}

func buildPred(e ast.Expr) (nativePred, bool) {
	switch e.Kind() {
	case ast.LiteralKind:
		b, ok := e.AsLiteral().Value().(bool)
		if !ok {
			return nil, false
		}

		return func(*jsondoc.Doc) bool { return b }, true

	case ast.CallKind:
		call := e.AsCall()
		if call.IsMemberFunction() {
			return nil, false
		}

		switch call.FunctionName() {
		case "_&&_", "_||_":
			if len(call.Args()) != 2 {
				return nil, false
			}

			l, ok1 := buildPred(call.Args()[0])
			r, ok2 := buildPred(call.Args()[1])
			if !ok1 || !ok2 {
				return nil, false
			}

			if call.FunctionName() == "_&&_" {
				return func(d *jsondoc.Doc) bool { return l(d) && r(d) }, true
			}

			return func(d *jsondoc.Doc) bool { return l(d) || r(d) }, true

		case "!_":
			if len(call.Args()) != 1 {
				return nil, false
			}

			p, ok := buildPred(call.Args()[0])
			if !ok {
				return nil, false
			}

			return func(d *jsondoc.Doc) bool { return !p(d) }, true
		}

		return buildOverload(call.FunctionName(), call.Args())
	}

	return nil, false
}

func stringLit(e ast.Expr) (string, bool) {
	if e.Kind() != ast.LiteralKind {
		return "", false
	}

	s, ok := e.AsLiteral().Value().(string)

	return s, ok
}

// scalarLit returns a literal as CEL hands it to an overload: string, int64, float64 or bool.
func scalarLit(e ast.Expr) (any, bool) {
	if e.Kind() != ast.LiteralKind {
		return nil, false
	}

	switch v := e.AsLiteral().Value().(type) {
	case string, int64, float64, bool:
		return v, true
	}

	return nil, false
}

func scalarList(e ast.Expr) ([]any, bool) {
	if e.Kind() != ast.ListKind {
		return nil, false
	}

	var out []any
	for _, el := range e.AsList().Elements() {
		v, ok := scalarLit(el)
		if !ok {
			return nil, false
		}

		out = append(out, v)
	}

	return out, true
}

func stringList(e ast.Expr) ([]string, bool) {
	items, ok := scalarList(e)
	if !ok {
		return nil, false
	}

	out := make([]string, len(items))
	for i, it := range items {
		s, ok := it.(string)
		if !ok {
			return nil, false
		}

		out[i] = strings.TrimSpace(s)
	}

	return out, true
}

func buildOverload(name string, args []ast.Expr) (nativePred, bool) {
	if len(args) < 1 {
		return nil, false
	}

	key, ok := stringLit(args[0])
	if !ok {
		return nil, false
	}

	if name == "exists" && len(args) == 1 {
		return func(d *jsondoc.Doc) bool { return d.Get(key).Exists() }, true
	}

	if len(args) != 2 {
		return nil, false
	}

	arg := args[1]

	switch name {
	case "equals":
		t, ok := scalarLit(arg)
		if !ok {
			return nil, false
		}

		tv := types.DefaultTypeAdapter.NativeToValue(t)

		return func(d *jsondoc.Doc) bool { return flexibleMatch(d.Get(key), tv) }, true

	case "oneOf":
		items, ok := scalarList(arg)
		if !ok {
			return nil, false
		}

		vals := make([]ref.Val, len(items))
		for i, it := range items {
			vals[i] = types.DefaultTypeAdapter.NativeToValue(it)
		}

		return func(d *jsondoc.Doc) bool {
			v := d.Get(key)
			if !v.Exists() {
				return false
			}

			for _, it := range vals {
				if flexibleMatch(v, it) {
					return true
				}
			}

			return false
		}, true

	case "equalsIgnoreCase":
		s, ok := stringLit(arg)
		if !ok {
			return nil, false
		}

		return func(d *jsondoc.Doc) bool {
			v := d.Get(key)

			return v.Exists() && v.Type == gjson.String && strings.EqualFold(v.String(), s)
		}, true

	case "contains", "startsWith", "endsWith":
		return textSearch(name, key, arg)

	case "containsAll":
		items, ok := stringList(arg)
		if !ok {
			return nil, false
		}

		return func(d *jsondoc.Doc) bool {
			s, ok := textContent(d.Get(key))
			if !ok {
				return false
			}

			for _, it := range items {
				if !strings.Contains(s, it) {
					return false
				}
			}

			return true
		}, true

	case "inCIDR":
		network, ok := stringLit(arg)
		if !ok {
			return nil, false
		}

		_, subnet, err := net.ParseCIDR(network)

		return func(d *jsondoc.Doc) bool {
			v := d.Get(key)
			if err != nil || !v.Exists() || v.Type != gjson.String {
				return false
			}

			ip := net.ParseIP(v.String())

			return ip != nil && subnet.Contains(ip)
		}, true

	case "regexMatch":
		pattern, ok := stringLit(arg)
		if !ok {
			return nil, false
		}

		re, err := rCache.Get(pattern)

		return func(d *jsondoc.Doc) bool {
			v := d.Get(key)
			if err != nil || !v.Exists() || v.Type != gjson.String {
				return false
			}

			return re.MatchString(v.Str)
		}, true

	case "lessOrEqual", "greaterOrEqual", "lessThan", "greaterThan":
		t, ok := scalarLit(arg)
		if !ok {
			return nil, false
		}

		f2, ok := litFloat(t)
		if !ok {
			return nil, false
		}

		cmp := map[string]func(a, b float64) bool{
			"lessOrEqual":    func(a, b float64) bool { return a <= b },
			"greaterOrEqual": func(a, b float64) bool { return a >= b },
			"lessThan":       func(a, b float64) bool { return a < b },
			"greaterThan":    func(a, b float64) bool { return a > b },
		}[name]

		return func(d *jsondoc.Doc) bool {
			f1, ok := getNumeric(d.Get(key))

			return ok && cmp(f1, f2)
		}, true
	}

	return nil, false
}

func textSearch(name, key string, arg ast.Expr) (nativePred, bool) {
	var match func(s, sub string) bool

	switch name {
	case "contains":
		match = strings.Contains
	case "startsWith":
		match = strings.HasPrefix
	default:
		match = strings.HasSuffix
	}

	if s, ok := stringLit(arg); ok {
		return func(d *jsondoc.Doc) bool {
			text, ok := textContent(d.Get(key))

			return ok && match(text, s)
		}, true
	}

	items, ok := stringList(arg)
	if !ok {
		return nil, false
	}

	return func(d *jsondoc.Doc) bool {
		text, ok := textContent(d.Get(key))
		if !ok {
			return false
		}

		for _, it := range items {
			if match(text, it) {
				return true
			}
		}

		return false
	}, true
}

func litFloat(t any) (float64, bool) {
	switch v := t.(type) {
	case float64:
		return v, true
	case int64:
		return float64(v), true
	case string:
		f, err := strconv.ParseFloat(v, 64)

		return f, err == nil
	}

	return 0, false
}
