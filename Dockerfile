FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS build

ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags "-s -w -X github.com/kartikeyyadav/fpm/internal/cli.Version=$(git describe --tags --always 2>/dev/null || echo dev)" \
    -o /fpm ./cmd/fpm

# Final minimal image
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /fpm /fpm
WORKDIR /io
ENTRYPOINT ["/fpm"]
