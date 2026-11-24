package plugins

import (
	"errors"

	"cel.dev/cel-go/cel"
	"github.com/threatwinds/go-sdk/catcher"
	"github.com/threatwinds/go-sdk/jsondoc"
)

// maxLeanExpressions bounds the compiled-expression cache for callers that feed in dynamic expressions.
const maxLeanExpressions = 20000

type leanEntry struct {
	// prg is nil when the expression needs event variables and must go through Evaluate.
	prg cel.Program
	// pred is the expression compiled to Go when it only uses what compileNative understands.
	pred nativePred
}

// EvaluateJSON evaluates an expression against a JSON document without building the values map.
//
// Expressions that only call the string-keyed overloads (exists, equals, oneOf, ...) read the
// document through `_data_`, so they need neither the per-key variables nor the parsed map that
// Evaluate builds on every call. Those are compiled once against a minimal environment and cached
// by expression text. An expression that references an event variable does not compile there and
// is evaluated through Evaluate, so results are the same either way.
func (c *CELCache) EvaluateJSON(data *string, expression string) (bool, error) {
	if data == nil {
		return false, catcher.Error("failed to evaluate CEL expression", errors.New("data pointer is nil"), map[string]any{
			"process": c.processName,
		})
	}

	if v, ok := c.lean.Load(expression); ok {
		return c.runLean(v.(*leanEntry), data, expression)
	}

	entry := &leanEntry{prg: c.compileLean(expression)}
	if c.leanSize.Load() < maxLeanExpressions {
		if _, loaded := c.lean.LoadOrStore(expression, entry); !loaded {
			c.leanSize.Add(1)
		}
	}

	return c.runLean(entry, data, expression)
}

// EvaluateDoc is EvaluateJSON for a document that is already parsed: the overloads read the
// parsed tree directly, so nothing is serialized or scanned.
func (c *CELCache) EvaluateDoc(doc *jsondoc.Doc, expression string) (bool, error) {
	if doc == nil {
		return false, catcher.Error("failed to evaluate CEL expression", errors.New("document is nil"), map[string]any{
			"process": c.processName,
		})
	}

	v, ok := c.lean.Load(expression)
	if !ok {
		entry := &leanEntry{prg: c.compileLean(expression)}
		if entry.prg != nil {
			entry.pred, _ = compileNative(expression)
		}
		if c.leanSize.Load() < maxLeanExpressions {
			if _, loaded := c.lean.LoadOrStore(expression, entry); !loaded {
				c.leanSize.Add(1)
			}
		}
		v = entry
	}

	entry := v.(*leanEntry)
	if entry.pred != nil {
		return entry.pred(doc), nil
	}

	if entry.prg == nil {
		text := doc.String()
		return c.Evaluate(&text, expression)
	}

	return c.execute(entry.prg, map[string]interface{}{"_data_": docVal{doc}})
}

func (c *CELCache) runLean(entry *leanEntry, data *string, expression string) (bool, error) {
	if entry.prg == nil {
		return c.Evaluate(data, expression)
	}

	return c.execute(entry.prg, map[string]interface{}{"_data_": *data})
}

func (c *CELCache) compileLean(expression string) cel.Program {
	env, err := cel.NewEnv(c.baseEnvOptions()...)
	if err != nil {
		return nil
	}

	ast, issues := env.Compile(c.transformExpression(expression))
	if issues != nil && issues.Err() != nil {
		return nil
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil
	}

	return prg
}

// DocExpression is an expression prepared once for evaluation against parsed documents. Holding it
// avoids looking the expression up by its text on every evaluation.
type DocExpression struct {
	cache *CELCache
	text  string
	entry *leanEntry
}

// PrepareDoc prepares expression for EvaluateDoc-style evaluation.
func (c *CELCache) PrepareDoc(expression string) *DocExpression {
	entry := &leanEntry{prg: c.compileLean(expression)}
	if entry.prg != nil {
		entry.pred, _ = compileNative(expression)
	}

	return &DocExpression{cache: c, text: expression, entry: entry}
}

// Eval gives the same answer as CELCache.EvaluateDoc for the prepared expression.
func (e *DocExpression) Eval(doc *jsondoc.Doc) (bool, error) {
	if doc == nil {
		return e.cache.EvaluateDoc(doc, e.text)
	}

	if e.entry.pred != nil {
		return e.entry.pred(doc), nil
	}

	if e.entry.prg == nil {
		text := doc.String()

		return e.cache.Evaluate(&text, e.text)
	}

	return e.cache.execute(e.entry.prg, map[string]interface{}{"_data_": docVal{doc}})
}
