package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"easyplan/internal/api"
	"easyplan/internal/store"
	"easyplan/internal/web"
)

func main() {
	var (
		port         = flag.Int("port", envInt("EASYPLAN_PORT", 9092), "port to listen on")
		host         = flag.String("host", envOr("EASYPLAN_HOST", ""), "address to bind (empty = all interfaces)")
		dataDir      = flag.String("data-dir", envOr("EASYPLAN_DATA_DIR", "./data"), "directory for the sqlite database")
		registration = flag.Bool("registration", envBool("EASYPLAN_REGISTRATION", true), "allow new people to register")
		secure       = flag.Bool("secure-cookies", envBool("EASYPLAN_SECURE_COOKIES", false), "always mark session cookies Secure (use behind HTTPS)")
	)
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0o700); err != nil {
		log.Fatalf("create data dir: %v", err)
	}
	db, err := store.Open(filepath.Join(*dataDir, "easyplan.db"))
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() { // drop expired sessions hourly
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			_ = db.PruneSessions(ctx)
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/api/", api.NewServer(db, api.Config{AllowRegistration: *registration, SecureCookies: *secure}))
	mux.Handle("/", spaHandler(web.FS()))

	addr := fmt.Sprintf("%s:%d", *host, *port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("easy plan listening on %s (registration %s)", addr, map[bool]string{true: "open", false: "closed"}[*registration])
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// spaHandler serves the embedded static build, falling back to index.html
// for client-side routes like /calendar/3.
func spaHandler(files fs.FS) http.Handler {
	fileServer := http.FileServerFS(files)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if st, err := fs.Stat(files, p); err != nil || st.IsDir() {
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFileFS(w, r, files, "index.html")
			return
		}
		if strings.HasPrefix(p, "_app/immutable/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		fileServer.ServeHTTP(w, r)
	})
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	if v, err := strconv.ParseBool(os.Getenv(key)); err == nil {
		return v
	}
	return def
}
