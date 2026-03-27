-- +migrate Up
CREATE SCHEMA IF NOT EXISTS uasl_reservation;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";
CREATE TYPE uasl_reservation.reservation_status AS ENUM (
    'PENDING',
    'RESERVED',
    'CANCELED',
    'RESCINDED',
    'INHERITED'
);
CREATE TYPE uasl_reservation.uasl_resource_type AS ENUM ('VEHICLE', 'PORT', 'PAYLOAD');
CREATE TYPE uasl_reservation.external_resource_type AS ENUM ('VEHICLE', 'PORT', 'PAYLOAD');
CREATE TYPE uasl_reservation.uasl_section_status AS ENUM ('AVAILABLE', 'CLOSED');

CREATE TABLE uasl_reservation.operations (
    "id" uuid PRIMARY KEY DEFAULT uuid_generate_v4()
);

CREATE TABLE uasl_reservation.uasl_operators (
    "id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    "name" varchar(255),
    "created_at" timestamp(0) with time zone NOT NULL DEFAULT now(),
    "updated_at" timestamp(0) with time zone NOT NULL DEFAULT now()
);

CREATE TABLE uasl_reservation.uasl_administrators (
    "id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    "ex_administrator_id" varchar(255) NOT NULL UNIQUE,
    "business_number" varchar(255),
    "name" varchar(255) NOT NULL,
    "is_internal" boolean NOT NULL DEFAULT false,
    "external_services" jsonb,
    "created_at" timestamp(0) with time zone NOT NULL DEFAULT now(),
    "updated_at" timestamp(0) with time zone NOT NULL DEFAULT now()
);

CREATE TABLE uasl_reservation.external_uasl_definitions (
    "id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    "ex_uasl_section_id" varchar(255) NOT NULL UNIQUE,
    "ex_uasl_id" varchar(255),
    "ex_administrator_id" varchar(255) NOT NULL,
    "geometry" geometry NOT NULL,
    "point_ids" text[] NOT NULL,
    "flight_purpose" varchar(255),
    "price_info" jsonb,
    "price_timezone" varchar(50) NOT NULL DEFAULT 'UTC',
    "price_version" integer NOT NULL DEFAULT 1,
    "status" uasl_reservation.uasl_section_status NOT NULL DEFAULT 'AVAILABLE',
    "created_at" timestamp(0) with time zone NOT NULL DEFAULT now(),
    "updated_at" timestamp(0) with time zone NOT NULL DEFAULT now(),
    CONSTRAINT "fk_external_uasl_definitions_administrator_id"
        FOREIGN KEY ("ex_administrator_id")
        REFERENCES uasl_reservation.uasl_administrators ("ex_administrator_id")
);

CREATE TABLE uasl_reservation.external_uasl_resources (
    "id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    "name" varchar(255) NOT NULL,
    "resource_id" uuid,
    "ex_uasl_section_id" varchar(255),
    "resource_type" uasl_reservation.uasl_resource_type NOT NULL,
    "ex_resource_id" varchar(255) NOT NULL UNIQUE,
    "organization_id" uuid,
    "estimated_price_per_minute" jsonb,
    "aircraft_info" jsonb,
    "created_at" timestamp(0) with time zone NOT NULL DEFAULT now(),
    "updated_at" timestamp(0) with time zone NOT NULL DEFAULT now()
);

CREATE TABLE uasl_reservation.uasl_reservations (
    "id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    "parent_uasl_reservation_id" uuid,
    "request_id" uuid NOT NULL,
    "ex_uasl_section_id" varchar(255),
    "ex_uasl_id" varchar(255),
    "ex_administrator_id" varchar(255),
    "start_at" timestamp(0) with time zone NOT NULL,
    "end_at" timestamp(0) with time zone NOT NULL,
    "accepted_at" timestamp(0) with time zone,
    "airspace_id" uuid NOT NULL,
    "ex_reserved_by" uuid,
    "organization_id" uuid,
    "project_id" uuid,
    "operation_id" uuid,
    "status" uasl_reservation.reservation_status NOT NULL,
    "pricing_rule_version" integer,
    "amount" integer,
    "estimated_at" timestamp(0) with time zone,
    "fixed_at" timestamp(0) with time zone,
    "sequence" integer,
    "conformity_assessment" jsonb,
    "destination_reservations" jsonb,
    "created_at" timestamp(0) with time zone NOT NULL DEFAULT now(),
    "updated_at" timestamp(0) with time zone NOT NULL DEFAULT now(),
    CONSTRAINT "fk_uasl_reservations_parent_id"
        FOREIGN KEY ("parent_uasl_reservation_id")
        REFERENCES uasl_reservation.uasl_reservations ("id"),
    CONSTRAINT "fk_uasl_reservations_operation_id"
        FOREIGN KEY ("operation_id")
        REFERENCES uasl_reservation.operations ("id")
);

