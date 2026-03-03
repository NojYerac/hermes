# Build-time version pins — override with --build-arg as needed.
ARG GO_VERSION=1.24
ARG GINKGO_VERSION=latest
ARG GOLANGCI_LINT_VERSION=latest

# ─── Stage 1: build hermes-bridge ────────────────────────────────────────────
FROM golang:${GO_VERSION}-alpine AS builder

RUN apk add --no-cache build-base git

WORKDIR /src

# Cache module downloads before copying the rest of the source.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /hermes-bridge ./cmd/hermes-bridge

# ─── Stage 2: runtime image ──────────────────────────────────────────────────
# Re-declare ARGs so they are in scope for this stage.
ARG GO_VERSION
ARG GINKGO_VERSION
ARG GOLANGCI_LINT_VERSION

FROM golang:${GO_VERSION}-alpine

# Propagate build args into the stage environment for use in RUN commands.
ARG GINKGO_VERSION
ARG GOLANGCI_LINT_VERSION

# System deps required for module fetching and CGO-optional builds.
RUN apk add --no-cache build-base git openssh ca-certificates curl

# Ginkgo BDD test runner.
RUN go install github.com/onsi/ginkgo/v2/ginkgo@${GINKGO_VERSION}

# GolangCI-Lint via the official binary distribution script.
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
    | sh -s -- -b /go/bin ${GOLANGCI_LINT_VERSION}

# Copy the compiled bridge binary from the builder stage.
COPY --from=builder /hermes-bridge /go/bin/hermes-bridge

# Create a non-root user and pre-create the Go cache directories so they are
# owned by that user before the volume mounts are applied at runtime.
RUN addgroup -g 1000 -S hermes && adduser -u 1000 -S -G hermes hermes \
    && mkdir -p /go/pkg/mod /.cache/go-build \
    && chown -R hermes:hermes /go/pkg/mod /.cache/go-build

USER hermes

# Bind address can be overridden at runtime; defaults to 127.0.0.1:3010.
ENV HERMES_ADDR=127.0.0.1:3010
# Comma-separated list of executable names the bridge may run.
# Defaults to go,ginkgo,golangci-lint when unset.
ENV HERMES_ALLOWED_COMMANDS=

EXPOSE 3010

ENTRYPOINT ["/go/bin/hermes-bridge"]
