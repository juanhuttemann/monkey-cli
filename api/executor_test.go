package api

import (
	"context"
	"errors"
	"testing"
)

type fixedExecutor struct {
	result string
	err    error
}

func (f fixedExecutor) ExecuteTool(_ context.Context, _ string, _ map[string]any) (string, error) {
	return f.result, f.err
}

func TestMultiExecutor_DispatchesToRegisteredExecutor(t *testing.T) {
	me := NewMultiExecutor()
	me.Register("alpha", fixedExecutor{result: "from alpha"})
	me.Register("beta", fixedExecutor{result: "from beta"})

	result, err := me.ExecuteTool(context.Background(), "alpha", nil)
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if result != "from alpha" {
		t.Errorf("ExecuteTool() = %q, want %q", result, "from alpha")
	}

	result, err = me.ExecuteTool(context.Background(), "beta", nil)
	if err != nil {
		t.Fatalf("ExecuteTool() returned error: %v", err)
	}
	if result != "from beta" {
		t.Errorf("ExecuteTool() = %q, want %q", result, "from beta")
	}
}

func TestMultiExecutor_UnknownToolReturnsError(t *testing.T) {
	me := NewMultiExecutor()
	_, err := me.ExecuteTool(context.Background(), "unknown", nil)
	if err == nil {
		t.Error("ExecuteTool() should return error for unregistered tool name")
	}
}

func TestMultiExecutor_PropagatesExecutorError(t *testing.T) {
	me := NewMultiExecutor()
	sentinel := errors.New("executor failed")
	me.Register("bad", fixedExecutor{err: sentinel})

	_, err := me.ExecuteTool(context.Background(), "bad", nil)
	if !errors.Is(err, sentinel) {
		t.Errorf("ExecuteTool() error = %v, want %v", err, sentinel)
	}
}

func TestMultiExecutor_PassesInputToExecutor(t *testing.T) {
	var gotInput map[string]any
	capture := captureExecutor{fn: func(_ string, input map[string]any) (string, error) {
		gotInput = input
		return "ok", nil
	}}

	me := NewMultiExecutor()
	me.Register("capture", capture)

	input := map[string]any{"key": "value"}
	_, _ = me.ExecuteTool(context.Background(), "capture", input)

	if gotInput["key"] != "value" {
		t.Errorf("executor received input %v, want key=value", gotInput)
	}
}

type captureExecutor struct {
	fn func(string, map[string]any) (string, error)
}

func (c captureExecutor) ExecuteTool(_ context.Context, name string, input map[string]any) (string, error) {
	return c.fn(name, input)
}
