package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/3x-ui-exporter/client"
	"github.com/3x-ui-exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	baseURL := os.Getenv("BASE_URL")
	token := os.Getenv("TOKEN")
	port := envInt("EXPORTER_PORT", 9847)
	timeoutSec := envInt("SCRAPE_TIMEOUT", 10)

	if baseURL == "" || token == "" {
		logger.Error("BASE_URL and TOKEN environment variables are required")
		os.Exit(1)
	}

	apiClient := client.New(baseURL, token, time.Duration(timeoutSec)*time.Second)
	coll := collector.New(apiClient, logger)

	reg := prometheus.NewRegistry()
	reg.MustRegister(coll)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if coll.IsHealthy() {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "ok")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, "unhealthy")
		}
	})

	addr := fmt.Sprintf(":%d", port)
	logger.Info("starting 3x-ui exporter", "addr", addr, "base_url", baseURL, "timeout", timeoutSec)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}
