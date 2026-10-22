package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"filetransfer.ir/api/internal/auth"
	"filetransfer.ir/api/internal/files"
	"filetransfer.ir/api/internal/storage"
	"filetransfer.ir/api/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type server struct {
	maxFileSize int64
}

func main() {
	maxFileSize := int64(50 * 1024 * 1024 * 1024)
	if value := os.Getenv("MAX_FILE_SIZE_BYTES"); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			maxFileSize = parsed
		}
	}

	s := &server{maxFileSize: maxFileSize}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /api/v1/config", s.config)
	authH, pool := registerAuth(mux)
	registerFiles(mux, authH, pool, maxFileSize)

	address := ":" + getenv("API_PORT", "8080")
	log.Printf("filetransfer API listening on %s", address)
	if err := http.ListenAndServe(address, logging(mux)); err != nil {
		log.Fatal(err)
	}
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "filetransfer-api",
	})
}

func (s *server) config(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"maxFileSizeBytes": s.maxFileSize,
		"maxFileSizeGB":    50,
		"uploadProtocol":   "tus",
		"locale":           "fa-IR",
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(started))
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// registerAuth wires OTP login when Postgres and Redis are configured. Without
// them the API still serves /health and /api/v1/config.
func registerAuth(mux *http.ServeMux) (*auth.Handlers, *pgxpool.Pool) {
	dbURL, redisURL := os.Getenv("DATABASE_URL"), os.Getenv("REDIS_URL")
	if dbURL == "" || redisURL == "" {
		log.Print("auth disabled: DATABASE_URL and REDIS_URL are required")
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	if _, err := pool.Exec(ctx, migrations.Users); err != nil {
		log.Fatalf("migrate users: %v", err)
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("redis url: %v", err)
	}
	rdb := redis.NewClient(opts)

	development := getenv("APP_ENV", "development") == "development"
	var sender auth.Sender
	switch {
	case os.Getenv("KAVENEGAR_API_KEY") != "":
		sender = auth.Kavenegar{APIKey: os.Getenv("KAVENEGAR_API_KEY"), Template: os.Getenv("KAVENEGAR_VERIFY_TEMPLATE")}
	case development:
		log.Print("KAVENEGAR_API_KEY unset: OTP codes will be logged (development only)")
		sender = auth.LogSender{}
	default:
		log.Fatal("KAVENEGAR_API_KEY is required outside development")
	}

	secret := os.Getenv("OTP_SECRET")
	if secret == "" {
		if !development {
			log.Fatal("OTP_SECRET is required outside development")
		}
		secret = "dev-only-otp-secret"
	}

	h := &auth.Handlers{
		OTP:          auth.NewOTP(rdb, sender, []byte(secret)),
		Sessions:     auth.NewSessions(rdb),
		Users:        auth.PGUsers{Pool: pool},
		SecureCookie: !development,
		TrustProxy:   os.Getenv("TRUST_PROXY") == "1",
	}
	h.Register(mux)
	return h, pool
}

// registerFiles wires uploads and shares when object storage is configured.
func registerFiles(mux *http.ServeMux, authH *auth.Handlers, pool *pgxpool.Pool, maxFileSize int64) {
	endpoint, bucket := os.Getenv("OBJECT_STORAGE_ENDPOINT"), os.Getenv("OBJECT_STORAGE_BUCKET")
	if authH == nil || endpoint == "" || bucket == "" {
		log.Print("files disabled: needs auth plus OBJECT_STORAGE_ENDPOINT and OBJECT_STORAGE_BUCKET")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := pool.Exec(ctx, migrations.Files); err != nil {
		log.Fatalf("migrate files: %v", err)
	}

	store := storage.New(storage.Config{
		Endpoint:       endpoint,
		PublicEndpoint: os.Getenv("OBJECT_STORAGE_PUBLIC_ENDPOINT"),
		Region:         os.Getenv("OBJECT_STORAGE_REGION"),
		Bucket:         bucket,
		AccessKey:      os.Getenv("OBJECT_STORAGE_ACCESS_KEY"),
		SecretKey:      os.Getenv("OBJECT_STORAGE_SECRET_KEY"),
		PathStyle:      getenv("OBJECT_STORAGE_PATH_STYLE", "1") == "1",
	})
	if getenv("APP_ENV", "development") == "development" {
		if err := store.EnsureBucket(ctx); err != nil {
			log.Fatalf("ensure bucket: %v", err)
		}
	}

	repo := files.PG{Pool: pool}
	files.NewHandlers(repo, store, authH, maxFileSize, getenv("PUBLIC_BASE_URL", "http://localhost:3000")).Register(mux)
	go files.RunJanitor(context.Background(), repo, store, 10*time.Minute, 24*time.Hour)
}