CREATE TABLE uasl_reservation.external_resource_reservations (
    "id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    "request_id" uuid NOT NULL,
    "ex_administrator_id" varchar(255) NOT NULL,
    "ex_reservation_id" varchar(255),
    "ex_resource_id" varchar(255) NOT NULL,
    "start_at" timestamp(0) with time zone,
    "end_at" timestamp(0) with time zone,
    "usage_type" integer,
    "amount" integer,
    "resource_type" uasl_reservation.external_resource_type NOT NULL,
    "created_at" timestamp(0) with time zone NOT NULL DEFAULT now(),
    "updated_at" timestamp(0) with time zone NOT NULL DEFAULT now()
);

CREATE TABLE uasl_reservation.uasl_settlements (
    "id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    "ex_administrator_id" varchar(255) NOT NULL,
    "operator_id" uuid NOT NULL,
    "target_year_month" date NOT NULL,
    "uasl_reservation_ids" uuid[],
    "total_amount" integer NOT NULL,
    "tax_rate" decimal(5,4) NOT NULL,
    "payment_confirmed_at" timestamp(0) with time zone,
    "submitted_at" timestamp(0) with time zone,
    "billed_at" timestamp(0) with time zone,
    "payment_due_at" timestamp(0) with time zone,
    "paid_at" timestamp(0) with time zone,
    "created_at" timestamp(0) with time zone NOT NULL DEFAULT now(),
    "updated_at" timestamp(0) with time zone NOT NULL DEFAULT now(),
    CONSTRAINT "fk_uasl_settlements_ex_administrator_id"
        FOREIGN KEY ("ex_administrator_id")
        REFERENCES uasl_reservation.uasl_administrators ("ex_administrator_id")
);

CREATE INDEX "idx_uasl_administrators_ex_administrator_id"
    ON uasl_reservation.uasl_administrators ("ex_administrator_id");
CREATE INDEX "idx_external_uasl_definitions_ex_uasl_section_id"
    ON uasl_reservation.external_uasl_definitions ("ex_uasl_section_id");
CREATE INDEX "idx_external_uasl_definitions_ex_administrator_id"
    ON uasl_reservation.external_uasl_definitions ("ex_administrator_id");
CREATE INDEX "idx_external_uasl_definitions_geometry"
    ON uasl_reservation.external_uasl_definitions USING GIST ("geometry");
CREATE INDEX "idx_external_uasl_definitions_point_ids"
    ON uasl_reservation.external_uasl_definitions USING GIN ("point_ids");
CREATE INDEX "idx_external_uasl_resources_ex_resource_id"
    ON uasl_reservation.external_uasl_resources ("ex_resource_id");
CREATE INDEX "idx_external_uasl_resources_section_id"
    ON uasl_reservation.external_uasl_resources ("ex_uasl_section_id");
CREATE INDEX "idx_external_uasl_resources_organization_id"
    ON uasl_reservation.external_uasl_resources ("organization_id");
CREATE INDEX "idx_uasl_reservations_request_id"
    ON uasl_reservation.uasl_reservations ("request_id");
CREATE INDEX "idx_uasl_reservations_ex_administrator_id"
    ON uasl_reservation.uasl_reservations ("ex_administrator_id");
CREATE INDEX "idx_uasl_reservations_ex_uasl_section_id"
    ON uasl_reservation.uasl_reservations ("ex_uasl_section_id");
CREATE INDEX "idx_uasl_reservations_organization_id"
    ON uasl_reservation.uasl_reservations ("organization_id");
CREATE INDEX "idx_uasl_reservations_project_id"
    ON uasl_reservation.uasl_reservations ("project_id");
