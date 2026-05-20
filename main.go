package main

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"lingo/cli"
	"lingo/dict"
	"lingo/store"
	"lingo/web"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed web/templates/*
var templateFiles embed.FS

// dataDirFromEnv resolves the data directory following XDG Base Directory:
//  1. LINGO_DATA_DIR or SD_DATA_DIR (explicit override)
//  2. $XDG_DATA_HOME/lingo
//  3. $HOME/.local/share/lingo (XDG default)
//  4. ./data (last resort)
func dataDirFromEnv() string {
	if d := os.Getenv("LINGO_DATA_DIR"); d != "" {
		return d
	}
	if d := os.Getenv("SD_DATA_DIR"); d != "" {
		return d
	}
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "lingo")
	}
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "lingo")
		}
	}
	if home, _ := os.UserHomeDir(); home != "" {
		return filepath.Join(home, ".local", "share", "lingo")
	}
	return "data"
}

func main() {
	os.Exit(run())
}

func run() int {
	dataDir := dataDirFromEnv()

	db, err := store.OpenDB(filepath.Join(dataDir, "lingo.db"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "db: %v\n", err)
		return 1
	}
	defer store.CloseDB(db)

	// Initialize ECDICT (optional English-Chinese dictionary).
	ecdictPath := os.Getenv("ECDICT_DB_PATH")
	if ecdictPath == "" {
		ecdictPath = filepath.Join(dataDir, "ecdict.db")
	}
	if err := dict.InitECDICT(ecdictPath); err != nil {
		fmt.Fprintf(os.Stderr, "ecdict: %v (Chinese definitions unavailable)\n", err)
	}
	defer dict.CloseECDICT()

	ws := store.NewWordStore(db)
	ps := store.NewPhraseStore(db)
	ss := store.NewSentenceStore(db)
	as := store.NewArticleStore(db)
	cs := store.NewCompositionStore(db)
	ts := store.NewTagStore(db)
	rl := store.NewReviewLog(db)

	cli.InitWord(ws)
	cli.InitPhrase(ps)
	cli.InitSentence(ss)
	cli.InitArticle(as)
	cli.InitComp(cs)
	cli.InitTag(ts)

	// Set up web server starter with graceful shutdown.
	webServer := func(port string) {
		srv, err := web.New(ws, ps, ss, as, cs, ts, rl, templateFiles)
		if err != nil {
			fmt.Fprintf(os.Stderr, "web: %v\n", err)
			return
		}
		srv.MustLoad()

		mux := http.NewServeMux()
		srv.Register(mux, staticFiles)

		httpServer := &http.Server{Addr: ":" + port, Handler: mux}

		serverErr := make(chan error, 1)
		go func() {
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				serverErr <- err
			}
		}()

		fmt.Fprintf(os.Stderr, "Listening on :%s\n", port)

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		select {
		case <-sigCh:
			fmt.Fprintf(os.Stderr, "\nShutting down...\n")
		case err := <-serverErr:
			fmt.Fprintf(os.Stderr, "web: %v\n", err)
		}

		signal.Stop(sigCh)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "shutdown: %v\n", err)
		}
	}
	cli.InitWeb(webServer)

	if err := cli.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}
