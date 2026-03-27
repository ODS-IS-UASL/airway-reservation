FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd
COPY external/ ./external
COPY internal/ ./internal
COPY containers/ ./containers
COPY database/ ./database

RUN go build -o /app/bin/monthly_settlement ./cmd/app/monthly_settlement/main.go

FROM alpine:3.17 AS runner

WORKDIR /app

ENV DOCKERIZE_VERSION v0.6.1
RUN apk add --no-cache wget \
    && wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz

COPY --from=builder /app/bin/monthly_settlement ./monthly_settlement
COPY --from=builder /app/database ./database

ENTRYPOINT ["dockerize", "-timeout", "60s", "-wait", "tcp://aw-postgres:5432", "./monthly_settlement"]
