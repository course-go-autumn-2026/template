package main

import (
"context"
"errors"
"log"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/go-chi/chi/v5"

"github.com/razrabbesezdav-art/template/internal/config"
"github.com/razrabbesezdav-art/template/internal/database"
)

func main() {
cfg, err := config.Load()
if err != nil {
log.Fatalf("load config: %v", err)
}

ctx, stop := signal.NotifyContext(
context.Background(),
os.Interrupt,
syscall.SIGTERM,
)
defer stop()

pool, err := database.NewPostgres(ctx, cfg)
if err != nil {
log.Fatalf("connect to PostgreSQL: %v", err)
}
defer pool.Close()

router := chi.NewRouter()

router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
_, _ = w.Write([]byte(`{"status":"ok"}`))
})

router.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
pingCtx, cancel := context.WithTimeout(r.Context(), cfg.DatabaseQueryTimeout)
defer cancel()

if err := pool.Ping(pingCtx); err != nil {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusServiceUnavailable)
_, _ = w.Write([]byte(`{"status":"unavailable"}`))
return
}

w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
_, _ = w.Write([]byte(`{"status":"ok"}`))
})

server := &http.Server{
Addr:              cfg.HTTPAddr,
Handler:           router,
ReadTimeout:       10 * time.Second,
ReadHeaderTimeout: 5 * time.Second,
WriteTimeout:      10 * time.Second,
IdleTimeout:       60 * time.Second,
}

serverErr := make(chan error, 1)
go func() {
log.Printf("HTTP server listening on %s", cfg.HTTPAddr)
serverErr <- server.ListenAndServe()
}()

select {
case err := <-serverErr:
if !errors.Is(err, http.ErrServerClosed) {
log.Fatalf("HTTP server: %v", err)
}
case <-ctx.Done():
    log.Println("shutdown signal received")

    shutdownCtx, cancel := context.WithTimeout(
        context.Background(),
        cfg.ShutdownTimeout,
    )

    if err := server.Shutdown(shutdownCtx); err != nil {
        log.Printf("graceful shutdown failed: %v", err)
        _ = server.Close()
    }

    cancel()

    if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
        log.Printf("HTTP server shutdown: %v", err)
    }

    log.Println("server stopped")
}
}
