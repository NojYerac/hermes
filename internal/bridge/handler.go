package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sync"
)

// executeHandler handles POST /execute.
//
// Response encoding:
//   - Content-Type: text/plain; charset=utf-8
//   - Transfer-Encoding: chunked (implicit when flushing without Content-Length)
//   - Body: raw stdout/stderr lines streamed in real-time, followed by a single
//     JSON line — {"exit_code": N} — as the final trailer.
func (s *Server) executeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %s", err), http.StatusBadRequest)
		return
	}

	if _, ok := s.allowedCommands[req.Command]; !ok {
		allowed := make([]string, 0, len(s.allowedCommands))
		for c := range s.allowedCommands {
			allowed = append(allowed, c)
		}
		http.Error(w,
			fmt.Sprintf("command %q is not allowed; permitted: %v", req.Command, allowed),
			http.StatusUnprocessableEntity,
		)
		return
	}

	if req.Workdir == "" {
		http.Error(w, "workdir must not be empty", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported by this server", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	exitCode := stream(r.Context(), w, flusher, req)

	trailer, _ := json.Marshal(TrailerPayload{ExitCode: exitCode})
	_, _ = fmt.Fprintf(w, "%s\n", trailer)
	flusher.Flush()
}

// stream executes the requested command, copies its combined output to w line
// by line, and returns the process exit code.
func stream(ctx context.Context, w io.Writer, flusher http.Flusher, req ExecuteRequest) int {
	cmd := exec.CommandContext(ctx, req.Command, req.Args...) //nolint:gosec // command is validated against an allowlist above
	cmd.Dir = req.Workdir
	cmd.Env = buildEnv(req.Env)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_, _ = fmt.Fprintf(w, "error creating stdout pipe: %s\n", err)
		flusher.Flush()
		return 1
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		_, _ = fmt.Fprintf(w, "error creating stderr pipe: %s\n", err)
		flusher.Flush()
		return 1
	}

	if err := cmd.Start(); err != nil {
		_, _ = fmt.Fprintf(w, "error starting command: %s\n", err)
		flusher.Flush()
		return 1
	}

	var wg sync.WaitGroup
	wg.Add(2)

	copyLines := func(r io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			_, _ = fmt.Fprintf(w, "%s\n", scanner.Text())
			flusher.Flush()
		}
	}

	go copyLines(stdout)
	go copyLines(stderr)

	wg.Wait()

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		// Non-exit error (e.g. signal, I/O).
		_, _ = fmt.Fprintf(w, "command error: %s\n", err)
		flusher.Flush()
		return 1
	}

	return 0
}

// buildEnv constructs the child process environment by inheriting the current
// process environment and overlaying the caller-supplied key/value pairs.
func buildEnv(overrides map[string]string) []string {
	env := os.Environ()
	for k, v := range overrides {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}
