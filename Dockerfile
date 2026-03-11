FROM golang:1.25.7

WORKDIR /work

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -v -o main ./cmd/main.go

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["./main"]