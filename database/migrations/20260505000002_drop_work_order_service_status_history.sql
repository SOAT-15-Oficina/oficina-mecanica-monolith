-- +goose Up
-- +goose StatementBegin
DROP TABLE IF EXISTS "work_order_service_status_history";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS "work_order_service_status_history" (
  "id" uuid PRIMARY KEY,
  "work_order_service_id" uuid NOT NULL,
  "status" varchar(30) NOT NULL,
  "changed_by_user_id" uuid,
  "note" text,
  "changed_at" timestamp NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_work_order_service_status_history_wos_id ON "work_order_service_status_history" ("work_order_service_id");
CREATE INDEX IF NOT EXISTS idx_work_order_service_status_history_changed_at ON "work_order_service_status_history" ("changed_at");

DO $$ BEGIN
  ALTER TABLE "work_order_service_status_history" ADD CONSTRAINT fk_wosh_work_order_service
    FOREIGN KEY ("work_order_service_id") REFERENCES "work_order_services" ("id") DEFERRABLE INITIALLY IMMEDIATE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  ALTER TABLE "work_order_service_status_history" ADD CONSTRAINT fk_wosh_changed_by
    FOREIGN KEY ("changed_by_user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- +goose StatementEnd
