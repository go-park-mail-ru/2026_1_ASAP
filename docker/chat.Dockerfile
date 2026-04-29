FROM golang:1.25.7-alpine AS build_stage

WORKDIR /app

ARG SERVICE_NAME=chat

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/service ./cmd/${SERVICE_NAME}

FROM alpine:3.21 AS run_stage

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=build_stage /app/service ./service
COPY --from=build_stage /app/configs ./configs
RUN chmod +x ./service

EXPOSE 8004/tcp

ENTRYPOINT ["./service"]
