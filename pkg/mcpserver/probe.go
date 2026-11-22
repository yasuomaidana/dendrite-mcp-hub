package mcpserver

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

// ProbeSpec is one read-only tool call used to check that a tool answers.
type ProbeSpec struct {
	Tool    string
	Args    map[string]any
	Timeout time.Duration // zero means DefaultProbeTimeout
}

// ProbeResult is the outcome of one ProbeSpec.
type ProbeResult struct {
	Tool     string
	OK       bool
	Skipped  bool
	Duration time.Duration
	Detail   string
}

// DefaultProbeTimeout bounds every probe that does not set its own.
const DefaultProbeTimeout = 5 * time.Second

// probeDetailMax is how many characters of a failure the matrix shows.
const probeDetailMax = 160

// ProbeTools calls every spec against srv sequentially. A tool the server does not register is
// reported as skipped (it may be outside the active profile), not as a failure.
func ProbeTools(ctx context.Context, srv *Server, specs []ProbeSpec) []ProbeResult {
	results := make([]ProbeResult, 0, len(specs))
	for _, spec := range specs {
		h, ok := srv.Handler(spec.Tool)
		if !ok {
			results = append(results, ProbeResult{Tool: spec.Tool, Skipped: true, Detail: "not registered"})
			continue
		}
		timeout := spec.Timeout
		if timeout <= 0 {
			timeout = DefaultProbeTimeout
		}
		results = append(results, ProbeOne(ctx, spec, h, timeout))
	}
	return results
}

// ProbeOne runs one handler with a timeout, recovering a panic into a failed result.
func ProbeOne(ctx context.Context, spec ProbeSpec, h ToolHandler, timeout time.Duration) ProbeResult {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args := spec.Args
	if args == nil {
		args = map[string]any{}
	}
	start := time.Now()
	type outcome struct {
		res *CallToolResult
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- outcome{err: fmt.Errorf("panic: %v", r)}
			}
		}()
		res, err := h(cctx, args)
		done <- outcome{res, err}
	}()
	pr := ProbeResult{Tool: spec.Tool}
	select {
	case o := <-done:
		pr.Duration = time.Since(start)
		pr.OK, pr.Detail = judge(o.res, o.err)
	case <-cctx.Done():
		pr.Duration = time.Since(start)
		pr.Detail = fmt.Sprintf("timeout after %s", timeout)
	}
	return pr
}

// judge classifies one finished call: ok, or the first line of what went wrong.
func judge(res *CallToolResult, err error) (ok bool, detail string) {
	switch {
	case err != nil:
		return false, firstLine(err.Error(), probeDetailMax)
	case res == nil:
		return false, "nil result"
	case res.IsError:
		return false, firstLine(ContentText(res), probeDetailMax)
	}
	return true, ""
}

// firstLine is s trimmed, cut at its first newline and at max runes, with "…" when cut by length.
func firstLine(s string, max int) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if r := []rune(s); len(r) > max {
		s = string(r[:max]) + "…"
	}
	return s
}

// ContentText joins the text items of a result.
func ContentText(r *CallToolResult) string {
	var sb strings.Builder
	for _, c := range r.Content {
		sb.WriteString(c.Text)
	}
	return sb.String()
}

// WriteProbeMatrix prints the pass/fail matrix and returns the number of failures.
func WriteProbeMatrix(w io.Writer, results []ProbeResult) int {
	sorted := append([]ProbeResult(nil), results...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Tool < sorted[j].Tool })
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fails := 0
	for _, r := range sorted {
		status := "PASS"
		switch {
		case r.Skipped:
			status = "SKIP"
		case !r.OK:
			status = "FAIL"
			fails++
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%dms\t%s\n", status, r.Tool, r.Duration.Milliseconds(), r.Detail)
	}
	_ = tw.Flush()
	return fails
}
