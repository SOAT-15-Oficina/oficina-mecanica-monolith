#BUILD GO APP
FROM golang:1.26.1-alpine AS build-stage
WORKDIR /app

# SHA do commit, gravado no binario pelo linker. E de la que sai o campo
# `version` de toda linha de log e a tag `version` do APM.
#
# Vem do LINKER e nao de uma variavel de ambiente do container: variavel pode ser
# trocada sem trocar a imagem, e o campo passaria a mentir sobre qual codigo
# esta rodando -- que e exatamente a pergunta que ele existe para responder
# quando uma regressao aparece no painel.
ARG VERSION=dev

COPY go.mod go.sum ./
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false \
    -ldflags="-s -w -X github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/observability.version=${VERSION}" \
    -o /techchallenge ./cmd/api/main.go

# SETUP CONTAINER RELEASE
FROM scratch AS release-stage
COPY --from=build-stage /techchallenge /techchallenge
COPY --from=build-stage /app/docs/swagger.yaml /docs/swagger.yaml

ENTRYPOINT ["/techchallenge"]
