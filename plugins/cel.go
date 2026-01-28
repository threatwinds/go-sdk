package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/threatwinds/go-sdk/catcher"
	"github.com/threatwinds/go-sdk/utils"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"reflect"
	"time"
)

type CELCache struct {
	cache       *expirable.LRU[string, *cachedProgram]
	once        sync.Once
	locks       [1024]sync.Mutex
	processName string
}

func NewCELCache(processName string) *CELCache {
	return &CELCache{
		processName: processName,
	}
}

func (c *CELCache) getLock(key string) *sync.Mutex {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return &c.locks[h.Sum32()%1024]
}

func (c *CELCache) Get(cacheKey string, expression string, valuesMap map[string]interface{}, envOption ...cel.EnvOption) (bool, error) {
	c.once.Do(func() {
		c.cache = expirable.NewLRU[string, *cachedProgram](10000, nil, time.Hour*24)
	})

	if cp, ok := c.cache.Get(cacheKey); ok {
		return c.execute(cp.prg, valuesMap)
	}

	// Use a sharded lock to prevent multiple simultaneous compilations of the same expression
	lock := c.getLock(cacheKey)
	lock.Lock()
	defer lock.Unlock()

	// Double-check the cache after acquiring the lock
	if cp, ok := c.cache.Get(cacheKey); ok {
		return c.execute(cp.prg, valuesMap)
	}

	envOptions := []cel.EnvOption{
		cel.Variable("_data_", cel.StringType),
		c.celExists(),
		c.safe(),
		c.inCIDR(),
		c.equals(),
		c.equalsIgnoreCase(),
		c.contains(),
		c.containsAll(),
		c.oneOf(),
		c.startsWith(),
		c.endsWith(),
		c.regexMatch(),
		c.lessThan(),
		c.greaterThan(),
		c.lessOrEqual(),
		c.greaterOrEqual(),
		c.isHour(),
		c.isMinute(),
		c.isDayOfWeek(),
		c.isWeekend(),
		c.isWorkDay(),
		c.isBetweenTime(),
	}

	// Add the provided environment options first (including cel.Types)
	envOptions = append(envOptions, envOption...)

	for k, v := range valuesMap {
		if k == "_data_" {
			continue
		}
		envOptions = append(envOptions, cel.Variable(k, c.valueToCelType(v)))
	}

	celEnv, err := cel.NewEnv(envOptions...)
	if err != nil {
		return false, catcher.Error("failed to start CEL environment", err, map[string]any{})
	}

	transformedExpr := c.transformExpression(expression)
	ast, issues := celEnv.Compile(transformedExpr)
	if issues != nil && issues.Err() != nil {
		return false, catcher.Error("failed to compile expression", errors.New("consult issues list for more information"), map[string]any{
			"issues":     issues.Errors(),
			"process":    c.processName,
			"expression": expression,
		})
	}

	prg, err := celEnv.Program(ast)
	if err != nil {
		return false, catcher.Error("failed to create program", err, map[string]any{
			"process":    c.processName,
			"expression": expression,
		})
	}

	c.cache.Add(cacheKey, &cachedProgram{prg: prg, env: celEnv})

	return c.execute(prg, valuesMap)
}

var rCache = new(RegexpCache)

type cachedProgram struct {
	prg cel.Program
	env *cel.Env
}

func (c *CELCache) getCacheKey(expression string, valuesMap map[string]interface{}) string {
	keys := make([]string, 0, len(valuesMap))
	for k, v := range valuesMap {
		if k == "_data_" {
			continue
		}
		keys = append(keys, fmt.Sprintf("%s:%T", k, v))
	}
	sort.Strings(keys)

	h := sha256.New()
	h.Write([]byte(expression))
	h.Write([]byte(strings.Join(keys, ",")))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *CELCache) transformExpression(expression string) string {
	overloads := []string{
		"exists", "safe", "inCIDR", "equals", "equalsIgnoreCase", "contains",
		"containsAll", "oneOf", "startsWith", "endsWith", "regexMatch",
		"lessThan", "greaterThan", "lessOrEqual", "greaterOrEqual",
		"isHour", "isMinute", "isDayOfWeek", "isWeekend", "isWorkDay", "isBetweenTime",
	}

	for _, f := range overloads {
		oldCall := f + "("
		newCall := f + "(_data_,"
		if strings.Contains(expression, oldCall) {
			expression = strings.ReplaceAll(expression, oldCall, newCall)
		}
	}
	return expression
}

