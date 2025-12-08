FROM golang:1.25.5 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum

# Cache deps before building and copying source so that we don't need to
# re-download as much, and so that source changes don't invalidate our
# dependency layer.
RUN go mod download

# Install dependencies
#
# NOTE: Stick to version v1.28.0 of kubectl-oidc_login, as v1.28.1 appears to
# have some issues related to the cache.
RUN apt-get update && \
    apt-get install -y curl zip && \
    curl -L -O https://dl.k8s.io/release/v1.30.1/bin/linux/$TARGETARCH/kubectl && \
    curl -L -O https://github.com/int128/kubelogin/releases/download/v1.28.0/kubelogin_linux_$TARGETARCH.zip && \
    unzip kubelogin_linux_$TARGETARCH.zip kubelogin && mv kubelogin kubectl-oidc_login && \
    chmod +x kubectl kubectl-oidc_login && \
    rm -f kubelogin_linux_$TARGETARCH.zip && \
    rm -rf /var/cache/apt/archives

# Build
COPY cmd/ ./cmd
COPY internal/ ./internal
COPY pkg/ ./pkg
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o inventory ./cmd/inventory

#FROM gcr.io/distroless/static:nonroot
FROM alpine:3.22
RUN apk add --update \
    python3 curl which bash
RUN curl -sSL https://sdk.cloud.google.com > /tmp/gcl && \
    bash /tmp/gcl --install-dir=/app --disable-prompts && \
    /app/google-cloud-sdk/bin/gcloud components install gke-gcloud-auth-plugin

RUN addgroup -S nonroot && adduser -S nonroot -G nonroot
WORKDIR /app
ENV PATH=$PATH:/app/bin:/app/google-cloud-sdk/bin
COPY --from=builder /workspace/kubectl ./bin/
COPY --from=builder /workspace/kubectl-oidc_login ./bin/
COPY --from=builder /workspace/inventory .
USER nonroot:nonroot

ENTRYPOINT ["/app/inventory"]
