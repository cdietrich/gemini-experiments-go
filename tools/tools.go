package tools

import (
	"context"
	"encoding/json"

	"google.golang.org/genai"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, args map[string]any) (string, error)
}

func ToFunctionDeclaration(t Tool) *genai.FunctionDeclaration {
	params := t.Parameters()
	if params == nil {
		params = map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}
	return &genai.FunctionDeclaration{
		Name:                 t.Name(),
		Description:          t.Description(),
		ParametersJsonSchema: params,
	}
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) FunctionDeclarations() []*genai.FunctionDeclaration {
	declarations := make([]*genai.FunctionDeclaration, 0, len(r.tools))
	for _, t := range r.tools {
		declarations = append(declarations, ToFunctionDeclaration(t))
	}
	return declarations
}

func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (string, error) {
	t, ok := r.Get(name)
	if !ok {
		return "", nil
	}
	return t.Execute(ctx, args)
}

func ParseFunctionCallArgs(fc *genai.FunctionCall) map[string]any {
	if fc == nil || fc.Args == nil {
		return map[string]any{}
	}

	args := make(map[string]any)
	for k, v := range fc.Args {
		switch val := v.(type) {
		case string:
			args[k] = val
		case float64:
			args[k] = val
		case int:
			args[k] = val
		case bool:
			args[k] = val
		case []any:
			args[k] = val
		case map[string]any:
			args[k] = val
		default:
			bytes, _ := json.Marshal(val)
			args[k] = string(bytes)
		}
	}
	return args
}
