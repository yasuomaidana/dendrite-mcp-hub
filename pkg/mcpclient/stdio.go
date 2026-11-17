package mcpclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Stdio transport limits.
const (
	// DefaultCallTimeout bounds a Send whose context has no deadline of its own.
	DefaultCallTimeout = 30 * time.Second
	// closeGrace is how long Close waits for the child to leave after its stdin is closed.
	closeGrace = 2 * time.Second
	// maxLine is the largest JSON-RPC line the reader accepts.
	maxLine = 10 * 1024 * 1024
	// stderrTail is how much of the child's stderr is kept for error messages.
	stderrTail = 4096
)

// pipeStdin and pipeStdout open the subprocess pipes; tests replace them to make the pipes fail.
var (
	pipeStdin  = (*exec.Cmd).StdinPipe
	pipeStdout = (*exec.Cmd).StdoutPipe
)

// StdioTransport runs an MCP server as a subprocess and speaks line-delimited JSON-RPC over its stdio.
//
// One reader goroutine routes each response to the call that waits for its id, so calls may overlap and
// answers may arrive in any order. Lines that are not responses (server notifications, server requests,
// log output on stdout) are skipped. The child's stderr is kept (last stderrTail bytes) and quoted in errors.
type StdioTransport struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stderr  *tailBuffer
	timeout time.Duration

	wmu     sync.Mutex // serialises writes to stdin
	mu      sync.Mutex // guards pending
	pending map[string]chan *JSONRPCResponse

	exited    chan struct{} // closed once the child is reaped
	closeOnce sync.Once
}

// NewStdioTransport starts command and returns a transport bound to its stdio.
func NewStdioTransport(command string, args ...string) (*StdioTransport, error) {
	cmd := exec.Command(command, args...)
	stdin, err := pipeStdin(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to pipe stdin: %w", err)
	}
	stdout, err := pipeStdout(cmd)
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("failed to pipe stdout: %w", err)
	}
	tail := &tailBuffer{max: stderrTail}
	cmd.Stderr = tail
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, fmt.Errorf("failed to start stdio command: %w", err)
	}
	s := &StdioTransport{
		cmd: cmd, stdin: stdin, stderr: tail, timeout: DefaultCallTimeout,
		pending: map[string]chan *JSONRPCResponse{}, exited: make(chan struct{}),
	}
	go s.readLoop(stdout)
	return s, nil
}

// SetTimeout changes the bound applied to calls whose context has no deadline.
func (s *StdioTransport) SetTimeout(d time.Duration) { s.timeout = d }

// readLoop routes responses until the child closes stdout, then reaps it and fails every waiting call.
func (s *StdioTransport) readLoop(stdout io.Reader) {
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), maxLine)
	for sc.Scan() {
		if key, res, ok := parseResponse(sc.Bytes()); ok {
			s.deliver(key, res)
		}
	}
	_ = s.cmd.Wait()
	close(s.exited)
}

// deliver hands a response to the call waiting for key; an unmatched response is dropped.
func (s *StdioTransport) deliver(key string, res *JSONRPCResponse) {
	s.mu.Lock()
	ch := s.pending[key]
	delete(s.pending, key)
	s.mu.Unlock()
	if ch != nil {
		ch <- res
	}
}

// parseResponse decodes one stdout line. ok is false for anything that is not a response to a request:
// blank or non-JSON lines, notifications and requests sent by the server.
func parseResponse(line []byte) (key string, res *JSONRPCResponse, ok bool) {
	var in struct {
		JSONRPCResponse
		Method string `json:"method"`
	}
	if err := json.Unmarshal(line, &in); err != nil || in.Method != "" || in.ID == nil {
		return "", nil, false
	}
	return idKey(in.ID), &in.JSONRPCResponse, true
}

// idKey normalises a JSON-RPC id: ids sent as integers come back from json as float64.
func idKey(id interface{}) string {
	if f, ok := id.(float64); ok {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return fmt.Sprint(id)
}

// write sends one line to the child.
func (s *StdioTransport) write(req *JSONRPCRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if _, err := s.stdin.Write(append(data, '\n')); err != nil {
		return s.wrap("failed to write to stdio stdin", err)
	}
	return nil
}

// wrap adds the child's recent stderr to an error.
func (s *StdioTransport) wrap(msg string, err error) error {
	if tail := strings.TrimSpace(s.stderr.String()); tail != "" {
		return fmt.Errorf("%s: %w (server stderr: %s)", msg, err, tail)
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// Notify writes a notification and returns without waiting for anything.
func (s *StdioTransport) Notify(_ context.Context, req *JSONRPCRequest) error {
	return s.write(req)
}

// Send writes a request and waits for the response with the same id, the end of ctx, or the child's exit.
func (s *StdioTransport) Send(ctx context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}
	key := idKey(req.ID)
	ch := make(chan *JSONRPCResponse, 1)
	s.mu.Lock()
	s.pending[key] = ch
	s.mu.Unlock()
	defer s.forget(key)

	if err := s.write(req); err != nil {
		return nil, err
	}
	select {
	case res := <-ch:
		return res, nil
	case <-ctx.Done():
		return nil, s.wrap("waiting for the mcp server", ctx.Err())
	case <-s.exited:
		return nil, s.wrap("mcp server exited", io.ErrUnexpectedEOF)
	}
}

// forget drops the registration of a call that is over.
func (s *StdioTransport) forget(key string) {
	s.mu.Lock()
	delete(s.pending, key)
	s.mu.Unlock()
}

// Close ends the child: it closes stdin, gives the process a moment to leave, then kills it, and always
// waits until it is reaped. Closing twice is harmless.
func (s *StdioTransport) Close() error {
	s.closeOnce.Do(func() {
		_ = s.stdin.Close()
		select {
		case <-s.exited:
		case <-time.After(closeGrace):
			_ = s.cmd.Process.Kill()
			<-s.exited
		}
	})
	return nil
}

// tailBuffer is an io.Writer that keeps the last max bytes written to it.
type tailBuffer struct {
	mu  sync.Mutex
	buf []byte
	max int
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if over := len(t.buf) - t.max; over > 0 {
		t.buf = t.buf[over:]
	}
	return len(p), nil
}

func (t *tailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.buf)
}
