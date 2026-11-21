package mcpserver_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/mcpserver"
)

type (
	rs = map[string]interface{}
	hs = mcpserver.ToolHandler
)

func text(r *mcpserver.CallToolResult) string { return mcpserver.ContentText(r) }

func TestResultHelpers(t *testing.T) {
	if got := text(mcpserver.TextResult("hi")); got != "hi" {
		t.Fatalf("TextResult = %q", got)
	}
	res, err := mcpserver.JsonResult(rs{"a": 1})
	if err != nil || res.IsError || !strings.Contains(text(res), "\"a\": 1") {
		t.Fatalf("JsonResult = %+v, %v", res, err)
	}
}

func TestJsonResultReportsAnUnmarshallableValue(t *testing.T) {
	if _, err := mcpserver.JsonResult(make(chan int)); err == nil {
		t.Fatal("JsonResult must report a value that cannot be marshalled")
	}
}

func TestErrorAndResultOrError(t *testing.T) {
	failed := mcpserver.ErrorResult("bad %d", 7)
	if !failed.IsError || text(failed) != "bad 7" {
		t.Fatalf("ErrorResult = %+v", failed)
	}
	if ok := mcpserver.ResultOrError(rs{"k": "v"}, nil, "x: %v"); ok.IsError || !strings.Contains(text(ok), `"k": "v"`) {
		t.Fatalf("ResultOrError(ok) = %+v", ok)
	}
	if bad := mcpserver.ResultOrError(nil, errors.New("boom"), "failed: %v"); !bad.IsError || text(bad) != "failed: boom" {
		t.Fatalf("ResultOrError(err) = %+v", bad)
	}
}

func TestSchemaHelpersAndAddTool(t *testing.T) {
	prop := mcpserver.TypedProp("string", "a name")
	list := mcpserver.ListProp("string", "names")
	if prop["type"] != "string" || list["type"] != "array" || list["items"].(rs)["type"] != "string" {
		t.Fatalf("props = %v, %v", prop, list)
	}
}

func TestObjectSchemasAndAddTool(t *testing.T) {
	prop := mcpserver.TypedProp("string", "a name")
	loose := mcpserver.PropsSchema(rs{"name": prop})
	strict := mcpserver.ObjectSchema([]string{"name"}, rs{"name": prop})
	if loose["type"] != "object" || loose["required"] != nil || strict["required"].([]string)[0] != "name" {
		t.Fatalf("schemas = %v, %v", loose, strict)
	}
	srv := mcpserver.NewServer(mcpserver.ServerConfig{})
	mcpserver.AddTool(srv, "t", "does t", strict, func(context.Context, map[string]interface{}) (*mcpserver.CallToolResult, error) {
		return mcpserver.TextResult("done"), nil
	})
	tools := srv.ListTools()
	if len(tools) != 1 || tools[0].Name != "t" || tools[0].Description != "does t" || tools[0].InputSchema["type"] != "object" {
		t.Fatalf("tools = %+v", tools)
	}
}

func TestFilterToolsKeepsIdentityAndSharesHandlers(t *testing.T) {
	src := mcpserver.NewServer(mcpserver.ServerConfig{Name: "n", Version: "v"})
	for _, name := range []string{"keep_a", "keep_b", "drop_c"} {
		mcpserver.AddTool(src, name, "", nil, func(context.Context, map[string]interface{}) (*mcpserver.CallToolResult, error) {
			return mcpserver.TextResult(name), nil
		})
	}
	got := mcpserver.FilterTools(src, func(name string) bool { return strings.HasPrefix(name, "keep_") })
	if info := got.Info(); info.Name != "n" || info.Version != "v" {
		t.Fatalf("identity = %+v", info)
	}
	if n := len(got.ListTools()); n != 2 {
		t.Fatalf("kept %d tools, want 2", n)
	}
	if _, ok := got.Handler("drop_c"); ok {
		t.Fatal("a filtered tool must be gone")
	}
	h, _ := got.Handler("keep_a")
	if res, _ := h(context.Background(), nil); text(res) != "keep_a" {
		t.Fatalf("shared handler answered %q", text(res))
	}
}

func TestProbeToolsClassifiesOutcomes(t *testing.T) {
	s := mcpserver.NewServer(mcpserver.ServerConfig{})
	reg := func(name string, h hs) { s.RegisterTool(mcpserver.Tool{Name: name}, h) }
	reg("ok", func(context.Context, map[string]any) (*mcpserver.CallToolResult, error) {
		return mcpserver.TextResult("fine"), nil
	})
	reg("err", func(context.Context, map[string]any) (*mcpserver.CallToolResult, error) {
		return nil, errors.New("plane unreachable\nstack")
	})
	reg("tool-error", func(context.Context, map[string]any) (*mcpserver.CallToolResult, error) {
		return mcpserver.ErrorResult("bad input"), nil
	})
	reg("nil", func(context.Context, map[string]any) (*mcpserver.CallToolResult, error) { return nil, nil })
	reg("slow", func(ctx context.Context, _ map[string]any) (*mcpserver.CallToolResult, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	reg("panics", func(context.Context, map[string]any) (*mcpserver.CallToolResult, error) { panic("boom") })

	got := mcpserver.ProbeTools(context.Background(), s, []mcpserver.ProbeSpec{
		{Tool: "ok", Args: rs{"x": 1}}, {Tool: "err"}, {Tool: "tool-error"}, {Tool: "nil"},
		{Tool: "slow", Timeout: 20 * time.Millisecond}, {Tool: "panics"}, {Tool: "missing"},
	})
	want := map[string][2]bool{ // ok, skipped
		"ok": {true, false}, "err": {false, false}, "tool-error": {false, false}, "nil": {false, false},
		"slow": {false, false}, "panics": {false, false}, "missing": {false, true},
	}
	for _, r := range got {
		if w := want[r.Tool]; r.OK != w[0] || r.Skipped != w[1] {
			t.Errorf("%s = ok:%v skipped:%v (%s)", r.Tool, r.OK, r.Skipped, r.Detail)
		}
	}
	var buf bytes.Buffer
	if fails := mcpserver.WriteProbeMatrix(&buf, got); fails != 5 {
		t.Errorf("fails = %d\n%s", fails, buf.String())
	}
	for _, s := range []string{"PASS", "FAIL", "SKIP", "plane unreachable", "timeout after", "nil result", "bad input", "panic: boom"} {
		if !strings.Contains(buf.String(), s) {
			t.Errorf("matrix lacks %q:\n%s", s, buf.String())
		}
	}
}

func TestProbeDetailIsOneBoundedLine(t *testing.T) {
	s := mcpserver.NewServer(mcpserver.ServerConfig{})
	long := strings.Repeat("é", 300) + "\nsecond line"
	s.RegisterTool(mcpserver.Tool{Name: "long"}, func(context.Context, map[string]any) (*mcpserver.CallToolResult, error) {
		return nil, errors.New(long)
	})
	r := mcpserver.ProbeTools(context.Background(), s, []mcpserver.ProbeSpec{{Tool: "long"}})[0]
	if n := len([]rune(r.Detail)); n != 161 || !strings.HasSuffix(r.Detail, "…") || strings.Contains(r.Detail, "second") {
		t.Fatalf("detail = %d runes: %.30s…", n, r.Detail)
	}
}
