package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/zy84338719/dgx-spark-exporter/internal/config"
	"github.com/zy84338719/dgx-spark-exporter/internal/logger"
	"github.com/zy84338719/dgx-spark-exporter/pkg/collectors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	Version   = "dev"
	Revision  = "unknown"
	Branch    = "unknown"
	BuildTime = "unknown"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	log.Info("starting DGX Spark Exporter",
		"version", Version,
		"revision", Revision,
		"branch", Branch,
		"build_time", BuildTime,
		"config", cfg.String(),
	)

	hostname, err := os.Hostname()
	if err != nil {
		log.Error("failed to get hostname", "error", err)
		os.Exit(1)
	}

	registry := prometheus.WrapRegistererWith(
		prometheus.Labels{"host": hostname},
		prometheus.DefaultRegisterer,
	)

	registerCollectors(cfg, log, registry)
	registerBuildInfo(registry)

	mux := http.NewServeMux()
	setupHandlers(mux, cfg)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("server listening", "address", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		log.Error("server error", "error", err)
		os.Exit(1)
	case sig := <-shutdown:
		log.Info("shutdown signal received", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	log.Info("server stopped gracefully")
}

func registerBuildInfo(registry prometheus.Registerer) {
	buildInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: collectors.Namespace,
			Name:      "exporter_build_info",
			Help:      "Build information of the exporter",
		},
		[]string{"version", "revision", "branch", "go_version"},
	)
	buildInfo.WithLabelValues(Version, Revision, Branch, runtime.Version()).Set(1)
	registry.MustRegister(buildInfo)
}

func registerCollectors(cfg *config.Config, log *slog.Logger, registry prometheus.Registerer) {
	if cfg.IsCollectorEnabled("cpu") {
		registry.MustRegister(collectors.NewCPUCollector(cfg, log))
	}
	if cfg.IsCollectorEnabled("gpu") {
		registry.MustRegister(collectors.NewGPUCollector(cfg, log))
	}
	if cfg.IsCollectorEnabled("memory") {
		registry.MustRegister(collectors.NewMemoryCollector(cfg, log))
	}
	if cfg.IsCollectorEnabled("disk") {
		registry.MustRegister(collectors.NewDiskCollector(cfg, log))
	}
	if cfg.IsCollectorEnabled("network") {
		registry.MustRegister(collectors.NewNetworkCollector(cfg, log))
	}
}

func setupHandlers(mux *http.ServeMux, cfg *config.Config) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><title>DGX Spark Exporter</title></head>
<body>
<h1>DGX Spark Exporter</h1>
<p><a href="`+cfg.MetricsPath+`">Metrics</a></p>
<p><a href="/health">Health</a></p>
<p><a href="/ready">Ready</a></p>
<p><a href="/version">Version</a></p>
</body>
</html>`)
	})

	mux.Handle(cfg.MetricsPath, promhttp.Handler())

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Ready")
	})

	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"version":    Version,
			"revision":   Revision,
			"branch":     Branch,
			"build_time": BuildTime,
			"go_version": runtime.Version(),
		})
	})
}
