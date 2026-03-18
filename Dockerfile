# syntax=docker/dockerfile:1
# Stage 1 — build
FROM golang:1.25-alpine AS builder

ARG VERSION=dev

WORKDIR /src

# Download deps first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w -X github.com/scttfrdmn/bucktooth/cmd/bucktooth/cmd.Version=${VERSION}" \
    -o /bin/bucktooth \
    ./cmd/bucktooth

# Stage 2 — runtime (distroless)
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /bin/bucktooth /usr/local/bin/bucktooth

EXPOSE 8080 18789

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/usr/local/bin/bucktooth", "status"]

ENTRYPOINT ["/usr/local/bin/bucktooth", "start"]
