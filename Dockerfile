FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build

ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
COPY vendor/ vendor/
RUN go mod download 2>/dev/null || true

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags "-s -w -X github.com/kartikeyyadav/fpm/internal/cli.Version=${VERSION}" \
    -o /fpm ./cmd/fpm

# Final minimal image
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /fpm /fpm
WORKDIR /io
ENTRYPOINT ["/fpm"]
