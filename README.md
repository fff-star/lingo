# Lingo

Personal language material manager — vocabulary, phrases, sentences, articles, compositions with FSRS spaced repetition. CLI-first, web-assisted, SQLite storage.

## Features

- **5 material types**: words, phrases, sentences, articles, compositions — independent storage, unified tag system
- **FSRS-5 spaced repetition**: same algorithm as Anki, 4-level rating, adaptive intervals for words and phrases
- **Dictionary lookup**: Merriam-Webster Collegiate Dictionary API for English definitions, phonetics, audio pronunciation, inflections; ECDICT SQLite database for offline Chinese translations (~770k entries)
- **LLM integration**: DeepSeek for article content extraction, composition analysis (grammar errors + model essay), word inflection suggestions
- **Built-in fuzzy search**: CLI interactive picker with real-time filtering; web client-side fuzzy filter with instant feedback
- **Review streak tracking**: daily consecutive review day counter with streak history
- **Per-tag progress stats**: mastery %, reviewed count, due count broken down by tag
- **Article inline word lookup**: click any word in an article to see its dictionary definition; one-click add to vocabulary
- **Audio pronunciation**: play button on word detail and review cards (from Merriam-Webster API)
- **Batch tag management**: bulk assign/unassign tags via CLI (`lingo tag assign`) or web UI
- **Dark mode** in web UI, persisted to localStorage
- **Single binary**: compiles to one static file, all assets embedded, fully offline-ready
- **Cross-platform**: Linux, Windows, macOS — same binary, same experience

## Quick Start

```bash
go build -o lingo .
```

### Environment

```bash
# Required for LLM features (article processing, composition analysis, inflection suggestions)
export DEEPSEEK_API_KEY="sk-..."

# Optional
export DEEPSEEK_BASE_URL="https://api.deepseek.com"   # default
export DEEPSEEK_MODEL="deepseek-chat"                  # default
export ECDICT_DB_PATH="/path/to/ecdict.db"              # English-Chinese dictionary (download separately)
export MW_API_KEY="..."                                 # Merriam-Webster Collegiate API key
```

Dictionary lookup and all other features work without any API keys.

### Add a word (automatic dictionary lookup)

```bash
./lingo word add ephemeral
```

### Add a phrase

```bash
./lingo phrase add "in the long run"
```

### Add a sentence

```bash
./lingo sent add                        # interactive
echo "..." | ./lingo sent add -         # piped
```

### Add an article / composition

```bash
./lingo article add                     # opens $EDITOR
./lingo article process <id-prefix>     # LLM extracts words/phrases/sentences
./lingo comp add                        # student essay
./lingo comp process <id-prefix>        # LLM analysis + model essay + grammar errors
```

### Search (built-in fuzzy, no fzf required)

```bash
./lingo search                          # interactive fuzzy picker (all types)
./lingo search --type word              # fuzzy picker, words only
./lingo search ubiquitous               # keyword search across all types
```

In the interactive picker: type to filter in real-time, ↑↓ to navigate, Enter to select and view full detail, Esc to quit.

### Review (FSRS spaced repetition)

```bash
./lingo review                          # stats: due, streak, today's progress
./lingo review start                    # interactive session
./lingo review start --tag gre --limit 10
```

### Tags

```bash
./lingo tag add gre --color "#ff6b6b"
./lingo tag rename gre GRE
./lingo tag rm GRE

# Batch operations
./lingo tag assign gre --type word --all
./lingo tag assign gre --type word --ids wd_abc123,wd_def456
./lingo tag unassign gre --type word --all --filter-tag oldtag
```

### Export

```bash
./lingo export --type word,phrase --format csv
./lingo list --type sent --tag argument
```

### Web

```bash
./lingo web
# → http://localhost:8080
```

Features: browse all types, fuzzy search, tag filtering, review cards with keyboard shortcuts (1-4), dark mode toggle, batch tag operations, per-tag progress stats, streak tracking, inline word lookup in articles.

### Data check

```bash
./lingo check                           # data integrity: orphan tags, duplicates, empty fields, cross-type consistency
```

## Data

Stored as SQLite at `$XDG_DATA_HOME/lingo/lingo.db` (Linux) or `%APPDATA%/lingo/lingo.db` (Windows):

| Platform | Default path |
|----------|-------------|
| Linux | `~/.local/share/lingo/lingo.db` |
| Windows | `%APPDATA%\lingo\lingo.db` |
| macOS | `~/.local/share/lingo/lingo.db` |

Set `LINGO_DATA_DIR` to override. The database uses WAL mode with a single writer for safety.

## FSRS Review

Based on the Free Spaced Repetition Scheduler (FSRS-5), the default algorithm in Anki. Supports words and phrases.

| Rating | Key | Meaning |
|--------|-----|---------|
| 1 | Again | forgot completely |
| 2 | Hard | recalled with difficulty |
| 3 | Good | recalled correctly |
| 4 | Easy | recalled easily |

The algorithm adapts stability and difficulty per card, scheduling reviews at optimal intervals. Review cards are due at midnight UTC.

## Tech

Go standard library mostly — `net/http`, `html/template`. SQLite via `modernc.org/sqlite` (pure Go, no CGO). Terminal fuzzy picker uses `golang.org/x/term`. Frontend uses Pico.css, htmx 2.0, and marked.js — all local assets, fully offline. Dual dictionary: Merriam-Webster API (English) + ECDICT SQLite (English-Chinese, offline). LLM integration via DeepSeek API (OpenAI-compatible).

## Build matrix

```bash
# Linux
go build -o lingo .

# Windows (x86_64)
GOOS=windows GOARCH=amd64 go build -o lingo.exe .

# Windows (x86 / 32-bit)
GOOS=windows GOARCH=386 go build -o lingo.exe .

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o lingo .

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o lingo .
```

All platforms produce a single static binary. No runtime, no DLLs, no installers needed.
