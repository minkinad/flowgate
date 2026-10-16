package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/minkinad/flowgate/internal/config"
	"github.com/minkinad/flowgate/internal/gateway"
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) == 1 && args[0] == "healthcheck" {
		c := &http.Client{Timeout: time.Second}
		resp, err := c.Get("http://127.0.0.1:8080/health/live")
		if err != nil {
			return 1
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return 1
		}
		return 0
	}
	if len(args) == 0 || (args[0] != "serve" && args[0] != "validate") {
		fmt.Fprintln(os.Stderr, "usage: flowgate serve|validate --config path")
		return 2
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	path := fs.String("config", "configs/flowgate.yaml", "YAML configuration")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	c, err := config.Load(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if args[0] == "validate" {
		fmt.Println("configuration valid")
		return 0
	}
	var handler slog.Handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: parseLevel(c.Server.LogLevel)})
	if c.Server.LogFormat == "text" {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: parseLevel(c.Server.LogLevel)})
	}
	logger := slog.New(handler)
	srv := &http.Server{Addr: c.Server.Listen, Handler: gateway.New(c, logger).Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	logger.Info("listening", "address", c.Server.Listen)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server failed", "error", err)
		return 1
	}
	return 0
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
