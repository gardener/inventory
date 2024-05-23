FROM golang:1.22.3 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum

# Cache deps before building and copying source so that we don't need to
# re-download as much, and so that source changes don't invalidate our
# dependency layer.
RUN go mod download

# Build
COPY cmd/ ./cmd
COPY internal/ ./internal
COPY pkg/ ./pkg
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o inventory ./cmd/inventory

FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=builder /workspace/inventory .
USER 65532:65532

ENTRYPOINT ["/app/inventory"]
