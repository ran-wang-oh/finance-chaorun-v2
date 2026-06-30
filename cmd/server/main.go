package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"finance.chao.run/v2/internal/api"
	"finance.chao.run/v2/internal/config"
	"finance.chao.run/v2/internal/service"
	"finance.chao.run/v2/internal/store/postgres"
)

func main() {
	ctx := context.Background()
	cfg := config.Default()

	if err := cfg.Validate(); err != nil {
		slog.Error("invalid config", "error", err)
		os.Exit(1)
	}

	// Connect to PostgreSQL and run migrations
	db, err := postgres.Connect(ctx, cfg.Database.DSN)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.RunMigrations(ctx); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations complete")

	// Wire up service with all stores
	svc := service.New(db).
		WithBooks(postgres.NewBookStore(db)).
		WithPeriods(postgres.NewPeriodStore(db)).
		WithInvoices(postgres.NewInvoiceStore(db)).
		WithInvoiceLines(postgres.NewInvoiceLineStore(db)).
		WithAccounts(postgres.NewAccountStore(db)).
		WithJournals(postgres.NewJournalStore(db)).
		WithAuditLog(postgres.NewAuditStore(db)).
		WithIdempotency(postgres.NewIdempotencyStore(db)).
		WithReconciliation(postgres.NewReconciliationStore(db)).
		WithReportMappings(postgres.NewReportMappingStore(db)).
		WithAdjustments(postgres.NewAdjustmentStore(db)).
		WithTaxProfiles(postgres.NewTaxProfileStore(db)).
		WithTaxReturns(postgres.NewTaxReturnStore(db)).
		WithRiskScans(postgres.NewRiskScanStore(db)).
		WithConsistencyChecks(postgres.NewConsistencyCheckStore(db))

	// Build HTTP server with idempotency support
	idempStore := postgres.NewIdempotencyStore(db)
	handler := api.RoutesWithIdempotency(svc, idempStore)

	srv := &http.Server{
		Addr:         cfg.Server.HTTPAddr,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		slog.Info("finance provider starting", "addr", cfg.Server.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		slog.Error("server error", "error", err)
		os.Exit(1)
	case s := <-sig:
		slog.Info("shutting down", "signal", s.String())
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}

func init() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	fmt.Println("init logger")
}
