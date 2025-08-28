# syntax=docker/dockerfile:1.7

############################
# Build stage (multi-arch) #
############################
FROM --platform=$BUILDPLATFORM golang:1.23-bookworm AS build
WORKDIR /src

# Reproducible, static build
ENV CGO_ENABLED=0 \
    GO111MODULE=on

# 1) Go module download (cached)
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# 2) Copy source
COPY . .

# 3) Build for target platform with build cache
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -buildvcs=false \
      -ldflags="-s -w" \
      -o /out/otel-backend ./cmd/otel-backend

#########################
# Runtime (distroless)  #
#########################
# Use distroless base (includes CA certs; still tiny; non-root)
FROM gcr.io/distroless/base-debian12:nonroot AS runtime
WORKDIR /app

# Copy static binary
COPY --from=build /out/otel-backend /app/otel-backend

# Drop capabilities / run as nonroot user
USER nonroot:nonroot
EXPOSE 8080

# No shell in distroless; entry directly to the binary
ENTRYPOINT ["/app/otel-backend"]