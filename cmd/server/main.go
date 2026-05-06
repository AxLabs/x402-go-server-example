// Package main is the entry point for the x402-go-server-example application.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/AxLabs/x402-go-server-example/internal/config"
	"github.com/AxLabs/x402-go-server-example/internal/httpapi"
	"github.com/AxLabs/x402-go-server-example/internal/logging"
	"github.com/AxLabs/x402-go-server-example/internal/version"
	"github.com/AxLabs/x402-go-server-example/internal/x402"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	logger := logging.NewLogger(cfg.LogLevel)

	vInfo := version.Info()
	logger.Info("starting x402-go-server-example",
		"version", vInfo.Version,
		"commit", vInfo.Commit,
		"build_time", vInfo.BuildTime,
		"facilitator_url", cfg.Facilitator.BaseURL,
		"payment_config_file", cfg.PaymentConfigFile,
		"paid_routes", len(cfg.Payment.Routes),
	)

	// Build the SDK-backed x402 middleware. This constructs the SDK's
	// HTTPFacilitatorClient, x402HTTPResourceServer with the EVM "exact"
	// scheme server, and a net/http middleware that performs the full
	// verify/settle protocol per request.
	mw, err := x402.Middleware(x402.Config{
		FacilitatorURL:         cfg.Facilitator.BaseURL,
		FacilitatorTimeout:     cfg.Facilitator.Timeout,
		Routes:                 httpapi.PaidRoutes(cfg),
		SyncFacilitatorOnStart: true,
		Timeout:                cfg.Server.RequestTimeout,
	})
	if err != nil {
		logger.Error("failed to build x402 middleware", "error", err)
		os.Exit(1)
	}

	router := httpapi.NewRouter(httpapi.RouterConfig{
		Config:         cfg,
		Logger:         logger,
		X402Middleware: mw,
	})

	server := httpapi.NewServer(
		cfg.Server.Addr,
		router,
		cfg.Server.ReadTimeout,
		cfg.Server.WriteTimeout,
		logger,
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		logger.Error("server error", "error", err)
		os.Exit(1)
	case sig := <-quit:
		logger.Info("received shutdown signal", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped gracefully")
}
