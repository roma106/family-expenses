FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/app

FROM alpine:3.23

WORKDIR /app

COPY --from=builder /server ./server
COPY --from=builder /src/frontend ./frontend
COPY --from=builder /src/migrations ./migrations

EXPOSE 8080

CMD ["./server"]