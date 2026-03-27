## ドローン航路予約システム

## 概要

ドローン航路予約システムは複数のAPIで構成されます。
- 航路予約API（仮押さえ）
- 航路予約確定API
- 航路予約取消API（運航事業者向け）
- 航路予約撤回API（航路運営者向け）
- （内部処理用）航路予約削除API
- 航路予約一覧取得API（運航事業者向け）
- 航路予約一覧取得API（航路運営者向け）
- 航路予約詳細取得API
- 空き状況確認API
- 料金見積もりAPI
- （内部処理用）予約完了通知API


### ディレクトリ構成

```
$ tree -L 2 -I ".aws-sam|.DS_Store|.git|tmp|.env" -a
.
├── .dockerignore                  # Dockerfileでコピーしないファイルを定義
├── .env.local                     # 環境変数ファイル（Local確認用。コンテナとしてデプロイする際は、.envにリネーム）
├── .gitignore
├── Makefile
├── README.md
├── cmd
│   ├── app                        # アプリ起動時に実行される（uasl_reservation）
│   └── migration                  # DBマイグレーションの設定と実行
├── containers
│   └── uasl_reservation           # Dockerfile・air設定
├── database
│   ├── dbdoc
│   ├── migration                  # SQLマイグレーション・シードデータ
│   └── tbls.yml
├── doc
│   └── schema.yaml
├── docker-compose.local.yml
├── docker-compose.yml
├── external
│   └── uasl                       # 外部連携クライアント
├── go.mod
├── go.sum
├── internal
│   ├── app                        # ハンドラー・ユースケース・ドメイン（uasl_reservation）
│   └── pkg                        # 共通パッケージ（DB、logger、mqtt、config）
├── LICENSE.txt
├── Makefile
├── proto
│   └── protobuf
└── README.md
```

### テーブル構成

[README.md](database/dbdoc/README.md)

## 開発環境構築手順

### ソフトウェアバージョン

- Go v1.23
- PostgreSQL 15
- Docker v20.10以上

### リポジトリクローン

```
git clone git@github.com:ODS-IS-UASL/uasl-reservation.git
```

### 環境変数ファイル設定（コンテナデプロイ用）

```
cp -pr .env.local .env
```

.env
USE_MQTT=true　// ./.broker.crt配置が必須

※ローカル環境での動作確認ではUSE_MQTT=falseとしてMQTTブローカーへのPublish処理をスキップ可能。

### go モジュールインストール

go.mod ファイルに書いてあるモジュールがローカルにインストールされる

```
go mod tidy -v
```

### RDS のマイグレーション

[README.md](cmd/migration/README.md)

ローカル環境のテーブルやカラムを作成更新する

```
make migrate
```

### シードデータの投入

アプリ起動時に環境変数 `ENABLE_SEED=true` を設定すると、マイグレーション実行後に自動でシードデータ（`database/migration/seed/seed_data.sql`）が投入されます。

### broker.crt ファイルの配置

API 実行時、主に create/update の際に mqtt 通信で publish を実施するため、ブローカーに対する証明書を配置しておく。<br>
なお\*.cert の拡張子は git 管理しない。<br>
MQTTブローカーへのイベントパブリッシュを行わない場合は、.envのUSE_MQTT=falseとすること。

### ローカルで動作確認する方法

#### コンテナの起動

以下のコマンドでDocker環境を構築し、アプリケーションを起動してください。

```
# ローカル用
$ make docker-local-up

# 外部連携用（本番環境）
$ make docker-up
```

#### 月次精算バッチの単体実行

月次精算処理だけを 1 回実行したい場合は、以下のコマンドを使用してください。

```bash
$ make monthly-settlement
```

このコマンドは `monthly_settlement` コンテナを起動してバッチを 1 回実行し、完了後にコンテナを終了します。

起動完了後、以下のログが表示されます：

```
データベースとの接続に成功しました

   ____    __
  / __/___/ /  ___
 / _// __/ _ \/ _ \
/___/\__/_//_/\___/ v4.11.4
High performance, minimalist Go web framework
https://echo.labstack.com
____________________________________O/_______
                                    O\
⇨ http server started on [::]:8080
```

#### APIの動作確認

Docker起動完了後、PostmanまたはcURLコマンドを使用してAPIをテストできます。以下は主要なAPIの実行例です：

