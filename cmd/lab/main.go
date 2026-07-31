package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"aisaaslab/internal/kernel"
	"aisaaslab/internal/modules/auth"
	"aisaaslab/internal/modules/billing"
	"aisaaslab/internal/modules/completion"
)

func main() {
	// --- 1. Config: layered load (defaults -> config.env -> real env vars) ---
	cfg := kernel.LoadConfig("config.env")
	app := kernel.NewApp(cfg)

	// --- 2. Encoders: register before anything that needs to look one up ---
	kernel.RegisterDefaultEncoders(app)

	// --- 3. Modules: each owns one feature slice, none import each other ---
	app.RegisterModule(auth.New())
	app.RegisterModule(completion.New())
	app.RegisterModule(billing.New())

	// --- 4. Migrations: idempotent, ordered seed steps ---
	migrator := kernel.NewMigrator(filepath.Join(cfg.DataDir, "schema_version.txt"))
	migrator.Register(kernel.Migration{
		Version: 1,
		Name:    "seed demo api keys",
		Up: func(ctx context.Context) error {
			app.Store.SeedAPIKey("demo-key-starter", "starter")
			app.Store.SeedAPIKey("demo-key-pro", "pro")
			app.Store.SetSubscriptionState("demo-key-starter", "trial")
			app.Store.SetSubscriptionState("demo-key-pro", "trial")
			return nil
		},
	})
	if err := migrator.Run(context.Background()); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// --- 5. Init every module (they register messages/handlers/policies/routes here) ---
	if err := app.InitAll(); err != nil {
		log.Fatalf("init failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := app.StartAll(ctx); err != nil {
		log.Fatalf("start failed: %v", err)
	}

	// --- 6. The shared HTTP server every HTTP-facing module registered routes into ---
	srv := &http.Server{Addr: ":" + cfg.HTTPPort, Handler: app.Mux}
	go func() {
		log.Printf("[lab] listening on :%s (quota=%d tokens/day)\n", cfg.HTTPPort, cfg.DailyTokenQuota)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	// --- 7. Graceful shutdown ---
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("[lab] shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)

	cancel()
	app.StopAll(context.Background())
}
