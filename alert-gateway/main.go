package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"alert-gateway/internal/handler"
	"alert-gateway/internal/router"
)

func main() {
	// .env is optional; if missing, the real system environment variables are used
	// (e.g. when set in Docker via env_file or environment).
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, using system environment variables")
	}

	configPath := mustEnv("CONFIG_PATH")
	listenAddr := getEnvOrDefault("LISTEN_ADDR", ":8080")

	cfg, err := router.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("loading config failed: %v", err)
	}

	rt := router.New(cfg)
	h := handler.New(rt)

	mux := http.NewServeMux()
	mux.Handle("/alert", h)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	log.Printf("alert-gateway listening on %s (config: %s)", listenAddr, configPath)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("required environment variable %q is not set", key)
	}
	return val
}

func getEnvOrDefault(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}