```bash
# 航路予約API（仮押さえ）
$ curl -i -X POST 'http://localhost:8080/v1/uaslReservations' \
  -H "Content-Type: application/json" \
  -d '{
    "operatorId": "7caceea5-c12b-4655-96fe-12db625c7fd7",
    "uaslSections": [
      {
        "uaslId": "fbbefdf2-b69f-46b8-80a1-4e41a6812fbb_6c216fcc-4a7a-4a51-9c8c-637b6d80730a",
        "uaslSectionId": "88508bd0-be86-4068-9972-2155d41cfc60",
        "startAt": "2026-04-01T10:00:00+09:00",
        "endAt": "2026-04-01T11:00:00+09:00"
      }
    ],
    "vehicles": [
      {
        "aircraftInfo": {
          "aircraftInfoId": 1,
          "registrationId": "JU1234567890AB",
          "maker": "メーカー1",
          "modelNumber": "型番1_1",
          "name": "機体1",
          "type": "回転翼航空機（ヘリコプター）",
          "length": 950
        }
      }
    ]
  }'

# 航路予約確定API
$ curl -i -X PUT 'http://localhost:8080/v1/uaslReservations/{requestId}/confirm' \
  -H "Content-Type: application/json" \
  -d '{"isInterConnect": false}'

# 航路予約取消API（運航事業者向け）
$ curl -i -X PUT 'http://localhost:8080/v1/uaslReservations/{requestId}/cancel' \
  -H "Content-Type: application/json" \
  -d '{"isInterConnect": false}'

# 航路予約撤回API（航路運営者向け）
$ curl -i -X PUT 'http://localhost:8080/v1/admin/uaslReservations/{requestId}/rescind' \
  -H "Content-Type: application/json" \
  -d '{"isInterConnect": false}'

# 航路予約削除API
$ curl -i -X DELETE 'http://localhost:8080/v1/uaslReservations/{requestId}'

# 航路予約一覧取得API（運航事業者向け）
$ curl -i -X GET 'http://localhost:8080/v1/operator/{operatorId}/uaslReservations'
$ curl -i -X GET 'http://localhost:8080/v1/operator/{operatorId}/uaslReservations?page=1'

# 航路予約一覧取得API（航路運営者向け）
$ curl -i -X GET 'http://localhost:8080/v1/admin/uaslReservations'
$ curl -i -X GET 'http://localhost:8080/v1/admin/uaslReservations?page=1'

# 航路予約詳細取得API
$ curl -i -X GET 'http://localhost:8080/v1/uaslReservations/{requestId}'

# 空き状況確認API
$ curl -i -X POST 'http://localhost:8080/v1/uaslReservations/availability' \
  -H "Content-Type: application/json" \
  -d '{
    "uaslSections": [
      {
        "uaslId": "fbbefdf2-b69f-46b8-80a1-4e41a6812fbb_6c216fcc-4a7a-4a51-9c8c-637b6d80730a",
        "uaslSectionId": "88508bd0-be86-4068-9972-2155d41cfc60"
      }
    ]
  }'

# 料金見積もりAPI
$ curl -i -X POST 'http://localhost:8080/v1/uaslReservations/estimate' \
  -H "Content-Type: application/json" \
  -d '{
    "uaslSections": [
      {
        "uaslSectionId": "88508bd0-be86-4068-9972-2155d41cfc60",
        "startAt": "2026-04-01T10:00:00+09:00",
        "endAt": "2026-04-01T11:00:00+09:00"
      }
    ]
  }'

# （内部処理用）予約完了通知API
$ curl -i -X POST 'http://localhost:8080/v1/uaslReservations/completed' \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": {{ uuid }},
    "reservationId": {{ uuid }},
    "operatorId": "7caceea5-c12b-4655-96fe-12db625c7fd7",
    "uaslId": "fbbefdf2-b69f-46b8-80a1-4e41a6812fbb_6c216fcc-4a7a-4a51-9c8c-637b6d80730a",
    "administratorId": "bf5722bc-5eaf-484c-9cf1-6b852ddebb1c",
    "flightPurpose": "河川監視",
    "status": "RESERVED",
    "subTotalAmount": 9000,
    "reservedAt": "2026-03-25T05:00:00Z",
    "estimatedAt": "2026-03-25T05:00:00Z",
    "updatedAt": "2026-03-25T05:00:00Z",
    "uaslSections": [
      {
        "uaslSectionId": "88508bd0-be86-4068-9972-2155d41cfc60",
        "sequence": 1,
        "startAt": "2026-04-01T10:00:00+09:00",
        "endAt": "2026-04-01T11:00:00+09:00",
        "amount": 9000
      }
    ],
    "operatingAircrafts": [
      {
        "aircraftInfoId": 1,
        "registrationId": "JU1234567890AB",
        "maker": "メーカー1",
        "modelNumber": "型番1_1",
        "name": "機体1",
        "type": "回転翼航空機（ヘリコプター）",
        "length": 950
      }
    ],
    "conformityAssessmentResults": [
      {
        "uaslSectionId": "88508bd0-be86-4068-9972-2155d41cfc60",
        "evaluationResults": "true",
        "type": "null",
        "aircraftInfo": {
          "aircraftInfoId": 1,
          "registrationId": "JU1234567890AB",
          "maker": "メーカー1",
          "modelNumber": "型番1_1",
          "name": "機体1",
          "type": "回転翼航空機（ヘリコプター）",
          "length": 950
        }
      }
    ]
  }'
```

