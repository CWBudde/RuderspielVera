// Command server runs the Ruderspiel HTTP API and (optionally) serves the built SPA.
//
// Embedding web/dist into the binary is a later step; for now the SPA is served from -web.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"ruderspiel/internal/api"
)

// config is the parsed command line.
type config struct {
	addr   string
	webDir string
	opts   api.Options
}

// parseFlags reads the command line: listen address, SPA directory and the API options.
func parseFlags() config {
	var c config

	flag.StringVar(&c.addr, "addr", ":8080", "listen address")
	flag.StringVar(&c.opts.DataDir, "data", "./data/games", "directory for persisted games (empty = in-memory only)")
	flag.StringVar(&c.webDir, "web", "web/dist", "directory of the built SPA (served if it exists)")
	flag.DurationVar(&c.opts.TurnTimeout, "turn-timeout", 10*time.Minute,
		"multiplayer: a bot decides for a human who keeps the game waiting this long (0 = off)")
	flag.DurationVar(&c.opts.LobbyMaxAge, "lobby-max-age", 24*time.Hour,
		"drop lobbies not started within this age (0 = off)")
	flag.Float64Var(&c.opts.CreateRate, "create-rate", 10,
		"per-IP limit for creating and joining games, requests per minute (0 = off)")
	flag.IntVar(&c.opts.CreateBurst, "create-burst", 10, "burst size for -create-rate")

	flag.Parse()

	return c
}

func main() {
	cfg := parseFlags()
	addr, webDir, opts := cfg.addr, cfg.webDir, cfg.opts

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(log)

	opts.Decider, opts.Logger = botDecider, log

	srv, err := api.New(opts)
	if err != nil {
		log.Error("startup failed", "err", err)
		os.Exit(1)
	}

	var static http.Handler

	st, err := os.Stat(filepath.Join(webDir, "index.html"))
	if err == nil && !st.IsDir() {
		static = spaHandler(webDir)
		log.Info("serving SPA", "dir", webDir)
	} else {
		log.Info("no SPA build found, API only", "dir", webDir)
	}

	hs := newHTTPServer(addr, srv.Handler(static))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("listening", "addr", addr)

		err := hs.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = hs.Shutdown(sctx)
	if err != nil {
		log.Error("shutdown failed", "err", err)
	}

	srv.Close()
}

// newHTTPServer configures the server. Requests derive their context from a base context that is
// cancelled on shutdown: that ends the long-lived event streams (GET /api/games/{id}/events), which
// Shutdown would otherwise wait for. The stream handler clears the read/write timeouts for its
// own connection.
func newHTTPServer(addr string, h http.Handler) *http.Server {
	baseCtx, cancelBase := context.WithCancel(context.Background())

	hs := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return baseCtx },
	}
	hs.RegisterOnShutdown(cancelBase)

	return hs
}

// spaHandler serves files from dir and falls back to index.html for unknown paths (client routing).
func spaHandler(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Clean("/" + r.URL.Path)
		if p != "/" && !strings.HasSuffix(p, "/") {
			// p is cleaned and rooted, so it cannot leave dir.
			st, err := os.Stat(filepath.Join(dir, filepath.FromSlash(p))) //nolint:gosec // see above
			if err == nil && !st.IsDir() {
				files.ServeHTTP(w, r)
				return
			}
		}

		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}
