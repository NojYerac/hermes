# ─── Stage 1: build hermes-bridge ────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache build-base git

WORKDIR /src

# Cache module downloads before copying the rest of the source.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /hermes-bridge ./cmd/hermes-bridge

# ─── Stage 2: runtime image ──────────────────────────────────────────────────
FROM golang:1.24-alpine

# System deps required for module fetching and CGO-optional builds.
RUN apk add --no-cache build-base git openssh ca-certificates curl

# Ginkgo BDD test runner.
RUN go install github.com/onsi/ginkgo/v2/ginkgo@latest

# GolangCI-Lint via the official binary distribution script.
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
    | sh -s -- -b /go/bin latest

# Copy the compiled bridge binary from the builder stage.
COPY --from=builder /hermes-bridge /go/bin/hermes-bridge

# Bind address can be overridden at runtime; defaults to 127.0.0.1:3010.
ENV HERMES_ADDR=127.0.0.1:3010
# Comma-separated list of executable names the bridge may run.
# Defaults to go,ginkgo,golangci-lint when unset.
ENV HERMES_ALLOWED_COMMANDS=

EXPOSE 3010

ENTRYPOINT ["/go/bin/hermes-bridge"]
