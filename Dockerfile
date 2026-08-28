#BUILD GO APP
FROM golang:1.26.1-alpine AS build-stage
WORKDIR /app
COPY go.mod go.sum ./
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false -ldflags="-s -w" -o /techchallenge ./cmd/api/main.go

# SETUP CONTAINER RELEASE
FROM scratch AS release-stage
COPY --from=build-stage /techchallenge /techchallenge
COPY --from=build-stage /app/docs/swagger.yaml /docs/swagger.yaml

ENTRYPOINT ["/techchallenge"]