#### 月次精算バッチの動作確認

月次精算バッチは、前月分の予約を集計して精算レコードを生成し、決済APIへ請求を行うバッチ処理です。

##### 必要な環境変数

| 環境変数 | 説明 | 例 |
|---------|------|-----|
| `POSTGRES_HOST` | DBホスト | `localhost` |
| `POSTGRES_PORT` | DBポート | `5432` |
| `POSTGRES_USER` | DBユーザー | `myadmin` |
| `POSTGRES_PASSWORD` | DBパスワード | `admin` |
| `POSTGRES_DB` | DB名 | `postgres` |
| `OURANOS_L3_API_KEY` | L3認証APIキー | ※本番環境のみ必要 |
| `L3_BASE_URL` | L3認証ベースURL | ※本番環境のみ必要 |
| `PAYMENT_SERVICE_ID` | 決済サービスID | 省略時はデフォルト値を使用 |

##### ビルドと実行

```bash
# ビルド
$ go build -o monthly_settlement ./cmd/app/monthly_settlement/

# 実行（ローカル環境）
$ POSTGRES_HOST=localhost \
  POSTGRES_PORT=5432 \
  POSTGRES_USER=myadmin \
  POSTGRES_PASSWORD=admin \
  POSTGRES_DB=postgres \
  ./monthly_settlement
```

##### 処理フロー

1. **集計（Phase 1）**: 前月分のステータスが `RESERVED` の予約を `ex_administrator_id × operator_id` 単位でグループ化し、`uasl_settlements` テーブルにUPSERT
2. **未提出取得（Phase 2）**: `submitted_at IS NULL` の精算レコードを取得
3. **決済API呼び出し（Phase 3）**: L3認証トークンを取得し、各精算レコードの金額確定APIを呼び出し。成功後に `payment_confirmed_at` → `submitted_at` の順で更新（二重決済防止）

##### ローカル環境での確認（集計フェーズのみ）

ローカル環境ではL3認証APIキーが不要な集計フェーズ（Phase 1）のみ確認できます。実行後に以下で精算レコードを確認してください。

```bash
# Dockerコンテナ経由でDBを確認
$ docker exec -i aw-postgres psql -U myadmin -d postgres \
  -c "SELECT id, ex_administrator_id, operator_id, target_year_month, total_amount, submitted_at FROM uasl_reservation.uasl_settlements;"
```

## 新規テーブルを作成する場合

database ディレクトリにサーバー名のディレクトリを追加
必要な sql ファイルを作成したら Makefile の migrate エリアに以下のように名前を追加

```Makefile
migrate:
    make uasl_reservation
```

その後作成・更新を再度を行う

```bash
make migrate
```

問題なければ最後に doc を更新する

```bash
tbls doc -c database/tbls.yml --rm-dist
```

## 外部にコンテナを連携する手順

外部向けの docker イメージの作成と立ち上げ

```bash
make docker-up
```

tar ファイルの作成

```bash
docker save -o ar.tar uasl_reservation-app monthly_settlement-app postgis/postgis:15-3.4
```

tar ファイルと env ファイルと compose ファイルを圧縮

```bash
tar -czvf containers.tar.gz ar.tar docker-compose.yml .env.local
```

## 外部向けコンテナイメージを実行する方法

圧縮ファイルを適当なディレクトリに移して展開

```bash
tar -xzvf containers.tar.gz
```

tar ファイルのロード

```bash
docker load < ar.tar
```

コンテナの立ち上げ

```bash
docker compose -f docker-compose.yml up
```

API の実行

```bash
curl -i -X GET http://localhost:8080/v1/uaslReservations/{requestId}
```

## 問合せ及び要望に関して

- 本リポジトリは現状は主に配布目的の運用となるため、IssueやPull Requestに関しては受け付けておりません。

## ライセンス

- 本リポジトリはMITライセンスで提供されています。
- 本リポジトリ内のソースコードの著作権は、KDDIスマートドローン株式会社に帰属します。

## 免責事項

- 本リポジトリの内容は予告なく変更・削除する可能性があります。
- 本リポジトリの利用により生じた損失及び損害等について、いかなる責任も負わないものとします。
