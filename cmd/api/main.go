package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/config"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/database"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/observability"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/routes"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/packages/email"
	"github.com/gofiber/fiber/v3"
)

// O binario tem dois modos:
//
//	techchallenge            sobe a API
//	techchallenge migrate    aplica as migrations e sai
//
// As migrations deixaram de rodar no boot porque o Deployment sobe com
// replicas >= 2 e HPA ate 10: N processos disputando o mesmo DDL e corrida.
// O pipeline do repo roda um Job com `migrate` antes do rollout.
func main() {
	// Antes de tudo: uma falha de boot tambem precisa sair estruturada. E o log
	// que aparece quando o pod nao sobe -- se ele for texto livre, e justamente
	// o incidente mais dificil que fica de fora da consulta.
	observability.Setup()

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrations()
		return
	}
	runServer()
}

func runMigrations() {
	cfg := mustLoadConfig()

	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		shutdownApp(err, "failed to connect to database", observability.Integration(observability.IntegrationRDS))
	}
	defer db.Close()

	database.RunMigrations(db)
}

func runServer() {
	cfg := mustLoadConfig()

	// O tracer sobe antes do primeiro span possivel e desce por ultimo, para o
	// buffer ser esvaziado no shutdown. Sem DD_TRACE_ENABLED=true isto e um
	// no-op -- ver observability.StartTracer.
	stopTracer := observability.StartTracer(slog.Default())
	defer stopTracer()

	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		shutdownApp(err, "failed to connect to database", observability.Integration(observability.IntegrationRDS))
	}

	emailProv, err := newEmailProvider(cfg)
	if err != nil {
		shutdownApp(err, "failed to create email provider", observability.Integration(observability.IntegrationSES))
	}

	slog.Info("dependencies initialized successfully")

	app := fiber.New(fiber.Config{})
	routes.RegisterRoutes(app, db, cfg, emailProv)

	if err := app.Listen(":" + cfg.Server.Port); err != nil {
		shutdownApp(err, "failed to start server")
	}
}

func mustLoadConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		shutdownApp(err, "failed to load configuration")
	}
	return cfg
}

func newEmailProvider(cfg *config.Config) (email.Provider, error) {
	return email.New(cfg.Email.Provider, email.Config{
		Host:           cfg.Email.Host,
		Port:           cfg.Email.Port,
		From:           cfg.Email.From,
		Region:         cfg.AWS.DefaultRegion,
		SESSenderEmail: cfg.AWS.SESSenderEmail,
		SESReplyTo:     cfg.AWS.SESReplyTo,
		SESConfigSet:   cfg.AWS.SESConfigSet,
	})
}

// shutdownApp registra e derruba o processo. A mensagem sai como linha
// estruturada, e nao em fmt.Println: um pod em CrashLoopBackOff e um dos poucos
// casos em que o log e a UNICA evidencia, e ele precisa ser consultavel como o
// resto.
func shutdownApp(err error, message string, attrs ...slog.Attr) {
	if err == nil {
		return
	}

	slog.LogAttrs(context.Background(), slog.LevelError, message, append(attrs, observability.Err(err))...)
	os.Exit(1)
}
