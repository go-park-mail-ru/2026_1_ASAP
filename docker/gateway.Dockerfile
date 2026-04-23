FROM golang:1.25.7-alpine AS build_stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/gateway ./cmd/gateway

FROM alpine:3.21 AS run_stage

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=build_stage /app/gateway ./gateway
COPY --from=build_stage /app/configs ./configs
RUN chmod +x ./gateway

EXPOSE 8080/tcp

ENTRYPOINT ["./gateway"]
