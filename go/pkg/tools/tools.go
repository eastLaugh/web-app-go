package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

// NameTransformer 把 runtime.FuncForPC().Name() 的原始结果（如
// "github.com/eastLaugh/web-app-go/go/ports.vector_search"）变换为对外暴露的工具名。
// 默认绑 StripPackagePath（DeepSeek 兼容）。如果想回到完整路径名（OpenAI 不强校验，
// 早期实现），改 NameStrategy = IdentityName 即可。
type NameTransformer func(rawName string) string

// IdentityName 原样返回，等价于变轨前的行为。
func IdentityName(s string) string { return s }

// StripPackagePath 截掉最后一个 "." 之前的所有内容，并校验残余字符是否落在
// ^[a-zA-Z0-9_-]+$ 之内（DeepSeek 服务端硬约束）。
// 命中不合规则 panic，强制开发者把函数名改成 ASCII 友好形态；
// 如未来需要支持中文 / 方法接收器等非 ASCII 名字，再换更重的转义方案。
func StripPackagePath(raw string) string {
	name := raw
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	if !toolNameRe.MatchString(name) {
		panic(fmt.Sprintf("tools: function name %q (from %q) does not match ^[a-zA-Z0-9_-]+$; rename to ASCII identifier", name, raw))
	}
	return name
}

var toolNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// NameStrategy 当前生效的命名策略。改这一行即可全局切换。
var NameStrategy NameTransformer = StripPackagePath

// Registry 工具注册表：name -> (fn, desc)。fn 必须为 func(ctx context.Context, args *T) string，结果直接作为 ToolMessage 内容。
type Registry struct {
	tools map[string]tool
}

type tool struct {
	fn       reflect.Value
	desc     string
	argsType reflect.Type // *Struct
}

// New 按 fn, desc, fn, desc, ... 解析 args 并建注册表。fn 必须为 func(ctx context.Context, args *T) string。
func New(args ...any) *Registry {
	r := &Registry{tools: make(map[string]tool)}
	for i := 0; i < len(args); i += 2 {
		if i+1 >= len(args) {
			panic("tools: args must be fn, desc pairs")
		}
		fn := args[i]
		desc, ok := args[i+1].(string)
		if !ok {
			panic("tools: desc must be string")
		}
		v := reflect.ValueOf(fn)
		t := v.Type()
		if t.Kind() != reflect.Func {
			panic("tools: fn must be func")
		}
		name := NameStrategy(runtime.FuncForPC(v.Pointer()).Name())
		if t.NumIn() != 2 || t.NumOut() != 1 {
			panic("tools: fn must be func(ctx context.Context, args *T) string, not method")
		}
		if t.Out(0).Kind() != reflect.String {
			panic("tools: fn must return string")
		}
		ctxType := reflect.TypeFor[context.Context]()
		if !t.In(0).Implements(ctxType) {
			panic("tools: first param must be context.Context")
		}
		argsType := t.In(1)
		if argsType.Kind() != reflect.Pointer || argsType.Elem().Kind() != reflect.Struct {
			panic("tools: second param must be *Struct")
		}
		r.tools[name] = tool{fn: v, desc: desc, argsType: argsType}
	}
	return r
}

// ToParams 生成 OpenAI Chat Completions 的 Tools 参数
func (r *Registry) ToParams() []openai.ChatCompletionToolUnionParam {
	out := make([]openai.ChatCompletionToolUnionParam, 0, len(r.tools))
	for name, t := range r.tools {
		params := openai.FunctionParameters{
			"type":       "object",
			"properties": schemaFromStruct(t.argsType.Elem()),
		}
		out = append(out, openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name:        name,
			Description: openai.String(t.desc),
			Parameters:  params,
		}))
	}
	return out
}

func schemaFromStruct(t reflect.Type) map[string]any {
	m := make(map[string]any)
	for f := range t.Fields() {
		if !f.IsExported() {
			panic("tools: field must be exported")
		}
		if f.Tag.Get("json") != "" {
			panic("tools: don't use json tag")
		}
		m[f.Name] = schemaForType(f.Type, f.Tag.Get("description"))
	}
	return m
}

func schemaForType(t reflect.Type, desc string) map[string]any {
	if t.Kind() == reflect.Slice {
		elem := t.Elem()
		if elem.Kind() == reflect.Struct {
			return map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "object", "properties": schemaFromStruct(elem)},
				"description": desc,
			}
		}
		return map[string]any{
			"type":        "array",
			"items":       map[string]any{"type": jsonType(elem)},
			"description": desc,
		}
	}
	return map[string]any{
		"type":        jsonType(t),
		"description": desc,
	}
}

func jsonType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	default:
		panic("tools: unsupported type: " + t.String())
	}
}

// Execute
func (r *Registry) Execute(ctx context.Context, name string, argumentsJSON string) (string, error) {
	t, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("tools: unknown tool %q", name)
	}
	ptr := reflect.New(t.argsType.Elem())
	if err := json.Unmarshal([]byte(argumentsJSON), ptr.Interface()); err != nil {
		return "", fmt.Errorf("tools: unmarshal args: %w", err)
	}
	// trace: 待打桩
	result := onion(base)(t.fn, ctx, ptr)
	return result, nil
}

type handler func(reflect.Value, context.Context, reflect.Value) string

var Logger = slog.Default()

func onion(next handler) handler {
	return func(fn reflect.Value, ctx context.Context, ptr reflect.Value) (result string) {
		Logger.Info("工具执行中", "tool", fn.String(), "args", ptr.Interface())
		defer func(t time.Time) {
			Logger.Info("工具执行完毕", "result", result, "duration", time.Since(t))
		}(time.Now())
		result = next(fn, ctx, ptr)
		return
	}
}

func base(fn reflect.Value, ctx context.Context, ptr reflect.Value) string {
	return fn.Call([]reflect.Value{reflect.ValueOf(ctx), ptr})[0].String()
}
