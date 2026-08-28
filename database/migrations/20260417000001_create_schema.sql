-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS "users" (
  "id" uuid PRIMARY KEY,
  "username" varchar(150) NOT NULL,
  "password_hash" varchar(255) NOT NULL,
  "role" varchar(30) NOT NULL,
  "created_at" timestamp NOT NULL,
  "updated_at" timestamp NOT NULL
);

CREATE TABLE IF NOT EXISTS "customers" (
  "id" uuid PRIMARY KEY,
  "name" varchar(150) NOT NULL,
  "document" varchar(20) UNIQUE NOT NULL,
  "document_type" varchar(10) NOT NULL,
  "phone" varchar(20),
  "email" varchar(150),
  "created_at" timestamp NOT NULL,
  "updated_at" timestamp NOT NULL
);

CREATE TABLE IF NOT EXISTS "vehicles" (
  "id" uuid PRIMARY KEY,
  "customer_id" uuid NOT NULL,
  "license_plate" varchar(10) UNIQUE NOT NULL,
  "brand" varchar(80) NOT NULL,
  "model" varchar(120) NOT NULL,
  "year" int NOT NULL,
  "created_at" timestamp NOT NULL,
  "updated_at" timestamp NOT NULL
);

CREATE TABLE IF NOT EXISTS "services" (
  "id" uuid PRIMARY KEY,
  "title" varchar(120) NOT NULL,
  "description" text,
  "price_cents" int NOT NULL,
  "estimated_time_minutes" int NOT NULL,
  "active" boolean NOT NULL DEFAULT true,
  "created_at" timestamp NOT NULL,
  "updated_at" timestamp NOT NULL
);

CREATE TABLE IF NOT EXISTS "supplies" (
  "id" uuid PRIMARY KEY,
  "title" varchar(120) NOT NULL,
  "type" varchar(20) NOT NULL,
  "price_cents" int NOT NULL,
  "stock_quantity" int NOT NULL DEFAULT 0,
  "minimum_stock" int NOT NULL DEFAULT 0,
  "active" boolean NOT NULL DEFAULT true,
  "created_at" timestamp NOT NULL,
  "updated_at" timestamp NOT NULL
);

CREATE TABLE IF NOT EXISTS "work_orders" (
  "id" uuid PRIMARY KEY,
  "code" varchar(30) UNIQUE NOT NULL,
  "title" varchar(150) NOT NULL,
  "description" text,
  "customer_id" uuid NOT NULL,
  "vehicle_id" uuid NOT NULL,
  "opened_by_user_id" uuid NOT NULL,
  "assigned_technician_id" uuid,
  "status" varchar(30) NOT NULL,
  "total_estimated_price_cents" int NOT NULL DEFAULT 0,
  "received_at" timestamp NOT NULL,
  "quote_sent_at" timestamp,
  "approved_at" timestamp,
  "started_at" timestamp,
  "finished_at" timestamp,
  "delivered_at" timestamp,
  "created_at" timestamp NOT NULL,
  "updated_at" timestamp NOT NULL
);

CREATE TABLE IF NOT EXISTS "work_order_services" (
  "id" uuid PRIMARY KEY,
  "work_order_id" uuid NOT NULL,
  "service_id" uuid NOT NULL,
  "service_title_snapshot" varchar(120) NOT NULL,
  "service_description_snapshot" text,
  "service_price_cents_snapshot" int NOT NULL,
  "service_estimated_time_minutes_snapshot" int NOT NULL,
  "approval_status" varchar(20) NOT NULL DEFAULT 'PENDENTE',
  "status" varchar(30) NOT NULL DEFAULT 'PENDENTE',
  "started_at" timestamp,
  "finished_at" timestamp,
  "created_at" timestamp NOT NULL,
  "updated_at" timestamp NOT NULL
);

CREATE TABLE IF NOT EXISTS "work_order_service_supplies" (
  "id" uuid PRIMARY KEY,
  "work_order_service_id" uuid NOT NULL,
  "supply_id" uuid NOT NULL,
  "supply_title_snapshot" varchar(120) NOT NULL,
  "supply_price_cents_snapshot" int NOT NULL,
  "supply_quantity" int NOT NULL,
  "created_at" timestamp NOT NULL,
  "updated_at" timestamp NOT NULL
);

CREATE TABLE IF NOT EXISTS "work_order_service_status_history" (
  "id" uuid PRIMARY KEY,
  "work_order_service_id" uuid NOT NULL,
  "status" varchar(30) NOT NULL,
  "changed_by_user_id" uuid,
  "note" text,
  "changed_at" timestamp NOT NULL
);

-- ==================== INDEXES ====================
CREATE INDEX IF NOT EXISTS idx_vehicles_customer_id ON "vehicles" ("customer_id");
CREATE INDEX IF NOT EXISTS idx_vehicles_license_plate ON "vehicles" ("license_plate");

