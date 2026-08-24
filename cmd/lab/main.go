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
	"aisaaslab/internal/modules/payment"
	"aisaaslab/internal/modules/subscription"
	"aisaaslab/internal/modules/tenant"
)

func main() {
	// --- 1. Config: layered load (defaults -> config.env -> real env vars) ---
	cfg := kernel.LoadConfig("config.env")
	app := kernel.NewApp(cfg)

	// --- 2. Encoders: register before anything that needs to look one up ---
	kernel.RegisterDefaultEncoders(app)

	// --- 3. Modules: each owns one feature slice, none import each other ---
	authMod := auth.New()
	completionMod := completion.New()
	subMod := subscription.New()
	billingMod := billing.New()
	tenantMod := tenant.New(authMod.Service())
	payMod := payment.New(subMod.Manager())

	app.RegisterModule(authMod)
	app.RegisterModule(completionMod)
	app.RegisterModule(subMod)
	app.RegisterModule(billingMod)
	app.RegisterModule(tenantMod)
	app.RegisterModule(payMod)

	// --- 4. Migrations: idempotent, ordered seed steps ---
	migrator := kernel.NewMigrator(filepath.Join(cfg.DataDir, "schema_version.txt"))
	migrator.Register(kernel.Migration{
		Version: 1,
		Name:    "seed schema marker",
		Up: func(ctx context.Context) error {
			return nil
		},
	})
	if err := migrator.Run(context.Background()); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// Always seed demo keys in memory for lab environment
	_ = authMod.Service().RegisterAPIKey("demo-key-starter", "starter")
	_ = authMod.Service().RegisterAPIKey("demo-key-pro", "pro")
	_ = authMod.Service().RegisterAPIKey("sk_lab_pro_8f92a1c4e7b3091d", "pro")
	_ = authMod.Service().RegisterAPIKey("sk_lab_free_3c7d91e8b2a5", "free")
	subMod.Manager().CreateContract(subscription.Contract{TenantKey: "demo-key-starter", PlanID: "starter", State: subscription.StateTrial})
	subMod.Manager().CreateContract(subscription.Contract{TenantKey: "demo-key-pro", PlanID: "pro", State: subscription.StateTrial})
	subMod.Manager().CreateContract(subscription.Contract{TenantKey: "sk_lab_pro_8f92a1c4e7b3091d", PlanID: "pro", State: subscription.StateTrial})
	subMod.Manager().CreateContract(subscription.Contract{TenantKey: "sk_lab_free_3c7d91e8b2a5", PlanID: "free", State: subscription.StateTrial})

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
	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	srv := &http.Server{Addr: ":" + cfg.HTTPPort, Handler: corsHandler(app.Mux)}
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
