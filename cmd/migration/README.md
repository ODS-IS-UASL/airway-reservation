# DB migrate command

## requirements
- [sql-migrate](https://github.com/rubenv/sql-migrate)

## How to create migration file
- Target folder is `database/migration/$schema_name`.
- The migration file must have a 4-digit sequential ${number}_file_name.sql (ex: 0005_hogefuga.sql), and the contents of the file must contain

``` sql
-- +migrate Up
-- Write the sql you want to apply

-- +migrate Down
-- Write the sql for down, but the basic down process is only for DDL, and can be left blank when adding seed data.
-- sql
````

## Usage

1. `docker-compose -f docker-compose.local.yml up -d postgres`
2. `go run ./cmd/migration/main.go schema apply -r $PWD/database/migration/uasl_reservation --schema-name uasl_reservation -u ${USER} -p 5432 -P ${PASSWORD} --dbname ${DB_NAME}`
3. `go run ./cmd/migration/main.go seed -u ${USER} -p 5432 -P ${PASSWORD} --dbname ${DB_NAME}`

## Notes

- UASL reservation related migrations are under `database/migration/uasl_reservation`.
