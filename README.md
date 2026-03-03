# Hermes

A lightweight Go sidecar that exposes the Go toolchain (`go`, `ginkgo`, `golangci-lint`) over HTTP, enabling orchestrators running in the same Kubernetes Pod to trigger builds, tests, and linting without bundling a Go SDK themselves.

---

## How it works

Hermes binds a minimal HTTP server to `127.0.0.1:3010` (loopback only). Clients `POST /execute` with a JSON payload; the bridge spawns the requested process and streams its combined output back via chunked transfer encoding, finishing with a JSON trailer containing the exit code.

```text
POST /execute
Content-Type: application/json

{
  "command": "go",
  "args":    ["test", "-v", "./..."],
  "workdir": "/home/node/.openclaw/workspace",
  "env":     { "CGO_ENABLED": "0" }
}
```

**Response body** (streamed, `text/plain`):

```text
<stdout / stderr lines, flushed in real-time>
{"exit_code": 0}
```

`GET /healthz` returns `200 ok` and is suitable for liveness probes.

---

## Configuration

| Environment variable       | Default                        | Description                                              |
|----------------------------|--------------------------------|----------------------------------------------------------|
| `HERMES_ADDR`              | `127.0.0.1:3010`               | TCP address to listen on.                                |
| `HERMES_ALLOWED_COMMANDS`  | `go,ginkgo,golangci-lint`      | Comma-separated allowlist of executable names.           |

---

## Development

```bash
make build   # compile → build/hermes-bridge
make run     # go run ./cmd/hermes-bridge
make test    # go test ./...
make lint    # golangci-lint run ./...
```

## Docker

```bash
docker build -t hermes .
docker run --rm -p 3010:3010 hermes
```
