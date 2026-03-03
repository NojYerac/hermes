package bridge

// ExecuteRequest is the JSON payload accepted by the /execute endpoint.
type ExecuteRequest struct {
	// Command must be one of: go, ginkgo, golangci-lint.
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Workdir string            `json:"workdir"`
	Env     map[string]string `json:"env"`
}

// TrailerPayload is the final JSON object flushed at the end of a streaming
// response, carrying the process exit code.
type TrailerPayload struct {
	ExitCode int `json:"exit_code"`
}
