# syntax=docker/dockerfile:1
FROM golang:1.25.7-alpine AS build_stage

WORKDIR /app

ARG SERVICE_NAME=media

COPY go.mod go.sum ./
RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/service ./cmd/${SERVICE_NAME}

FROM alpine:3.21 AS run_stage

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=build_stage /app/service ./service
COPY --from=build_stage /app/configs ./configs
RUN chmod +x ./service

EXPOSE 8003/tcp

ENTRYPOINT ["./service"]
