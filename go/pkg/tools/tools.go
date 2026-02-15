package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"runtime"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

// Registry 工具注册表：name -> (fn, desc)。fn 必须为 func(ctx context.Context, args *T) (string, error)，执行时自动注入 ctx。
type Registry struct {
	tools map[string]tool
}

type tool struct {
	fn       reflect.Value
	desc     string
	argsType reflect.Type // *Struct
}

// New 按 fn, desc, fn, desc, ... 解析 args 并建注册表。fn 必须为 func(ctx context.Context, args *T) (string, error)。
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
		name := runtime.FuncForPC(v.Pointer()).Name()
		if t.NumIn() != 2 || t.NumOut() != 2 {
			panic("tools: fn must be func(ctx context.Context, args *T) (string, error), not method")
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
		m[f.Name] = map[string]any{
			"type":        jsonType(f.Type),
			"description": f.Tag.Get("description"),
		}
	}
	return m
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
		return "string"
	}
}

// Execute
func (r *Registry) Execute(ctx context.Context, name string, argumentsJSON string) (string, error) {
	t, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("tools: unknown tool %q", name)
	}
	slog.Info("工具执行", "tool", name, "args", argumentsJSON)
	ptr := reflect.New(t.argsType.Elem())
	if err := json.Unmarshal([]byte(argumentsJSON), ptr.Interface()); err != nil {
		return "", fmt.Errorf("tools: unmarshal args: %w", err)
	}
	out := t.fn.Call([]reflect.Value{reflect.ValueOf(ctx), ptr})
	if !out[1].IsNil() {
		err := out[1].Interface().(error)
		slog.Info("工具执行完毕", "tool", name, "err", err.Error())
		return "", err
	}
	result := out[0].String()
	slog.Info("工具执行完毕", "tool", name, "result_len", len(result))
	return result, nil
}