// Eval evaluates a CEL expression against the given data and returns the boolean result if successful.
// 'data' can be a proto.Message, a JSON string (or *string), or a map[string]any.
func (c *CELCache) Eval(expression string, data any, envOption ...cel.EnvOption) (bool, error) {
	if data == nil {
		return false, catcher.Error("failed to evaluate CEL expression", errors.New("required parameter 'data' is nil"), map[string]any{
			"process": c.processName,
		})
	}

	var valuesMap map[string]interface{}
	var jsonData string

	switch v := data.(type) {
	case proto.Message:
		jsonPtr, err := utils.ProtoMessageToString(v)
		if err != nil {
			return false, catcher.Error("failed to convert proto to string for CEL evaluation", err, map[string]any{
				"process": c.processName,
			})
		}
		jsonData = *jsonPtr
		err = json.Unmarshal([]byte(jsonData), &valuesMap)
		if err != nil {
			return false, catcher.Error("failed to convert proto to map for CEL evaluation", err, map[string]any{
				"process": c.processName,
			})
		}
	case *string:
		if v == nil {
			return false, catcher.Error("failed to evaluate CEL expression", errors.New("data pointer is nil"), map[string]any{
				"process": c.processName,
			})
		}
		jsonData = *v
		err := json.Unmarshal([]byte(jsonData), &valuesMap)
		if err != nil {
			return false, catcher.Error("failed to unmarshal JSON data", err, map[string]any{
				"process": c.processName,
			})
		}
	case string:
		jsonData = v
		err := json.Unmarshal([]byte(jsonData), &valuesMap)
		if err != nil {
			return false, catcher.Error("failed to unmarshal JSON data", err, map[string]any{
				"process": c.processName,
			})
		}
	case map[string]interface{}:
		valuesMap = v
		b, _ := json.Marshal(v)
		jsonData = string(b)
	default:
		// Attempt to handle other types by converting to JSON first
		b, err := json.Marshal(v)
		if err != nil {
			return false, catcher.Error("failed to marshal data to JSON", err, map[string]any{
				"process": c.processName,
			})
		}
		jsonData = string(b)
		err = json.Unmarshal(b, &valuesMap)
		if err != nil {
			return false, catcher.Error("failed to unmarshal intermediate JSON data", err, map[string]any{
				"process": c.processName,
			})
		}
	}

	if valuesMap == nil {
		valuesMap = make(map[string]interface{})
	}

	// Internal data for overloads
	valuesMap["_data_"] = jsonData

	cacheKey := c.getCacheKey(expression, valuesMap)
	return c.Get(cacheKey, expression, valuesMap, envOption...)
}

// Evaluate is a wrapper around Eval for backward compatibility.
// Note: It uses (data, expression) order while Eval uses (expression, data).
func (c *CELCache) Evaluate(data *string, expression string, envOption ...cel.EnvOption) (bool, error) {
	return c.Eval(expression, data, envOption...)
}

// execute is the internal method that performs the actual CEL program evaluation.
func (c *CELCache) execute(prg cel.Program, valuesMap map[string]interface{}) (bool, error) {
	out, details, err := prg.Eval(valuesMap)
	if err != nil {
		return false, catcher.Error("failed to evaluate program", err, map[string]any{
			"details": details,
			"process": c.processName,
		})
	}

	if out.Type() == cel.BoolType {
		return out.Value().(bool), nil
	}

	return false, nil
}

func (c *CELCache) valueToCelType(value interface{}) *cel.Type {
	switch value.(type) {
	case bool:
		return cel.BoolType
	case string:
		return cel.StringType
	case int, int32, int64:
		return cel.IntType
	case uint, uint32, uint64:
		return cel.UintType
	case float32, float64:
		return cel.DoubleType
	case []byte:
		return cel.BytesType
	case time.Time:
		return cel.TimestampType
	case map[string]interface{}:
		return cel.MapType(cel.StringType, cel.DynType)
	case map[string]*structpb.Value:
		return cel.MapType(cel.StringType, cel.DynType)
	case []interface{}:
		return cel.ListType(cel.DynType)
	case nil:
		return cel.NullType
	case proto.Message:
		return cel.DynType
	default:
		t := reflect.TypeOf(value)
		return cel.ObjectType(t.String())
	}
}
