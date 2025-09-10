//go:build botguard

package botguard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/dop251/goja"
)

// GojaSolver executes a user-provided JS file to produce Botguard tokens.
// The script must define a global function `bgAttest(input)` returning a string token
// or an object { token: string, ttlSeconds?: number }.
type GojaSolver struct {
	scriptPath string
}

func NewGojaSolver() *GojaSolver { return &GojaSolver{} }

func NewGojaSolverWithScript(scriptPath string) *GojaSolver {
	return &GojaSolver{scriptPath: scriptPath}
}

func (s *GojaSolver) Attest(ctx context.Context, input Input) (Output, error) {
	if s == nil || s.scriptPath == "" {
		return Output{}, errors.New("goja solver: script path not set")
	}
	script, err := os.ReadFile(s.scriptPath)
	if err != nil {
		return Output{}, fmt.Errorf("read script: %w", err)
	}
	vm := goja.New()

	// Provide a minimal console.log
	_ = vm.Set("console", map[string]any{
		"log": func(...any) {},
	})

	// Encode input as a plain JS object
	inJSON, _ := json.Marshal(input)
	var inObj map[string]any
	_ = json.Unmarshal(inJSON, &inObj)
	_ = vm.Set("__bgInput", inObj)

	if _, err := vm.RunScript(s.scriptPath, string(script)); err != nil {
		return Output{}, fmt.Errorf("run script: %w", err)
	}

	fn, ok := goja.AssertFunction(vm.Get("bgAttest"))
	if !ok {
		return Output{}, errors.New("bgAttest function not found in script")
	}
	// Call bgAttest(__bgInput)
	res, err := fn(goja.Undefined(), vm.Get("__bgInput"))
	if err != nil {
		return Output{}, fmt.Errorf("bgAttest error: %w", err)
	}

	var out Output
	if goja.IsUndefined(res) || goja.IsNull(res) {
		return Output{}, errors.New("bgAttest returned undefined/null")
	}
	if str, ok := res.Export().(string); ok {
		out.Token = str
		return out, nil
	}
	// Try object form
	if obj := res.ToObject(vm); obj != nil {
		if v := obj.Get("token"); !goja.IsUndefined(v) && !goja.IsNull(v) {
			out.Token = v.String()
		}
		if v := obj.Get("ttlSeconds"); !goja.IsUndefined(v) && !goja.IsNull(v) {
			if n, ok := v.Export().(int64); ok && n > 0 {
				out.ExpiresAt = time.Now().Add(time.Duration(n) * time.Second)
			}
		}
		return out, nil
	}
	return Output{}, errors.New("unexpected bgAttest return type")
}
