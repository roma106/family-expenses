package app

import (
	"context"
	"errors"
	"family-expenses/internal/agent"
	"family-expenses/internal/config"
	"family-expenses/internal/db"
	"family-expenses/internal/handler"
	"family-expenses/internal/llm"
	"family-expenses/internal/repository"
	"family-expenses/internal/service"
	"family-expenses/internal/tg"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

type App struct {
	Http   *echo.Echo
	Bot    *tg.Bot
	DB     *sqlx.DB
	Config *config.Config
}

func New() (*App, error) {
	cfg, err := config.ReadConfig()
	if err != nil {
		return nil, err
	}

	database, err := db.ConnectToDB(cfg)
	if err != nil {
		return nil, err
	}

	if err := db.RunMigrations(database); err != nil {
		database.Close()
		return nil, err
	}

	repo := &repository.Repository{DB: database}
	svc := &service.Service{Repo: repo}

	client := llm.NewClient(
		cfg.LLMURL,
		cfg.LLMModel,
		cfg.LLMAPIKey,
	)

	botAgent := &agent.Agent{
		LLM:     client,
		Service: svc,
	}

	bot, err := tg.NewBot(cfg.TgApiKey, botAgent.Process)
	if err != nil {
		database.Close()
		return nil, err
	}

	e := echo.New()

	h := &handler.Handler{Service: svc}
	handler.RegisterRoutes(e, h)

	return &App{
		Http:   e,
		Bot:    bot,
		DB:     database,
		Config: cfg,
	}, nil
}

func (app *App) Run() error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	botDone := make(chan struct{})

	go func() {
		defer close(botDone)
		app.Bot.Start(ctx)
	}()

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- app.Http.Start(":" + app.Config.HttpPort)
	}()

	slog.Info("App started")

	var runErr error

	select {
	case <-ctx.Done():
		slog.Info("Shutdown signal received")

	case runErr = <-serverErr:
		if errors.Is(runErr, http.ErrServerClosed) {
			runErr = nil
		}
		if runErr != nil {
			slog.Error("HTTP server failed", "error", runErr)
		}
	}

	stop()

	err := app.Shutdown(botDone)
	if runErr != nil {
		return runErr
	}

	return err
}

func (app *App) Shutdown(botDone <-chan struct{}) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	slog.Info("Shutting down app")

	shutdownErr := app.Http.Shutdown(ctx)
	if shutdownErr != nil {
		slog.Error("Failed to shutdown HTTP server", "error", shutdownErr)
	}

	select {
	case <-botDone:
		slog.Info("Telegram bot stopped")
	case <-ctx.Done():
		slog.Warn("Telegram bot shutdown timeout")
	}

	dbErr := app.DB.Close()
	if dbErr != nil {
		slog.Error("Failed to close database", "error", dbErr)
	}

	slog.Info("App stopped")

	if shutdownErr != nil {
		return shutdownErr
	}

	return dbErr
}