CREATE INDEX IF NOT EXISTS idx_work_orders_code ON "work_orders" ("code");
CREATE INDEX IF NOT EXISTS idx_work_orders_customer_id ON "work_orders" ("customer_id");
CREATE INDEX IF NOT EXISTS idx_work_orders_vehicle_id ON "work_orders" ("vehicle_id");
CREATE INDEX IF NOT EXISTS idx_work_orders_status ON "work_orders" ("status");

CREATE INDEX IF NOT EXISTS idx_work_order_services_work_order_id ON "work_order_services" ("work_order_id");
CREATE INDEX IF NOT EXISTS idx_work_order_services_service_id ON "work_order_services" ("service_id");
CREATE INDEX IF NOT EXISTS idx_work_order_services_approval_status ON "work_order_services" ("approval_status");
CREATE INDEX IF NOT EXISTS idx_work_order_services_status ON "work_order_services" ("status");

CREATE INDEX IF NOT EXISTS idx_work_order_service_supplies_wos_id ON "work_order_service_supplies" ("work_order_service_id");
CREATE INDEX IF NOT EXISTS idx_work_order_service_supplies_supply_id ON "work_order_service_supplies" ("supply_id");
CREATE UNIQUE INDEX IF NOT EXISTS idx_work_order_service_supplies_wos_supply ON "work_order_service_supplies" ("work_order_service_id", "supply_id");

CREATE INDEX IF NOT EXISTS idx_work_order_service_status_history_wos_id ON "work_order_service_status_history" ("work_order_service_id");
CREATE INDEX IF NOT EXISTS idx_work_order_service_status_history_changed_at ON "work_order_service_status_history" ("changed_at");

-- ==================== FOREIGN KEYS ====================
DO $$ BEGIN
  ALTER TABLE "vehicles" ADD CONSTRAINT fk_vehicles_customer
    FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") DEFERRABLE INITIALLY IMMEDIATE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  ALTER TABLE "work_orders" ADD CONSTRAINT fk_work_orders_customer
    FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") DEFERRABLE INITIALLY IMMEDIATE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  ALTER TABLE "work_orders" ADD CONSTRAINT fk_work_orders_vehicle
    FOREIGN KEY ("vehicle_id") REFERENCES "vehicles" ("id") DEFERRABLE INITIALLY IMMEDIATE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  ALTER TABLE "work_orders" ADD CONSTRAINT fk_work_orders_opened_by
    FOREIGN KEY ("opened_by_user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  ALTER TABLE "work_orders" ADD CONSTRAINT fk_work_orders_technician
    FOREIGN KEY ("assigned_technician_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  ALTER TABLE "work_order_services" ADD CONSTRAINT fk_wos_work_order
    FOREIGN KEY ("work_order_id") REFERENCES "work_orders" ("id") DEFERRABLE INITIALLY IMMEDIATE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  ALTER TABLE "work_order_services" ADD CONSTRAINT fk_wos_service
    FOREIGN KEY ("service_id") REFERENCES "services" ("id") DEFERRABLE INITIALLY IMMEDIATE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  ALTER TABLE "work_order_service_supplies" ADD CONSTRAINT fk_wosi_work_order_service
    FOREIGN KEY ("work_order_service_id") REFERENCES "work_order_services" ("id") DEFERRABLE INITIALLY IMMEDIATE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  ALTER TABLE "work_order_service_supplies" ADD CONSTRAINT fk_wosi_supply
    FOREIGN KEY ("supply_id") REFERENCES "supplies" ("id") DEFERRABLE INITIALLY IMMEDIATE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  ALTER TABLE "work_order_service_status_history" ADD CONSTRAINT fk_wosh_work_order_service
    FOREIGN KEY ("work_order_service_id") REFERENCES "work_order_services" ("id") DEFERRABLE INITIALLY IMMEDIATE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  ALTER TABLE "work_order_service_status_history" ADD CONSTRAINT fk_wosh_changed_by
    FOREIGN KEY ("changed_by_user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  ALTER TABLE "users" ADD CONSTRAINT users_username_key UNIQUE ("username");
EXCEPTION WHEN duplicate_table THEN NULL; END $$;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "work_order_service_status_history";
DROP TABLE IF EXISTS "work_order_service_supplies";
DROP TABLE IF EXISTS "work_order_services";
DROP TABLE IF EXISTS "work_orders";
DROP TABLE IF EXISTS "supplies";
DROP TABLE IF EXISTS "services";
DROP TABLE IF EXISTS "vehicles";
DROP TABLE IF EXISTS "customers";
DROP TABLE IF EXISTS "users";
-- +goose StatementEnd
