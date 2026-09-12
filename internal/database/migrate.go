package database

import (
	"context"
	"log/slog"
	"os"

	dbmigrations "github.com/SOAT-15-Oficina/oficina-mecanica-monolith/database"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/observability"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

func RunMigrations(pool *pgxpool.Pool) {
	ctx := context.Background()

	goose.SetBaseFS(dbmigrations.Migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		fatal(ctx, "failed to set goose dialect", err)
	}

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	if err := goose.Up(db, "migrations"); err != nil {
		fatal(ctx, "failed to run migrations", err, observability.Integration(observability.IntegrationRDS))
	}

	slog.InfoContext(ctx, "migrations applied successfully")
}

func fatal(ctx context.Context, message string, err error, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, slog.LevelError, message, append(attrs, observability.Err(err))...)
	os.Exit(1)
}
