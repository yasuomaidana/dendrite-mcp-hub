package mcpclient

import (
	"errors"
	"io"
	"os/exec"
	"testing"
)

// swap replaces a package-level seam for the length of the test.
func swap[T any](t *testing.T, target *T, value T) {
	t.Helper()
	old := *target
	t.Cleanup(func() { *target = old })
	*target = value
}

func TestNewStdioTransportReportsPipeFailures(t *testing.T) {
	boom := errors.New("pipe failed")
	swap(t, &pipeStdin, func(*exec.Cmd) (io.WriteCloser, error) { return nil, boom })
	if _, err := NewStdioTransport("true"); !errors.Is(err, boom) {
		t.Fatalf("stdin failure = %v", err)
	}
	swap(t, &pipeStdin, (*exec.Cmd).StdinPipe)
	swap(t, &pipeStdout, func(*exec.Cmd) (io.ReadCloser, error) { return nil, boom })
	if _, err := NewStdioTransport("true"); !errors.Is(err, boom) {
		t.Fatalf("stdout failure = %v", err)
	}
}

func TestIDKeyNormalisesJSONNumbers(t *testing.T) {
	for id, want := range map[interface{}]string{float64(7): "7", 7: "7", "7": "7", float64(1e21): "1000000000000000000000"} {
		if got := idKey(id); got != want {
			t.Errorf("idKey(%v) = %q, want %q", id, got, want)
		}
	}
}

func TestTailBufferKeepsTheEnd(t *testing.T) {
	tail := &tailBuffer{max: 5}
	_, _ = tail.Write([]byte("abc"))
	_, _ = tail.Write([]byte("defgh"))
	if got := tail.String(); got != "defgh" {
		t.Fatalf("tail = %q", got)
	}
}
