package mcpclient_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/mcpclient"
	"github.com/yasuomaidana/dendrite-mcp-hub/pkg/mcpserver"
)

// helperEnv selects what the test binary does when it is started as a stand-in MCP server subprocess.
const helperEnv = "MCPCLIENT_HELPER"

// startHelper launches this test binary as a real stdio MCP server in the given mode.
func startHelper(t *testing.T, mode string) *mcpclient.StdioTransport {
	t.Helper()
	t.Setenv(helperEnv, mode)
	tr, err := mcpclient.NewStdioTransport(os.Args[0], "-test.run=^TestHelperServer$")
	if err != nil {
		t.Fatalf("start helper %s: %v", mode, err)
	}
	t.Cleanup(func() { _ = tr.Close() })
	return tr
}

// TestHelperServer is the subprocess body, not a test: it returns at once unless helperEnv is set.
func TestHelperServer(_ *testing.T) {
	mode := os.Getenv(helperEnv)
	if mode == "" {
		return
	}
	helpers[mode]()
	os.Exit(0)
}

var helpers = map[string]func(){
	"server":  helperServer,
	"noisy":   helperNoisy,
	"reorder": helperReorder,
	"crash":   helperCrash,
	"silent":  helperSilent,
	"stuck":   func() { time.Sleep(time.Hour) },
}

// requests yields the request lines on stdin.
func requests() *bufio.Scanner {
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	return sc
}

// helperServer is a real hub MCP server with one tool, preceded by noise on stdout and stderr.
func helperServer() {
	fmt.Fprintln(os.Stderr, "helper starting")
	fmt.Println("log: helper starting on stdout")
	fmt.Println(`{"jsonrpc":"2.0","method":"notifications/message","params":{"data":"hi"}}`)
	srv := mcpserver.NewServer(mcpserver.ServerConfig{Name: "helper", Version: "9.9"})
	srv.RegisterTool(mcpserver.Tool{Name: "echo", Description: "echoes text"},
		func(_ context.Context, args map[string]interface{}) (*mcpserver.CallToolResult, error) {
			text, _ := args["text"].(string)
			return &mcpserver.CallToolResult{Content: []mcpserver.ContentItem{{Type: "text", Text: "echo:" + text}}}, nil
		})
	_ = srv.ServeStdio(context.Background(), os.Stdin, os.Stdout)
}

// helperNoisy answers every request, but first writes a blank line, a log line, a notification, a server
// request that reuses the id, and a response nobody asked for.
func helperNoisy() {
	for sc := requests(); sc.Scan(); {
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.Unmarshal(sc.Bytes(), &req)
		if len(req.ID) == 0 {
			continue
		}
		fmt.Println()
		fmt.Println("log: working")
		fmt.Println(`{"jsonrpc":"2.0","method":"notifications/progress"}`)
		fmt.Printf("{\"jsonrpc\":\"2.0\",\"id\":%s,\"method\":\"roots/list\"}\n", req.ID)
		fmt.Println(`{"jsonrpc":"2.0","id":999,"result":{"stray":true}}`)
		fmt.Printf("{\"jsonrpc\":\"2.0\",\"id\":%s,\"result\":{\"ok\":true}}\n", req.ID)
	}
}

// helperReorder reads two requests and answers the second one first.
func helperReorder() {
	var ids []string
	for sc := requests(); len(ids) < 2 && sc.Scan(); {
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.Unmarshal(sc.Bytes(), &req)
		ids = append(ids, string(req.ID))
	}
	for i := len(ids) - 1; i >= 0; i-- {
		fmt.Printf("{\"jsonrpc\":\"2.0\",\"id\":%s,\"result\":{\"answered\":%s}}\n", ids[i], ids[i])
	}
}

// helperCrash writes a long stderr and dies on the first request.
func helperCrash() {
	requests().Scan()
	fmt.Fprint(os.Stderr, strings.Repeat("x", 6000)+"fatal: boom\n")
	os.Exit(3)
}

// helperSilent reads requests and never answers.
func helperSilent() {
	for sc := requests(); sc.Scan(); {
	}
}

func rpc(id interface{}, method string) *mcpclient.JSONRPCRequest {
	return &mcpclient.JSONRPCRequest{JSONRPC: "2.0", ID: id, Method: method}
}

func within(t *testing.T, d time.Duration, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { fn(); close(done) }()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("%s did not finish within %v", what, d)
	}
}
