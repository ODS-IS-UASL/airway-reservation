# Builder ステージ
FROM golang:1.23-alpine AS builder

WORKDIR /app

# モジュールファイルをコピーして依存関係をダウンロード
COPY go.mod go.sum ./
RUN go mod download

# 必要なソースコードをコピー
COPY cmd/ ./cmd
COPY external/ ./external
COPY internal/ ./internal
COPY containers/ ./containers
COPY database/ ./database
COPY broker.crt ./

# バイナリをビルド
RUN go build -o /app/bin/main ./cmd/app/uasl_reservation/main.go

# 実行ステージ
FROM alpine:3.17 AS runner

WORKDIR /app

# dockerize ツールをインストール
ENV DOCKERIZE_VERSION v0.6.1
RUN apk add --no-cache wget \
    && wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz

# バイナリのみをコピー
COPY --from=builder /app/bin/main ./main
COPY --from=builder /app/database ./database
COPY broker.crt ./

# 必要に応じて環境変数を設定
ENV GIN_MODE=release

# 実行コマンド
# コンテナ起動時のエントリーポイント
ENTRYPOINT ["dockerize", "-timeout", "60s", "-wait", "tcp://aw-postgres:5432", "./main"]
