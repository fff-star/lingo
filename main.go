package main

import (
	"embed"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"lingo/cli"
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
	if home, _ := os.UserHomeDir(); home != "" {
		return filepath.Join(home, ".local", "share", "lingo")
	}
	return "data"
}

func main() {
	dataDir := dataDirFromEnv()

	ws := store.NewWordStore(filepath.Join(dataDir, "words.json"))
	ps := store.NewPhraseStore(filepath.Join(dataDir, "phrases.json"))
	ss := store.NewSentenceStore(filepath.Join(dataDir, "sentences.json"))
	as := store.NewArticleStore(filepath.Join(dataDir, "articles.json"))
	cs := store.NewCompositionStore(filepath.Join(dataDir, "compositions.json"))
	ts := store.NewTagStore(filepath.Join(dataDir, "tags.json"))
	rl := store.NewReviewLog(filepath.Join(dataDir, "review_log.json"))

	cli.InitWord(ws)
	cli.InitPhrase(ps)
	cli.InitSentence(ss)
	cli.InitArticle(as)
	cli.InitComp(cs)
	cli.InitTag(ts)

	// Set up web server starter.
	webServer := func(port string) {
		srv, err := web.New(ws, ps, ss, as, cs, ts, rl, templateFiles)
		if err != nil {
			fmt.Fprintf(os.Stderr, "web: %v\n", err)
			os.Exit(1)
		}
		srv.MustLoad()

		mux := http.NewServeMux()
		srv.Register(mux, staticFiles)

		fmt.Fprintf(os.Stderr, "Listening on :%s\n", port)
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			fmt.Fprintf(os.Stderr, "web: %v\n", err)
			os.Exit(1)
		}
	}
	cli.InitWeb(webServer)

	cli.Run(cli.Stores{})
}