CREATE INDEX "idx_uasl_reservations_time_range"
    ON uasl_reservation.uasl_reservations ("start_at", "end_at");
CREATE INDEX "idx_external_resource_reservations_request_id"
    ON uasl_reservation.external_resource_reservations ("request_id");
CREATE INDEX "idx_external_resource_reservations_ex_administrator_id"
    ON uasl_reservation.external_resource_reservations ("ex_administrator_id");
CREATE INDEX "idx_external_resource_reservations_ex_resource_id"
    ON uasl_reservation.external_resource_reservations ("ex_resource_id");
CREATE UNIQUE INDEX "idx_admin_operator_month"
    ON uasl_reservation.uasl_settlements ("ex_administrator_id", "operator_id", "target_year_month");
CREATE INDEX "idx_uasl_settlements_ex_administrator_id"
    ON uasl_reservation.uasl_settlements ("ex_administrator_id");
CREATE INDEX "idx_uasl_settlements_operator_id"
    ON uasl_reservation.uasl_settlements ("operator_id");
CREATE INDEX "idx_uasl_settlements_target_year_month"
    ON uasl_reservation.uasl_settlements ("target_year_month");

-- +migrate Down
DROP INDEX IF EXISTS uasl_reservation.idx_uasl_settlements_target_year_month;
DROP INDEX IF EXISTS uasl_reservation.idx_uasl_settlements_operator_id;
DROP INDEX IF EXISTS uasl_reservation.idx_uasl_settlements_ex_administrator_id;
DROP INDEX IF EXISTS uasl_reservation.idx_admin_operator_month;
DROP INDEX IF EXISTS uasl_reservation.idx_external_resource_reservations_ex_resource_id;
DROP INDEX IF EXISTS uasl_reservation.idx_external_resource_reservations_ex_administrator_id;
DROP INDEX IF EXISTS uasl_reservation.idx_external_resource_reservations_request_id;
DROP INDEX IF EXISTS uasl_reservation.idx_uasl_reservations_time_range;
DROP INDEX IF EXISTS uasl_reservation.idx_uasl_reservations_project_id;
DROP INDEX IF EXISTS uasl_reservation.idx_uasl_reservations_organization_id;
DROP INDEX IF EXISTS uasl_reservation.idx_uasl_reservations_ex_uasl_section_id;
DROP INDEX IF EXISTS uasl_reservation.idx_uasl_reservations_ex_administrator_id;
DROP INDEX IF EXISTS uasl_reservation.idx_uasl_reservations_request_id;
DROP INDEX IF EXISTS uasl_reservation.idx_external_uasl_resources_organization_id;
DROP INDEX IF EXISTS uasl_reservation.idx_external_uasl_resources_section_id;
DROP INDEX IF EXISTS uasl_reservation.idx_external_uasl_resources_ex_resource_id;
DROP INDEX IF EXISTS uasl_reservation.idx_external_uasl_definitions_point_ids;
DROP INDEX IF EXISTS uasl_reservation.idx_external_uasl_definitions_geometry;
DROP INDEX IF EXISTS uasl_reservation.idx_external_uasl_definitions_ex_administrator_id;
DROP INDEX IF EXISTS uasl_reservation.idx_external_uasl_definitions_ex_uasl_section_id;
DROP INDEX IF EXISTS uasl_reservation.idx_uasl_administrators_ex_administrator_id;

DROP TABLE IF EXISTS uasl_reservation.uasl_settlements;
DROP TABLE IF EXISTS uasl_reservation.external_resource_reservations;
DROP TABLE IF EXISTS uasl_reservation.uasl_reservations;
DROP TABLE IF EXISTS uasl_reservation.external_uasl_resources;
DROP TABLE IF EXISTS uasl_reservation.external_uasl_definitions;
DROP TABLE IF EXISTS uasl_reservation.uasl_administrators;
DROP TABLE IF EXISTS uasl_reservation.uasl_operators;
DROP TABLE IF EXISTS uasl_reservation.operations;

DROP TYPE IF EXISTS uasl_reservation.uasl_section_status;
DROP TYPE IF EXISTS uasl_reservation.external_resource_type;
DROP TYPE IF EXISTS uasl_reservation.uasl_resource_type;
DROP TYPE IF EXISTS uasl_reservation.reservation_status;
