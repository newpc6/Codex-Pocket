package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"codexpocket/internal/config"
	"codexpocket/internal/httpapi"
	"codexpocket/internal/runtime"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	agent := runtime.NewAgent(cfg, logger)
	if err := agent.Start(ctx); err != nil {
		logger.Error("failed to start agent", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: httpapi.NewServer(agent, logger, cfg).Handler(),
	}

	go func() {
		<-ctx.Done()
		if err := agent.Stop(); err != nil {
			logger.Warn("failed to stop agent cleanly", "error", err)
		}
		_ = server.Shutdown(context.Background())
	}()

	logger.Info("codexpocket agent listening", "addr", cfg.ListenAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("http server failed", "error", err)
		os.Exit(1)
	}
}
