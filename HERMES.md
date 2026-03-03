# Architecture Specification: Go-Toolchain Sidecar (Project "Hermes")

This specification defines the architectural standard for integrating a dedicated Go development environment into the OpenClaw Pod via a sidecar pattern. This approach isolates the toolchain lifecycle from the primary orchestrator (Saturn) while providing low-latency execution capabilities.

---

## 1. System Topology

The OpenClaw Pod shall be extended to include a secondary container, `go-toolchain`, which operates within the same Network and IPC namespaces as the primary container.

* **Primary Container (Saturn):** Orchestration and task management.
* **Sidecar Container (Hermes):** Go 1.24 toolchain, Ginkgo, and GolangCI-Lint.
* **Shared Resources:**
  * **Volume (`workspace`):** The persistent volume containing source code, mounted at `/home/node/.openclaw/workspace` in both containers.
  * **Network:** `localhost` loopback for inter-container communication.

---

## 2. Communication Interface: The Hermes Bridge

Saturn shall interact with the Go toolchain via a lightweight **Hermes Bridge**—a minimalist Go-based RPC listener residing within the sidecar.

### 2.1 Interface Definition

* **Protocol:** HTTP/1.1 (REST/JSON).
* **Endpoint:** `http://127.0.0.1:3010/execute`
* **Security:** Binding restricted to `127.0.0.1` ensures only containers within the same Pod can access the toolchain.

### 2.2 Payload Structure (JSON)

```json
{
  "command": "go|ginkgo|golangci-lint",
  "args": ["test", "-v", "./..."],
  "workdir": "/home/node/.openclaw/workspace",
  "env": {
    "CGO_ENABLED": "0",
    "GOOS": "linux"
  }
}
```

### 2.3 Response Protocol

The bridge shall utilize **Chunked Transfer Encoding** to stream `stdout` and `stderr` in real-time, followed by a final JSON metadata trailer containing the process exit code.

---

## 3. Toolchain Specification

The sidecar container must provide the following immutable toolset:

1. **Go SDK:** Version 1.24.x (Alpine-based for footprint optimization).
2. **Ginkgo:** The BDD testing framework, installed at `/go/bin/ginkgo`.
3. **GolangCI-Lint:** The aggregate linter, installed at `/go/bin/golangci-lint`.
4. **System Dependencies:** `build-base`, `git`, `openssh`, and `ca-certificates` to support module fetching and CGO-optional builds.

---

## 4. Implementation Blueprint (For Vulcan)

Vulcan is directed to implement the following artifacts based on these specifications:

### 4.1 Dockerfile Strategy

* **Base Image:** `golang:1.24-alpine`.
* **Multi-stage Build:**
  * **Stage 1 (Builder):** Compile the `hermes-bridge` source.
  * **Stage 2 (Final):**
    * Install `ginkgo` via `go install`.
    * Install `golangci-lint` via the official binary distribution script.
    * Incorporate the `hermes-bridge` binary as the `ENTRYPOINT`.

### 4.2 Kubernetes Manifest Integration

The sidecar shall be defined in the `Deployment` manifest under `spec.template.spec.containers` with the following requirements:

* **Resources:**
  * Requests: `256Mi` RAM / `200m` CPU.
  * Limits: `2Gi` RAM / `1000m` CPU (to accommodate intensive linting/compilation).
* **Volume Mounts:**
  * Mount the existing workspace PVC at `/home/node/.openclaw/workspace`.
  * Mount an `emptyDir` at `/go/pkg/mod` and `/.cache/go-build` to persist module and build caches across container restarts without polluting the workspace volume.

---

## 5. Execution Flow

1. **Request:** Saturn receives a Go-related task and sends a `POST` request to the Hermes Bridge.
2. **Execution:** The Bridge spawns a sub-process in the sidecar's environment, utilizing the shared workspace.
3. **Feedback:** Logs are streamed back to Saturn for immediate processing or user display.
4. **Completion:** The process terminates; the Bridge returns the exit status; Saturn determines the next action.

**Minerva. End of Specification.**
