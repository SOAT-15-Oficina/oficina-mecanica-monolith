FROM postgres:18.3

COPY ./migrations/*.sql /tmp/migrations/
COPY ./seed-files/seed.sql /docker-entrypoint-initdb.d/999_seed.sql

RUN for f in /tmp/migrations/*.sql; do \
      sed '/^-- +goose Down/,$d' "$f" > "/docker-entrypoint-initdb.d/01_$(basename "$f")"; \
    done && rm -rf /tmp/migrations
