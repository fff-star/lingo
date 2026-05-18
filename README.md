# Lingo

Personal language material manager — vocabulary, phrases, sentences, articles, compositions with FSRS spaced repetition. CLI-first, web-assisted, JSON file storage.

## Features

- **5 types**: words, phrases, sentences, articles, compositions — independent storage, unified tags
- **FSRS-5 review**: same algorithm as Anki, 4-level rating, adaptive intervals
- **Dictionary lookup**: Free Dictionary API for definitions, phonetics, examples
- **LLM integration**: DeepSeek for article processing, composition analysis, word inflection suggestions
- **CLI with fzf**: fuzzy search, interactive add/edit/delete/review
- **Web UI**: browse, search (htmx), export, review cards (Pico.css, local assets, offline-ready)

## Quick Start

```bash
go build -o lingo .
```

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
./lingo comp add                        # student essay with AI analysis
./lingo comp process <id-prefix>        # LLM scoring + feedback
```

### Search

```bash
./lingo search ubiquitous           # all types
./lingo search --type word gre      # words only
./lingo search --fzf                # interactive fuzzy search
```

### Review (FSRS spaced repetition)

```bash
./lingo review                      # stats: due, reviewed, new
./lingo review start                # interactive session
./lingo review start --tag gre --limit 10
```

### List & filter

```bash
./lingo list                        # all items
./lingo list --type word --tag gre  # words with tag
```

### Tags

```bash
./lingo tag add gre
./lingo tag rename gre GRE
./lingo tag rm GRE
```

### Export

```bash
./lingo export --type word,phrase --format csv
```

### Web

```bash
./lingo web
# → http://localhost:8080
```

### Data check

```bash
./lingo check                       # orphan tags, duplicates, empty fields
```

## Data

Stored as JSON in `~/.local/share/lingo/` (XDG default, or `data/` for development):

```
~/.local/share/lingo/
├── words.json
├── phrases.json
├── sentences.json
├── articles.json
├── compositions.json
├── review_log.json
└── tags.json
```

Human-readable, git-trackable. Set `LINGO_DATA_DIR` (or `SD_DATA_DIR` as fallback) to change location.

## FSRS Review

Based on the Free Spaced Repetition Scheduler (FSRS-5), the default algorithm in Anki.

| Rating | Meaning |
|--------|---------|
| 1 | Again — forgot completely |
| 2 | Hard — recalled with difficulty |
| 3 | Good — recalled correctly |
| 4 | Easy — recalled easily |

The algorithm adapts stability and difficulty per card, scheduling reviews at optimal intervals.

## Tech

Go standard library only — `net/http`, `html/template`. Frontend uses Pico.css, htmx, and marked.js (local assets, zero JS written, fully offline). LLM integration via DeepSeek API (OpenAI-compatible) for article processing, composition analysis, and word inflection suggestions.
