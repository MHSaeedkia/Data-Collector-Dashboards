package main

import (
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/joho/godotenv"

	"alert-gateway/internal/handler"
	"alert-gateway/internal/router"
)

func main() {
	// .env is optional; if missing, the real system environment variables are used
	// (e.g. when set in Docker via env_file or environment).
	// The error is kept and logged after the logger is set up, so that LOG_LEVEL
	// coming from .env is already in effect.
	dotenvErr := godotenv.Load()

	setupLogger(getEnvOrDefault("LOG_LEVEL", "info"))

	if dotenvErr != nil {
		slog.Info(".env file not found, using system environment variables")
	}

	configPath := mustEnv("CONFIG_PATH")
	listenAddr := getEnvOrDefault("LISTEN_ADDR", ":8080")

	cfg, err := router.LoadConfig(configPath)
	if err != nil {
		slog.Error("loading config failed", "path", configPath, "error", err)
		os.Exit(1)
	}
	logConfig(configPath, cfg)

	rt := router.New(cfg)
	h := handler.New(rt)

	mux := http.NewServeMux()
	mux.Handle("/alert", h)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		slog.Debug("health check", "remote_addr", r.RemoteAddr)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	// Catch-all: makes a misconfigured Grafana webhook URL (wrong path) visible
	// instead of silently 404ing.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		slog.Warn("request to unknown path",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent())
		http.NotFound(w, r)
	})

	slog.Info("alert-gateway listening", "addr", listenAddr, "config", configPath)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// setupLogger installs a text slog handler on stdout at the given level.
// Recognised levels: debug, info, warn, error. Anything else falls back to info.
func setupLogger(level string) {
	var lvl slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})))
}

// logConfig prints a summary of what was loaded, so a routing surprise can be
// diagnosed from the startup lines alone without reading the mounted config.
func logConfig(path string, cfg *router.Config) {
	channels := make([]string, 0, len(cfg.Channels))
	for name, cc := range cfg.Channels {
		channels = append(channels, name+"("+cc.Type+")")
	}
	sort.Strings(channels)

	slog.Info("config loaded",
		"path", path,
		"routes", len(cfg.Routes),
		"channels", strings.Join(channels, ","),
		"default_channels", strings.Join(cfg.DefaultChannels, ","))

	for _, rule := range cfg.Routes {
		match := make([]string, 0, len(rule.Match))
		for k, v := range rule.Match {
			match = append(match, k+"="+v)
		}
		sort.Strings(match)
		slog.Debug("route rule",
			"name", rule.Name,
			"match", strings.Join(match, ","),
			"channels", strings.Join(rule.Channels, ","))
	}
}

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		slog.Error("required environment variable is not set", "key", key)
		os.Exit(1)
	}
	return val
}

func getEnvOrDefault(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}
