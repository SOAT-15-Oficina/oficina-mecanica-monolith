#BUILD GO APP
FROM golang:1.26.1-alpine AS build-stage
WORKDIR /app

ARG VERSION=dev

COPY go.mod go.sum ./
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false \
    -ldflags="-s -w -X github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/observability.version=${VERSION}" \
    -o /techchallenge ./cmd/api/main.go

# SETUP CONTAINER RELEASE
FROM scratch AS release-stage
COPY --from=build-stage /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build-stage /techchallenge /techchallenge
COPY --from=build-stage /app/docs/swagger.yaml /docs/swagger.yaml

ENTRYPOINT ["/techchallenge"]
