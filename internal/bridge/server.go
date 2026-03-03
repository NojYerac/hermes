package bridge

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

const (
	// DefaultAddr is the loopback address the bridge binds to.  Binding to
	// 127.0.0.1 rather than 0.0.0.0 ensures only containers sharing the same
	// Pod network namespace can reach the bridge.
	DefaultAddr = "127.0.0.1:3010"

	readHeaderTimeout = 10 * time.Second
	shutdownTimeout   = 30 * time.Second
)

// DefaultAllowedCommands is the set of executable names permitted when none
// are explicitly specified in Config.
var DefaultAllowedCommands = []string{"go", "ginkgo", "golangci-lint"}

// Config holds runtime configuration for the Server.
type Config struct {
	// Addr is the TCP address to listen on. Defaults to DefaultAddr.
	Addr string
	// AllowedCommands is the list of executable names the bridge may run.
	// Defaults to DefaultAllowedCommands when empty.
	AllowedCommands []string
}

// Server is the Hermes Bridge HTTP server.
type Server struct {
	http            *http.Server
	logger          *slog.Logger
	allowedCommands map[string]struct{}
}

// New constructs a Server from cfg.
func New(cfg Config, logger *slog.Logger) *Server {
	if cfg.Addr == "" {
		cfg.Addr = DefaultAddr
	}
	if len(cfg.AllowedCommands) == 0 {
		cfg.AllowedCommands = DefaultAllowedCommands
	}

	allowed := make(map[string]struct{}, len(cfg.AllowedCommands))
	for _, c := range cfg.AllowedCommands {
		allowed[c] = struct{}{}
	}

	srv := &Server{
		logger:          logger,
		allowedCommands: allowed,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/execute", srv.executeHandler)
	mux.HandleFunc("/healthz", healthzHandler)

	srv.http = &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	return srv
}

// ListenAndServe starts the HTTP listener and blocks until the provided context
// is cancelled, after which it attempts a graceful shutdown.
func (s *Server) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.http.Addr)
	if err != nil {
		return err
	}

	allowedList := make([]string, 0, len(s.allowedCommands))
	for c := range s.allowedCommands {
		allowedList = append(allowedList, c)
	}
	s.logger.Info("hermes bridge listening",
		slog.String("addr", ln.Addr().String()),
		slog.Any("allowed_commands", allowedList),
	)

	errCh := make(chan error, 1)
	go func() {
		if err := s.http.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.logger.Info("shutting down hermes bridge")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		return s.http.Shutdown(shutdownCtx)
	}
}

// healthzHandler is a lightweight liveness probe endpoint.
func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}
